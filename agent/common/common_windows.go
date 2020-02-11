// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package common

import (
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"path/filepath"

	"github.com/frankhang/doppler/config"

	"github.com/frankhang/util/logutil"
	"github.com/frankhang/doppler/util/winutil"
	"github.com/cihub/seelog"
	"golang.org/x/sys/windows/registry"
	yaml "gopkg.in/yaml.v2"
)

var (
	// PyChecksPath holds the path to the python checks from integrations-core shipped with the agent
	PyChecksPath = filepath.Join(_here, "..", "checks.d")
	distPath     string
	// ViewsPath holds the path to the folder containing the GUI support files
	viewsPath   string
	enabledVals = map[string]bool{"yes": true, "true": true, "1": true,
		"no": false, "false": false, "0": false}
	subServices = map[string]string{"logs_enabled": "logs_enabled",
		"apm_enabled":     "apm_config.enabled",
		"process_enabled": "process_config.enabled"}
)

var (
	// DefaultConfPath points to the folder containing datadog.yaml
	DefaultConfPath = "c:\\programdata\\datadog"
	// DefaultLogFile points to the log file that will be used if not configured
	DefaultLogFile = "c:\\programdata\\datadog\\logs\\agent.log"
	// DefaultDCALogFile points to the log file that will be used if not configured
	DefaultDCALogFile = "c:\\programdata\\datadog\\logs\\cluster-agent.log"
)

func init() {
	pd, err := winutil.GetProgramDataDir()
	if err == nil {
		DefaultConfPath = pd
		DefaultLogFile = filepath.Join(pd, "logs", "agent.log")
		DefaultDCALogFile = filepath.Join(pd, "logs", "cluster-agent.log")
	} else {
		winutil.LogEventViewer(config.ServiceName, 0x8000000F, DefaultConfPath)
	}
}

// EnableLoggingToFile -- set up logging to file
func EnableLoggingToFile() {
	seeConfig := `
<seelog>
	<outputs>
		<rollingfile type="size" filename="c:\\ProgramData\\DataDog\\Logs\\agent.log" maxsize="1000000" maxrolls="2" />
	</outputs>
</seelog>`
	logger, _ := seelog.LoggerFromConfigAsBytes([]byte(seeConfig))
	log.ReplaceLogger(logger)
}

