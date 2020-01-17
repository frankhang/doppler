package main

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/frankhang/util/config"
	"github.com/frankhang/util/errors"
	"os"
	"sync/atomic"
)

type Config struct {
	config.Config
	Test string `toml:"test" json:"test"`
}

var (
	cfg         *Config
	GlobalConf  = atomic.Value{}
	DefaultConf = Config{
		Config: config.DefaultConf,
		Test:   "t1",
	}

	hotReloadConfigItems = []string{"Performance.MaxProcs", "Performance.MaxMemory", "OOMAction", "MemQuotaQuery"}

	deprecatedConfig = map[string]struct{}{
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

func loadConfig() string {
	cfg = GetGlobalConfig()
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
