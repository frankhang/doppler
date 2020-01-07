package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/frankhang/util/logutil"
	"go.uber.org/automaxprocs/maxprocs"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/frankhang/util/config"
	"github.com/frankhang/util/sys/linux"
	"github.com/frankhang/util/tcp"

	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/log"

	"github.com/frankhang/util/metrics"
	"github.com/frankhang/util/signal"
	"github.com/frankhang/util/systimemon"
	"github.com/opentracing/opentracing-go"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
	"github.com/struCoder/pidusage"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/zap"
)

// Flag Names
const (
	nmVersion          = "V"
	nmConfig           = "config"
	nmConfigCheck      = "config-check"
	nmConfigStrict     = "config-strict"
	nmHost             = "host"
	nmPort             = "P"
	nmLogLevel         = "L"
	nmLogFile          = "log-file"
	nmReportStatus     = "report-status"
	nmStatusHost       = "status-host"
	nmStatusPort       = "status"
	nmMetricsAddr      = "metrics-addr"
	nmMetricsInterval  = "metrics-interval"
	nmTokenLimit       = "token-limit"
	nmAffinityCPU                = "affinity-cpus"
)

var (
	version      = flagBoolean(nmVersion, false, "print version information and exit")
	configPath   = flag.String(nmConfig, "", "config file path")
	configCheck  = flagBoolean(nmConfigCheck, false, "check config file validity and exit")
	configStrict = flagBoolean(nmConfigStrict, false, "enforce config file validity")

	// Base

	host             = flag.String(nmHost, "0.0.0.0", "server host")
	port             = flag.String(nmPort, "10001", "server port")
	tokenLimit       = flag.Int(nmTokenLimit, 1000, "the limit of concurrent executed sessions")
	affinityCPU      = flag.String(nmAffinityCPU, "", "affinity cpu (cpu-no. separated by comma, e.g. 1,2,3)")

	// Log
	logLevel     = flag.String(nmLogLevel, "info", "log level: info, debug, warn, error, fatal")
	logFile      = flag.String(nmLogFile, "", "log file path")

	// Status
	reportStatus    = flagBoolean(nmReportStatus, true, "If enable status report HTTP service.")
	statusHost      = flag.String(nmStatusHost, "0.0.0.0", "server status host")
	statusPort      = flag.String(nmStatusPort, "10080", "server status port")
	metricsAddr     = flag.String(nmMetricsAddr, "", "prometheus pushgateway address, leaves it empty will disable prometheus push.")
	metricsInterval = flag.Uint(nmMetricsInterval, 15, "prometheus client push interval in second, set \"0\" to disable prometheus push.")

)

var (
	cfg      *config.Config
	svr      *tcp.Server
	graceful bool
)

var deprecatedConfig = map[string]struct{}{
	"pessimistic-txn.ttl": {},
	"log.rotate":          {},
}
// hotReloadConfigItems lists all config items which support hot-reload.
var hotReloadConfigItems = []string{"Performance.MaxProcs", "Performance.MaxMemory", "OOMAction", "MemQuotaQuery"}

func main() {
	flag.Parse()
	if *version {
		//fmt.Println(printer.Get...Info())
		os.Exit(0)
	}

	registerMetrics()
	configWarning := loadConfig()
	overrideConfig()
	if err := cfg.Valid(); err != nil {
		fmt.Fprintln(os.Stderr, "invalid config", err)
		os.Exit(1)
	}
	if *configCheck {
		fmt.Println("config check successful")
		os.Exit(0)
	}
	setGlobalVars()
	setCPUAffinity()
	setupLog()
	// If configStrict had been specified, and there had been an error, the server would already
	// have exited by now. If configWarning is not an empty string, write it to the log now that
	// it's been properly set up.
	if configWarning != "" {
		log.Warn(configWarning)
	}
	setupTracing() // Should before createServer and after setup config.
	printInfo()
	setupMetrics()
	createServer()
	signal.SetupSignalHandler(serverShutdown)
	runServer()
	cleanup()
	syncLog()
}

func exit() {
	syncLog()
	os.Exit(0)
}

func syncLog() {
	if err := log.Sync(); err != nil {
		fmt.Fprintln(os.Stderr, "sync log err:", err)
		os.Exit(1)
	}
}