func getInstallPath() string {
	// fetch the installation path from the registry
	installpath := filepath.Join(_here, "..")
	var s string
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\DataDog\Datadog Agent`, registry.QUERY_VALUE)
	if err != nil {
		logutil.BgLogger().Warn("Failed to open registry key", zap.Error(err))
	} else {
		defer k.Close()
		s, _, err = k.GetStringValue("InstallPath")
		if err != nil {
			logutil.BgLogger().Warn("Installpath not found in registry", zap.Error(err))
		}
	}
	// if unable to figure out the install path from the registry,
	// just compute it relative to the executable.
	if s == "" {
		s = installpath
	}
	return s
}

// GetDistPath returns the fully qualified path to the 'dist' directory
func GetDistPath() string {
	if len(distPath) == 0 {
		var s string
		if s = getInstallPath(); s == "" {
			return ""
		}
		distPath = filepath.Join(s, `bin/agent/dist`)
	}
	return distPath
}

// GetViewsPath returns the fully qualified path to the GUI's 'views' directory
func GetViewsPath() string {
	if len(viewsPath) == 0 {
		var s string
		if s = getInstallPath(); s == "" {
			return ""
		}
		viewsPath = filepath.Join(s, "bin", "agent", "dist", "views")
		logutil.BgLogger().Debug(fmt.Sprintf("ViewsPath is now %s", viewsPath))
	}
	return viewsPath
}

// CheckAndUpgradeConfig checks to see if there's an old datadog.conf, and if
// datadog.yaml is either missing or incomplete (no API key).  If so, upgrade it
func CheckAndUpgradeConfig() error {
	datadogConfPath := filepath.Join(DefaultConfPath, "datadog.conf")
	if _, err := os.Stat(datadogConfPath); os.IsNotExist(err) {
		logutil.BgLogger().Debug("Previous config file not found, not upgrading")
		return nil
	}
	config.Datadog.AddConfigPath(DefaultConfPath)
	err := config.Load()
	if err == nil {
		// was able to read config, check for api key
		if config.Datadog.GetString("api_key") != "" {
			logutil.BgLogger().Debug("Datadog.yaml found, and API key present.  Not upgrading config")
			return nil
		}
	}
	return ImportConfig(DefaultConfPath, DefaultConfPath, false)
}

// ImportRegistryConfig imports settings from Windows registry into datadog.yaml
func ImportRegistryConfig() error {

	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		"SOFTWARE\\Datadog\\Datadog Agent",
		registry.ALL_ACCESS)
	if err != nil {
		if err == registry.ErrNotExist {
			logutil.BgLogger().Debug("Windows installation key not found, not updating config")
			return nil
		}
		// otherwise, unexpected error
		logutil.BgLogger().Warnf("Unexpected error getting registry config", zap.Error(err))
		return err
	}
	defer k.Close()

	err = SetupConfigWithoutSecrets("", "")
	if err != nil {
		return fmt.Errorf("unable to set up global agent configuration: %v", err)
	}

	// store the current datadog.yaml path
	datadogYamlPath := config.Datadog.ConfigFileUsed()

	if config.Datadog.GetString("api_key") != "" {
		return fmt.Errorf("%s seems to contain a valid configuration, not overwriting config",
			datadogYamlPath)
	}

	overrides := make(map[string]interface{})

	var val string

	if val, _, err = k.GetStringValue("api_key"); err == nil && val != "" {
		overrides["api_key"] = val
		logutil.BgLogger().Debug("Setting API key")
	} else {
		logutil.BgLogger().Debug("API key not found, not setting")
	}
	if val, _, err = k.GetStringValue("tags"); err == nil && val != "" {
		overrides["tags"] = strings.Split(val, ",")
		logutil.BgLogger().Debug(fmt.Sprintf("Setting tags %s", val))
	} else {
		logutil.BgLogger().Debug("Tags not found, not setting")
	}
	if val, _, err = k.GetStringValue("hostname"); err == nil && val != "" {
		overrides["hostname"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting hostname %s", val))
	} else {
		logutil.BgLogger().Debug("hostname not found in registry: using default value")
	}
	if val, _, err = k.GetStringValue("cmd_port"); err == nil && val != "" {
		cmdPortInt, err := strconv.Atoi(val)
		if err != nil {
			logutil.BgLogger().Warn(fmt.Sprintf("Not setting api port, invalid configuration %s", val), zap.Error(err))
		} else if cmdPortInt <= 0 || cmdPortInt > 65534 {
			logutil.BgLogger().Warn(fmt.Sprintf("Not setting api port, invalid configuration %s", val))
		} else {
			overrides["cmd_port"] = cmdPortInt
			logutil.BgLogger().Debug(fmt.Sprintf("Setting cmd_port  %d", cmdPortInt))
		}
	} else {
		logutil.BgLogger().Debug("cmd_port not found, not setting")
	}
	for key, cfg := range subServices {
		if val, _, err = k.GetStringValue(key); err == nil {
			val = strings.ToLower(val)
			if enabled, ok := enabledVals[val]; ok {
				// some of the entries require booleans, some
				// of the entries require strings.
				if enabled {
					switch cfg {
					case "logs_enabled":
						overrides[cfg] = true
					case "apm_config.enabled":
						overrides[cfg] = true
					case "process_config.enabled":
						overrides[cfg] = "true"
					}
					logutil.BgLogger().Debug(fmt.Sprintf("Setting %s to true", cfg))
				} else {
					switch cfg {
					case "logs_enabled":
						overrides[cfg] = false
					case "apm_config.enabled":
						overrides[cfg] = false
					case "process_config.enabled":
						overrides[cfg] = "disabled"
					}
					logutil.BgLogger().Debug(fmt.Sprintf("Setting %s to false", cfg))
				}
			} else {
				logutil.BgLogger().Warn(fmt.Sprintf("Unknown setting %s = %s", key, val))
			}
		}
	}
	if val, _, err = k.GetStringValue("proxy_host"); err == nil && val != "" {
		var u *url.URL
		if u, err = url.Parse(val); err != nil {
			logutil.BgLogger().Warn("unable to import value of settings 'proxy_host'", zap.Error(err))
		} else {
			// set scheme if missing
			if u.Scheme == "" {
				u, _ = url.Parse("http://" + val)
			}
			if val, _, err = k.GetStringValue("proxy_port"); err == nil && val != "" {
				u.Host = u.Host + ":" + val
			}
			if user, _, _ := k.GetStringValue("proxy_user"); err == nil && user != "" {
				if pass, _, _ := k.GetStringValue("proxy_password"); err == nil && pass != "" {
					u.User = url.UserPassword(user, pass)
				} else {
					u.User = url.User(user)
				}
			}
		}
		proxyMap := make(map[string]string)
		proxyMap["http"] = u.String()
		proxyMap["https"] = u.String()
		overrides["proxy"] = proxyMap
	} else {
		logutil.BgLogger().Debug("proxy key not found, not setting proxy config")
	}
	if val, _, err = k.GetStringValue("site"); err == nil && val != "" {
		overrides["site"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting site to %s", val))
	}
	if val, _, err = k.GetStringValue("dd_url"); err == nil && val != "" {
		overrides["dd_url"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting dd_url to %s", val))
	}
	if val, _, err = k.GetStringValue("logs_dd_url"); err == nil && val != "" {
		overrides["logs_config.logs_dd_url"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting logs_config.dd_url to %s", val))
	}
	if val, _, err = k.GetStringValue("process_dd_url"); err == nil && val != "" {
		overrides["process_config.process_dd_url"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting process_config.process_dd_url to %s", val))
	}
	if val, _, err = k.GetStringValue("trace_dd_url"); err == nil && val != "" {
		overrides["apm_config.apm_dd_url"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting apm_config.apm_dd_url to %s", val))
	}
	if val, _, err = k.GetStringValue("py_version"); err == nil && val != "" {
		overrides["python_version"] = val
		logutil.BgLogger().Debug(fmt.Sprintf("Setting python version to %s", val))
	}

	// apply overrides to the config
	config.AddOverrides(overrides)

	// build the global agent configuration
	err = SetupConfigWithoutSecrets("", "")
	if err != nil {
		return fmt.Errorf("unable to set up global agent configuration: %v", err)
	}

	// dump the current configuration to datadog.yaml
	b, err := yaml.Marshal(config.Datadog.AllSettings())
	if err != nil {
		logutil.BgLogger().Error("unable to unmarshal config to YAML", zap.Error(err))
		return fmt.Errorf("unable to unmarshal config to YAML: %v", err)
	}
	// file permissions will be used only to create the file if doesn't exist,
	// please note on Windows such permissions have no effect.
	if err = ioutil.WriteFile(datadogYamlPath, b, 0640); err != nil {
		logutil.BgLogger().Error(fmt.Sprintf("unable to unmarshal config to %s", datadogYamlPath), zap.Error(err))
		return fmt.Errorf("unable to unmarshal config to %s: %v", datadogYamlPath, err)
	}

	valuenames := []string{"api_key", "tags", "hostname",
		"proxy_host", "proxy_port", "proxy_user", "proxy_password", "cmd_port"}
	for _, valuename := range valuenames {
		k.DeleteValue(valuename)
	}
	for valuename := range subServices {
		k.DeleteValue(valuename)
	}
	logutil.BgLogger().Debug("Successfully wrote the config into %s", datadogYamlPath)

	return nil
}
