package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/frankhang/doppler/api/healthprobe"
	"github.com/frankhang/doppler/forwarder"
	"github.com/frankhang/doppler/metadata"
	"github.com/frankhang/doppler/serializer"
	"github.com/frankhang/doppler/agent"
	"github.com/frankhang/doppler/aggregator"
	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/metrics"
	"github.com/frankhang/doppler/status/health"
	"github.com/frankhang/doppler/tagger"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/util/config"
	"github.com/frankhang/util/logutil"


	"go.uber.org/automaxprocs/maxprocs"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	//l "github.com/sirupsen/logrus"
	sig "github.com/frankhang/util/signal"
	"github.com/frankhang/util/sys/linux"
	"github.com/frankhang/util/tcp"

	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/log"

	m "github.com/frankhang/util/metrics"

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

	metaScheduler *metadata.Scheduler
	statsd        *agent.Server

)

var (
	svr      *tcp.Server
	graceful bool
)


// hotReloadConfigItems lists all config items which support hot-reload.

func main() {
	flag.Parse()
	if *version {
		//fmt.Println(printer.Get...Info())
		os.Exit(0)
	}

	registerMetrics()
	configWarning := loadConfig()
	overrideConfig()
	if err := Cfg.Valid(); err != nil {
		fmt.Fprintln(os.Stderr, "invalidx config", err)
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
	sig.SetupSignalHandler(serverShutdown)
	runServer()
	//cleanup()
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
	m.RegisterMetrics()
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
	return fmt.Sprintf("%s_%d", hostname, Cfg.Port)
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


func setGlobalVars() {

	runtime.GOMAXPROCS(int(Cfg.Performance.MaxProcs))

}


func setupLog() {
	err := logutil.InitZapLogger(Cfg.Log.ToLogConfig())
	errors.MustNil(err)

	err = logutil.InitLogger(Cfg.Log.ToLogConfig())
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
	//tierDriver := NewTireDriver(cfg)
	//var err error
	//svr, err = tcp.NewServer(cfg, tierDriver)
	//errors.MustNil(err)

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
		m.TimeJumpBackCounter.Inc()
	}
	callBackCount := 0
	sucessCallBack := func() {
		callBackCount++
		// It is callback by monitor per second, we increase metrics.KeepAliveCounter per 5s.
		if callBackCount >= 5 {
			callBackCount = 0
			m.KeepAliveCounter.Inc()
			updateCPUUsageMetrics()
		}
	}
	go systimemon.StartMonitor(time.Now, systimeErrHandler, sucessCallBack)

	pushMetric(Cfg.Status.MetricsAddr, time.Duration(Cfg.Status.MetricsInterval)*time.Second)
}

func updateCPUUsageMetrics() {
	sysInfo, err := pidusage.GetStat(os.Getpid())
	if err != nil {
		return
	}
	m.CPUUsagePercentageGauge.Set(sysInfo.CPU)
}

func setupTracing() {
	tracingCfg := Cfg.OpenTracing.ToTracingConfig()
	tracer, _, err := tracingCfg.New("tire")
	if err != nil {
		log.Fatal("setup jaeger tracer failed", zap.String("error message", err.Error()))
	}
	opentracing.SetGlobalTracer(tracer)
}


func cleanup() {
	if graceful {
		svr.GracefulDown(context.Background(), nil)
	} else {
		svr.TryGracefulDown()
	}

}

func isDeprecatedConfigItem(items []string) bool {
	for _, item := range items {
		if _, ok := DeprecatedConfig[item]; !ok {
			return false
		}
	}
	return true
}


func overrideConfig() {
	actualFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		actualFlags[f.Name] = true
	})

	// Base
	if actualFlags[nmHost] {
		Cfg.Host = *host
	}
	if len(Cfg.AdvertiseAddress) == 0 {
		Cfg.AdvertiseAddress = Cfg.Host
	}
	var err error
	if actualFlags[nmPort] {
		var p int
		p, err = strconv.Atoi(*port)
		errors.MustNil(err)
		Cfg.Port = uint(p)
	}

	if actualFlags[nmTokenLimit] {
		Cfg.TokenLimit = uint(*tokenLimit)
	}

	// Log
	if actualFlags[nmLogLevel] {
		Cfg.Log.Level = *logLevel
	}
	if actualFlags[nmLogFile] {
		Cfg.Log.File.Filename = *logFile
	}

	// Status
	if actualFlags[nmReportStatus] {
		Cfg.Status.ReportStatus = *reportStatus
	}
	if actualFlags[nmStatusHost] {
		Cfg.Status.StatusHost = *statusHost
	}
	if actualFlags[nmStatusPort] {
		var p int
		p, err = strconv.Atoi(*statusPort)
		errors.MustNil(err)
		Cfg.Status.StatusPort = uint(p)
	}
	if actualFlags[nmMetricsAddr] {
		Cfg.Status.MetricsAddr = *metricsAddr
	}
	if actualFlags[nmMetricsInterval] {
		Cfg.Status.MetricsInterval = *metricsInterval
	}


}