func setCPUAffinity() {
	if affinityCPU == nil || len(*affinityCPU) == 0 {
		return
	}
	var cpu []int
	for _, af := range strings.Split(*affinityCPU, ",") {
		af = strings.TrimSpace(af)
		if len(af) > 0 {
			c, err := strconv.Atoi(af)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wrong affinity cpu config: %s", *affinityCPU)
				exit()
			}
			cpu = append(cpu, c)
		}
	}
	err := linux.SetAffinity(cpu)
	if err != nil {
		fmt.Fprintf(os.Stderr, "set cpu affinity failure: %v", err)
		exit()
	}
	runtime.GOMAXPROCS(len(cpu))
}

func registerMetrics() {
	metrics.RegisterMetrics()
}

// Prometheus push.
const zeroDuration = time.Duration(0)

// pushMetric pushes metrics in background.
func pushMetric(addr string, interval time.Duration) {
	if interval == zeroDuration || len(addr) == 0 {
		log.Info("disable Prometheus push client")
		return
	}
	log.Info("start prometheus push client", zap.String("server addr", addr), zap.String("interval", interval.String()))
	go prometheusPushClient(addr, interval)
}

// prometheusPushClient pushes metrics to Prometheus Pushgateway.
func prometheusPushClient(addr string, interval time.Duration) {
	// TODO: do not have uniq name, so we use host+port to compose a name.
	job := "iot"
	pusher := push.New(addr, job)
	pusher = pusher.Gatherer(prometheus.DefaultGatherer)
	pusher = pusher.Grouping("instance", instanceName())
	for {
		err := pusher.Push()
		if err != nil {
			log.Error("could not push metrics to prometheus pushgateway", zap.String("err", err.Error()))
		}
		time.Sleep(interval)
	}
}

func instanceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return fmt.Sprintf("%s_%d", hostname, cfg.Port)
}

// parseDuration parses lease argument string.
func parseDuration(lease string) time.Duration {
	dur, err := time.ParseDuration(lease)
	if err != nil {
		dur, err = time.ParseDuration(lease + "s")
	}
	if err != nil || dur < 0 {
		log.Fatal("invalid lease duration", zap.String("lease", lease))
	}
	return dur
}

func flagBoolean(name string, defaultVal bool, usage string) *bool {
	if !defaultVal {
		// Fix #4125, golang do not print default false value in usage, so we append it.
		usage = fmt.Sprintf("%s (default false)", usage)
		return flag.Bool(name, defaultVal, usage)
	}
	return flag.Bool(name, defaultVal, usage)
}

//var deprecatedConfig = map[string]struct{}{
//	"pessimistic-txn.ttl": {},
//	"log.rotate":          {},
//}


func setGlobalVars() {

	runtime.GOMAXPROCS(int(cfg.Performance.MaxProcs))

}


func setupLog() {
	err := logutil.InitZapLogger(cfg.Log.ToLogConfig())
	errors.MustNil(err)

	err = logutil.InitLogger(cfg.Log.ToLogConfig())
	errors.MustNil(err)
	// Disable automaxprocs log
	nopLog := func(string, ...interface{}) {}
	_, err = maxprocs.Set(maxprocs.Logger(nopLog))
	errors.MustNil(err)
}

func printInfo() {
	// Make sure the info is always printed.
	level := log.GetLevel()
	log.SetLevel(zap.InfoLevel)
	//printer.Print...Info()
	log.SetLevel(level)
}

func createServer() {
	tierDriver := NewTireDriver(cfg)
	var err error
	svr, err = tcp.NewServer(cfg, tierDriver)
	errors.MustNil(err)

}

func serverShutdown(isgraceful bool) {
	if isgraceful {
		graceful = true
	}
	svr.Close()
}

func setupMetrics() {
	// Enable the mutex profile, 1/10 of mutex blocking event sampling.
	runtime.SetMutexProfileFraction(10)
	systimeErrHandler := func() {
		metrics.TimeJumpBackCounter.Inc()
	}
	callBackCount := 0
	sucessCallBack := func() {
		callBackCount++
		// It is callback by monitor per second, we increase metrics.KeepAliveCounter per 5s.
		if callBackCount >= 5 {
			callBackCount = 0
			metrics.KeepAliveCounter.Inc()
			updateCPUUsageMetrics()
		}
	}
	go systimemon.StartMonitor(time.Now, systimeErrHandler, sucessCallBack)

	pushMetric(cfg.Status.MetricsAddr, time.Duration(cfg.Status.MetricsInterval)*time.Second)
}

