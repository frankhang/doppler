package config

import (
	"github.com/BurntSushi/toml"
	"github.com/frankhang/util/config"
	"os"
	"sync/atomic"
	"time"
)

type Config struct {
	config.Config
	Test string `toml:"test" json:"test"`

	HealthPort         int  `toml:"health_port" json:"health_port"`
	InventoriesEnabled bool `toml:"inventories_enabled" json:"inventories_enabled"`
	MetricsStatsEnable bool `toml:"metrics_stats_enable" json:"metrics_stats_enable"`

	MetricNamespace          string   `toml:"metric_namespac" json:"metric_namespace"`
	MetricNamespaceBlacklist []string `toml:"metric_namespace_blacklist" json:"metric_namespace_blacklist"`
	ForwardHost              string   `toml:"forward_host" json:"forward_host"`
	ForwardPort              int      `toml:"forward_port" json:"forward_port"`
	CacheSize                int      `toml:"cache_size" json:"cache_size"`

	HistogramCopyToDistribution       bool   `toml:"histogram_copy_to_distribution" json:"histogram_copy_to_distribution"`
	HistogramCopyToDistributionPrefix string `toml:"histogram_copy_to_distribution_prefix" json:"histogram_copy_to_distribution_prefix"`

	AgentBufferSize               int  `toml:"agent_buffer_size" json:"agent_buffer_size"`
	AgentPacketBufferSize         int  `toml:"agent_packet_buffer_size" json:"agent_packet_buffer_size"`
	AgentPacketBufferFlushTimeout int  `toml:"agent_packet_buffer_flush_timeout" json:"agent_packet_buffer_flush_timeout"` //ms
	AgentQueueSize                int  `toml:"agent_queue_size" json:"agent_queue_size"`
	AgentSoRcvbuf                 int  `toml:"agent_so_rcvbuf" json:"agent_so_rcvbuf"`
	AgentNonLocalTraffic          bool `toml:"agent_non_local_traffic" json:"agent_non_local_traffic"`
	AgentOriginDetection          bool `toml:"agent_origin_detection" json:"agent_origin_detection"`

	AgentStatsEnable bool `toml:"agent_stats_enable" json:"agent_stats_enable"`
	AgentStatsBuffer int  `toml:"agent_stats_buffer" json:"agent_stats_buffer"`

	AgentTags []string `toml:"agent_tags" json:"agent_tags"` //接受时加
	Tags      []string `toml:"tags" json:"tags"`             //转发时加

	ForwarderNumWorkers        int `toml:"forwarder_num_workers" json:"forwarder_num_workers"`
	ForwarderRetryQueueMaxSize int `toml:"forwarder_retry_queue_max_size" json:"forwarder_retry_queue_max_size"`

	EnablePayloadsEvents         bool `toml:"enable_payloads.events" json:"enable_payloads.events"`
	EnablePayloadsSeries         bool `toml:"enable_payloads.series" json:"enable_payloads.series"`
	EnablePayloadsServiceChecks  bool `toml:"enable_payloads.service_checks" json:"enable_payloads.service_checks"`
	EnablePayloadsSketches       bool `toml:"enable_payloads.sketches" json:"enable_payloads.sketches"`
	EnablePayloadsJsonToV1Intake bool `toml:"enable_payloads.json_to_v1_intake" json:"enable_payloads.json_to_v1_intake"`

	HostName                       string `toml:"hostname" json:"hostname"`
	HostNameFqdn                   bool   `toml:"hostname_fqdn" json:"hostname_fqdn"`
	HostnameForceConfigAsCanonical bool   `toml:"hostname_force_config_as_canonical" json:"hostname_force_config_as_canonical"`

	ApiKey string `toml:"api_key" json:"api_key"`

	MetadataEndpointsMaxHostnameSize int `toml:"metadata_endpoints_max_hostname_size" json:"metadata_endpoints_max_hostname_size"`

	TagValueSplitSeparator   map[string]string   `toml:"tag_value_split_separator" json:"tag_value_split_separator"`
	ClusterName              string              `toml:"cluster_name" json:"cluster_name"`
	NetworkId                string              `toml:"network_id" json:"network_id"`
	EnableGohai              bool                `toml:"enable_gohai" json:"enable_gohai"`
	EnableMetadataCollection bool                `toml:"enable_metadata_collection" json:"enable_metadata_collection"`
	MetadataProviders        []MetadataProviders `toml:"metadata_providers" json:"metadata_providers"`
}

// MetadataProviders helps unmarshalling `metadata_providers` config param
type MetadataProviders struct {
	Name     string        `toml:"name" json:"name"`
	Interval time.Duration `toml:"interval" json:"interval"`
}

var (
	Cfg         *Config
	GlobalConf  = atomic.Value{}
	DefaultConf = Config{
		Config: config.DefaultConf,
		Test:   "t1",

		HealthPort: 0,

		HistogramCopyToDistribution:   false,
		AgentBufferSize:               8192,
		AgentPacketBufferSize:         32,
		AgentPacketBufferFlushTimeout: 100,
		AgentQueueSize:                1024,

		ForwarderNumWorkers:        1,
		ForwarderRetryQueueMaxSize: 30,

		HostNameFqdn:                     true,
		MetadataEndpointsMaxHostnameSize: 512,
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

// IsContainerized returns whether the Agent is running on a Docker container
func IsContainerized() bool {
	return os.Getenv("DOCKER_DD_AGENT") != ""
}
