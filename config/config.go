package config

import (
	"github.com/BurntSushi/toml"
	"github.com/frankhang/util/config"
	"sync/atomic"
)

type Config struct {
	config.Config
	Test string `toml:"test" json:"test"`


	HealthPort int `toml:"health_port" json:"health_port"`
	InventoriesEnabled bool `toml:"inventories_enabled" json:"inventories_enabled"`

	AgentOriginDetection bool `toml:"agent_origin_detection" json:"agent_origin_detection"`

	AgentStatsEnable bool `toml:"agent_stats_enable" json:"agent_stats_enable"`
	AgentStatsBuffer int `toml:"agent_stats_buffer" json:"agent_stats_buffer"`

	AgentTags []string `toml:"agent_tags" json:"agent_tags"`

	MetricsStatsEnable bool `toml:"metrics_stats_enable" json:"metrics_stats_enable"`
	QueueSize int `toml:"queue_size" json:"queue_size"`
	BufferSize int `toml:"queue_size" json:"buffer_size"`
	MetricNamespace string `toml:"metric_namespac" json:"metric_namespace"`
	MetricNamespaceBlacklist []string `toml:"metric_namespace_blacklist" json:"metric_namespace_blacklist"`
	ForwardHost string `toml:"forward_host" json:"forward_host"`
	ForwardPort int `toml:"forward_port" json:"forward_port"`
	CacheSize int `toml:"cache_size" json:"cache_size"`

	HistogramCopyToDistribution bool `toml:"histogram_copy_to_distribution" json:"histogram_copy_to_distribution"`
	HistogramCopyToDistributionPrefix string `toml:"histogram_copy_to_distribution_prefix" json:"histogram_copy_to_distribution_prefix"`

}

var (
	Cfg         *Config
	GlobalConf  = atomic.Value{}
	DefaultConf = Config{
		Config: config.DefaultConf,
		Test:   "t1",
		HealthPort: 0,
		QueueSize: 20,
		BufferSize: 8192,
		HistogramCopyToDistribution: false,
	}

	HotReloadConfigItems = []string{"Performance.MaxProcs", "Performance.MaxMemory", "OOMAction", "MemQuotaQuery"}

	DeprecatedConfig = map[string]struct{}{
		"pessimistic-txn.ttl": {},
		"log.rotate":          {},
	}
)

func init() {
	GlobalConf.Store(&DefaultConf)
}

// GetGlobalConfig returns the global configuration for this server.
// It should store configuration from command line and configuration file.
// Other parts of the system can read the global configuration use this function.
func GetGlobalConfig() *Config {
	return GlobalConf.Load().(*Config)
}

// Load loads config options from a toml file.
func (c *Config) Load(confFile string) error {
	metaData, err := toml.DecodeFile(confFile, c)
	if c.TokenLimit == 0 {
		c.TokenLimit = 1000
	}
	// If any items in confFile file are not mapped into the Config struct, issue
	// an error and stop the server from starting.
	undecoded := metaData.Undecoded()
	if len(undecoded) > 0 && err == nil {
		var undecodedItems []string
		for _, item := range undecoded {
			undecodedItems = append(undecodedItems, item.String())
		}
		err = &config.ErrConfigValidationFailed{confFile, undecodedItems}
	}

	return err
}