func updateCPUUsageMetrics() {
	sysInfo, err := pidusage.GetStat(os.Getpid())
	if err != nil {
		return
	}
	metrics.CPUUsagePercentageGauge.Set(sysInfo.CPU)
}

func setupTracing() {
	tracingCfg := cfg.OpenTracing.ToTracingConfig()
	tracer, _, err := tracingCfg.New("tire")
	if err != nil {
		log.Fatal("setup jaeger tracer failed", zap.String("error message", err.Error()))
	}
	opentracing.SetGlobalTracer(tracer)
}

func runServer() {
	err := svr.Run()
	errors.MustNil(err)
}

func cleanup() {
	if graceful {
		svr.GracefulDown(context.Background(), nil)
	} else {
		svr.TryGracefulDown()
	}

}

func reloadConfig(nc, c *config.Config) {
	// Just a part of config items need to be reload explicitly.
	// Some of them like OOMAction are always used by getting from global config directly
	// like config.GetGlobalConfig().OOMAction.
	// These config items will become available naturally after the global config pointer
	// is updated in function ReloadGlobalConfig.
	if nc.Performance.MaxMemory != c.Performance.MaxMemory {
		//
	}

}




func isDeprecatedConfigItem(items []string) bool {
	for _, item := range items {
		if _, ok := deprecatedConfig[item]; !ok {
			return false
		}
	}
	return true
}
func loadConfig() string {
	cfg = config.GetGlobalConfig()
	if *configPath != "" {
		// Not all config items are supported now.
		config.SetConfReloader(*configPath, reloadConfig, hotReloadConfigItems...)

		err := cfg.Load(*configPath)
		if err == nil {
			return ""
		}

		// Unused config item erro turns to warnings.
		if tmp, ok := err.(*config.ErrConfigValidationFailed); ok {
			if isDeprecatedConfigItem(tmp.UndecodedItems) {
				return err.Error()
			}
			// This block is to accommodate an interim situation where strict config checking
			// is not the default behavior of server. The warning message must be deferred until
			// logging has been set up. After strict config checking is the default behavior,
			// This should all be removed.
			if !*configCheck && !*configStrict {
				return err.Error()
			}
		}

		errors.MustNil(err)
	} else {
		// configCheck should have the config file specified.
		if *configCheck {
			fmt.Fprintln(os.Stderr, "config check failed", errors.New("no config file specified for config-check"))
			os.Exit(1)
		}
	}
	return ""
}

func overrideConfig() {
	actualFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		actualFlags[f.Name] = true
	})

	// Base
	if actualFlags[nmHost] {
		cfg.Host = *host
	}
	if len(cfg.AdvertiseAddress) == 0 {
		cfg.AdvertiseAddress = cfg.Host
	}
	var err error
	if actualFlags[nmPort] {
		var p int
		p, err = strconv.Atoi(*port)
		errors.MustNil(err)
		cfg.Port = uint(p)
	}

	if actualFlags[nmTokenLimit] {
		cfg.TokenLimit = uint(*tokenLimit)
	}

	// Log
	if actualFlags[nmLogLevel] {
		cfg.Log.Level = *logLevel
	}
	if actualFlags[nmLogFile] {
		cfg.Log.File.Filename = *logFile
	}

	// Status
	if actualFlags[nmReportStatus] {
		cfg.Status.ReportStatus = *reportStatus
	}
	if actualFlags[nmStatusHost] {
		cfg.Status.StatusHost = *statusHost
	}
	if actualFlags[nmStatusPort] {
		var p int
		p, err = strconv.Atoi(*statusPort)
		errors.MustNil(err)
		cfg.Status.StatusPort = uint(p)
	}
	if actualFlags[nmMetricsAddr] {
		cfg.Status.MetricsAddr = *metricsAddr
	}
	if actualFlags[nmMetricsInterval] {
		cfg.Status.MetricsInterval = *metricsInterval
	}


}