func loadConfig() string {
	Cfg = GetGlobalConfig()
	if *configPath != "" {
		// Not all config items are supported now.
		config.SetConfReloader(*configPath, reloadConfig, HotReloadConfigItems...)

		err := Cfg.Load(*configPath)
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


func runServer() {


	mainCtx, mainCtxCancel, err := runAgent()
	errors.MustNil(err)
	// Setup a channel to catch OS signals
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

	// Block here until we receive the interrupt signal
	<-signalCh

	stopAgent(mainCtx, mainCtxCancel)

}


func runAgent() (mainCtx context.Context, mainCtxCancel context.CancelFunc, err error) {
	// Main context passed to components
	mainCtx, mainCtxCancel = context.WithCancel(context.Background())


	//if !config.Datadog.IsSet("api_key") {
	//	log.Critical("no API key configured, exiting")
	//	return
	//}

	// Setup healthcheck port
	var healthPort = Cfg.HealthPort
	if healthPort > 0 {
		err = healthprobe.Serve(mainCtx, healthPort)
		if err != nil {
			err = errors.Trace(err)
			return
		}
		logutil.BgLogger().Info("Health check listening...", zap.Int("port", healthPort))
	}

	// setup the forwarder
	keysPerDomain, err := GetMultipleEndpoints()
	if err != nil {
		logutil.BgLogger().Error("Misconfiguration of agent endpoints", zap.Error(err))
	}
	f := forwarder.NewDefaultForwarder(keysPerDomain)
	f.Start()
	s := serializer.NewSerializer(f)

	hname, err := util.GetHostname()
	if err != nil {
		logutil.BgLogger().Warn("Error getting hostname", zap.Error(err))
		hname = ""
	}
	logutil.BgLogger().Info("Using hostname", zap.String("hostname", hname))


	// setup the metadata collector
	metaScheduler = metadata.NewScheduler(s)
	if err = metadata.SetupMetadataCollection(metaScheduler, []string{"host"}); err != nil {
		metaScheduler.Stop()
		return
	}

	if Cfg.InventoriesEnabled {
		if err = metadata.SetupInventories(metaScheduler, nil, nil); err != nil {
			return
		}
	}

	// container tagging initialisation if origin detection is on
	if Cfg.AgentOriginDetection {
		tagger.Init()
	}

	metricSamplePool := metrics.NewMetricSamplePool(32)
	aggregatorInstance := aggregator.InitAggregator(s, metricSamplePool, hname, "agent")
	sampleC, eventC, serviceCheckC := aggregatorInstance.GetBufferedChannels()
	statsd, err = agent.NewServer(metricSamplePool, sampleC, eventC, serviceCheckC)
	if err != nil {
		logutil.BgLogger().Error("Unable to start dogstatsd")
		err = errors.Trace(err)
		return
	}
	return
}

func stopAgent(ctx context.Context, cancel context.CancelFunc) {
	// retrieve the agent health before stopping the components
	// GetStatusNonBlocking has a 100ms timeout to avoid blocking
	health, err := health.GetStatusNonBlocking()
	if err != nil {
		logutil.BgLogger().Warn("Agent: health unknown", zap.Error(err))
	} else if len(health.Unhealthy) > 0 {
		logutil.BgLogger().Warn("Agent: some components were unhealthy", zap.Strings("components", health.Unhealthy))
	}

	// gracefully shut down any component
	cancel()

	metaScheduler.Stop()
	statsd.Stop()
	logutil.BgLogger().Info("See ya!")
	//log.Flush()
	return
}

