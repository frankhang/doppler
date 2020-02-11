// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package common

import (
	"fmt"
	"go.uber.org/zap"
	"path/filepath"

	"github.com/frankhang/doppler/autodiscovery"
	"github.com/frankhang/doppler/autodiscovery/providers"
	"github.com/frankhang/doppler/autodiscovery/scheduler"
	"github.com/frankhang/doppler/collector"
	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/logs"
	"github.com/frankhang/doppler/tagger"
	"github.com/frankhang/util/logutil"
)

// SetupAutoConfig configures the global AutoConfig:
//   1. add the configuration providers
//   2. add the check loaders
func SetupAutoConfig(confdPath string) {
	// start tagging system
	tagger.Init()

	// create the Collector instance and start all the components
	// NOTICE: this will also setup the Python environment, if available
	Coll = collector.NewCollector(GetPythonPaths()...)

	// creating the meta scheduler
	metaScheduler := scheduler.NewMetaScheduler()

	// registering the check scheduler
	metaScheduler.Register("check", collector.InitCheckScheduler(Coll))

	// registering the logs scheduler
	if logs.IsAgentRunning() {
		metaScheduler.Register("logs", logs.GetScheduler())
	}

	// create the Autoconfig instance
	AC = autodiscovery.NewAutoConfig(metaScheduler)

	// Add the configuration providers
	// File Provider is hardocded and always enabled
	confSearchPaths := []string{
		confdPath,
		filepath.Join(GetDistPath(), "conf.d"),
		"",
	}
	AC.AddConfigProvider(providers.NewFileConfigProvider(confSearchPaths), false, 0)

	// Register additional configuration providers
	var CP []config.ConfigurationProviders
	err := config.Datadog.UnmarshalKey("config_providers", &CP)

	if err == nil {
		// Add extra config providers
		for _, name := range config.Datadog.GetStringSlice("extra_config_providers") {
			CP = append(CP, config.ConfigurationProviders{Name: name, Polling: true})
		}
		for _, cp := range CP {
			factory, found := providers.ProviderCatalog[cp.Name]
			if found {
				configProvider, err := factory(cp)
				if err == nil {
					pollInterval := providers.GetPollInterval(cp)
					if cp.Polling {
						logutil.BgLogger().Info(fmt.Sprintf("Registering %s config provider polled every %s", cp.Name, pollInterval.String()))
					} else {
						logutil.BgLogger().Info(fmt.Sprintf("Registering %s config provider", cp.Name))
					}
					AC.AddConfigProvider(configProvider, cp.Polling, pollInterval)
				} else {
					logutil.BgLogger().Error(fmt.Sprintf("Error while adding config provider %v", cp.Name), zap.Error(err))
				}
			} else {
				logutil.BgLogger().Error(fmt.Sprintf("Unable to find this provider in the catalog: %v", cp.Name))
			}
		}
	} else {
		logutil.BgLogger().Error("Error while reading 'config_providers' settings", zap.Error(err))
	}

	// Autodiscovery listeners
	// for now, no need to implement a registry of available listeners since we
	// have only docker
	var listeners []config.Listeners
	err = config.Datadog.UnmarshalKey("listeners", &listeners)
	if err == nil {
		// Add extra listeners
		for _, name := range config.Datadog.GetStringSlice("extra_listeners") {
			listeners = append(listeners, config.Listeners{Name: name})
		}
		listeners = AutoAddListeners(listeners)
		AC.AddListeners(listeners)
	} else {
		logutil.BgLogger().Error("Error while reading 'listeners' settings", zap.Error(err))
	}
}

// StartAutoConfig starts the autoconfig:
//   1. load all the configurations available at startup
//   2. run all the Checks for each configuration found
func StartAutoConfig() {
	AC.LoadAndRun()
}
