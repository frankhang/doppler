// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package config

import (

	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	//yaml "gopkg.in/yaml.v2"



	"github.com/frankhang/doppler/version"
)

const (
	// DefaultSite is the default site the Agent sends data to.
	DefaultSite    = "datadoghq.com"
	infraURLPrefix = "https://app."

	// DefaultNumWorkers default number of workers for our check runner
	DefaultNumWorkers = 4
	// MaxNumWorkers maximum number of workers for our check runner
	MaxNumWorkers = 25

	// DefaultForwarderRecoveryInterval is the default recovery interval,
	// also used if the user-provided value is invalid.
	DefaultForwarderRecoveryInterval = 2

	megaByte = 1024 * 1024

	// DefaultBatchWait is the default HTTP batch wait in second for logs
	DefaultBatchWait = 5
)

var overrideVars = make(map[string]interface{})

// Datadog is the global configuration object
var (
	Datadog Config2
	proxies *Proxy
)

// Variables to initialize at build time
var (
	DefaultPython string

	// ForceDefaultPython has its value set to true at compile time if we should ignore
	// the Python version set in the configuration and use `DefaultPython` instead.
	// We use this to force Python 3 in the Agent 7 as it's the only one available.
	ForceDefaultPython string
)

// MetadataProviders helps unmarshalling `metadata_providers` config param
type MetadataProviders struct {
	Name     string        `mapstructure:"name"`
	Interval time.Duration `mapstructure:"interval"`
}

// ConfigurationProviders helps unmarshalling `config_providers` config param
type ConfigurationProviders struct {
	Name             string `mapstructure:"name"`
	Polling          bool   `mapstructure:"polling"`
	PollInterval     string `mapstructure:"poll_interval"`
	TemplateURL      string `mapstructure:"template_url"`
	TemplateDir      string `mapstructure:"template_dir"`
	Username         string `mapstructure:"username"`
	Password         string `mapstructure:"password"`
	CAFile           string `mapstructure:"ca_file"`
	CAPath           string `mapstructure:"ca_path"`
	CertFile         string `mapstructure:"cert_file"`
	KeyFile          string `mapstructure:"key_file"`
	Token            string `mapstructure:"token"`
	GraceTimeSeconds int    `mapstructure:"grace_time_seconds"`
}

// Listeners helps unmarshalling `listeners` config param
type Listeners struct {
	Name string `mapstructure:"name"`
}

// Proxy represents the configuration for proxies in the agent
type Proxy struct {
	HTTP    string   `mapstructure:"http"`
	HTTPS   string   `mapstructure:"https"`
	NoProxy []string `mapstructure:"no_proxy"`
}

// MappingProfile represent a group of mappings
type MappingProfile struct {
	Name     string          `mapstructure:"name"`
	Prefix   string          `mapstructure:"prefix"`
	Mappings []MetricMapping `mapstructure:"mappings"`
}

// MetricMapping represent one mapping rule
type MetricMapping struct {
	Match     string            `mapstructure:"match"`
	MatchType string            `mapstructure:"match_type"`
	Name      string            `mapstructure:"name"`
	Tags      map[string]string `mapstructure:"tags"`
}

func init() {
	osinit()
	// Configure Datadog global configuration
	Datadog = NewConfig("datadog", "DD", strings.NewReplacer(".", "_"))
	// Configuration defaults
	initConfig(Datadog)
}

// initConfig initializes the config defaults on a config
func initConfig(config Config2) {

}

var (
	ddURLs = map[string]interface{}{
		"app.datadoghq.com": nil,
		"app.datadoghq.eu":  nil,
		"app.datad0g.com":   nil,
		"app.datad0g.eu":    nil,
	}
)

// GetProxies returns the proxy settings from the configuration
func GetProxies() *Proxy {
	return proxies
}

// loadProxyFromEnv overrides the proxy settings with environment variables
func loadProxyFromEnv(config Config) {

}

// Load reads configs files and initializes the config module
func Load() error {
	return nil
}

// LoadWithoutSecret reads configs files, initializes the config module without decrypting any secrets
func LoadWithoutSecret() error {
	return nil
}

func findUnknownKeys(config Config) []string {
	return nil
}

func load(config Config, origin string, loadSecret bool) error {

	return nil
}

// ResolveSecrets merges all the secret values from origin into config. Secret values
// are identified by a value of the form "ENC[key]" where key is the secret key.
// See: https://github.com/DataDog/datadog-agent/blob/master/docs/agent/secrets.md
func ResolveSecrets(config Config, origin string) error {

	return nil
}

// Avoid log ingestion breaking because of a newline in the API key
func sanitizeAPIKey(config Config) {
}

// GetMainInfraEndpoint returns the main DD Infra URL defined in the config, based on the value of `site` and `dd_url`
func GetMainInfraEndpoint() string {
	return ""
}

// GetMainEndpoint returns the main DD URL defined in the config, based on `site` and the prefix, or ddURLKey
func GetMainEndpoint(prefix string, ddURLKey string) string {
	return ""
}

// GetMultipleEndpoints returns the api keys per domain specified in the main agent config
func GetMultipleEndpoints() (map[string][]string, error) {
	return nil, nil
}

// getDomainPrefix provides the right prefix for agent X.Y.Z
func getDomainPrefix(app string) string {
	v, _ := version.Agent()
	return fmt.Sprintf("%d-%d-%d-%s.agent", v.Major, v.Minor, v.Patch, app)
}

// AddAgentVersionToDomain prefixes the domain with the agent version: X-Y-Z.domain
func AddAgentVersionToDomain(DDURL string, app string) (string, error) {
	u, err := url.Parse(DDURL)
	if err != nil {
		return "", err
	}

	// we don't udpdate unknown URL (ie: proxy or custom StatsD server)
	if _, found := ddURLs[u.Host]; !found {
		return DDURL, nil
	}

	subdomain := strings.Split(u.Host, ".")[0]
	newSubdomain := getDomainPrefix(app)

	u.Host = strings.Replace(u.Host, subdomain, newSubdomain, 1)
	return u.String(), nil
}

func getMainInfraEndpointWithConfig(config Config) string {
	return GetMainEndpointWithConfig(config, infraURLPrefix, "dd_url")
}

// GetMainEndpointWithConfig implements the logic to extract the DD URL from a config, based on `site` and ddURLKey
func GetMainEndpointWithConfig(config Config, prefix string, ddURLKey string) (resolvedDDURL string) {

	return
}

// getMultipleEndpointsWithConfig implements the logic to extract the api keys per domain from an agent config
func getMultipleEndpointsWithConfig(config Config) (map[string][]string, error) {

	return nil, nil
}

// IsContainerized returns whether the Agent is running on a Docker container
func IsContainerized() bool {
	return os.Getenv("DOCKER_DD_AGENT") != ""
}

// FileUsedDir returns the absolute path to the folder containing the config
// file used to populate the registry
func FileUsedDir() string {
	return filepath.Dir(Datadog.ConfigFileUsed())
}

// GetIPCAddress returns the IPC address or an error if the address is not local
func GetIPCAddress() (string, error) {
	address := Datadog.GetString("ipc_address")
	if address == "localhost" {
		return address, nil
	}
	ip := net.ParseIP(address)
	if ip == nil {
		return "", fmt.Errorf("ipc_address was set to an invalid IP address: %s", address)
	}
	for _, cidr := range []string{
		"127.0.0.0/8", // IPv4 loopback
		"::1/128",     // IPv6 loopback
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			return "", err
		}
		if block.Contains(ip) {
			return address, nil
		}
	}
	return "", fmt.Errorf("ipc_address was set to a non-loopback IP address: %s", address)
}

// GetEnv retrieves the value of the environment variable named by the key,
// or def if the environment variable was not set.
func GetEnv(key, def string) string {
	value, found := os.LookupEnv(key)
	if !found {
		return def
	}
	return value
}

// IsKubernetes returns whether the Agent is running on a kubernetes cluster
func IsKubernetes() bool {
	// Injected by Kubernetes itself
	if os.Getenv("KUBERNETES_SERVICE_PORT") != "" {
		return true
	}
	// support of Datadog environment variable for Kubernetes
	if os.Getenv("KUBERNETES") != "" {
		return true
	}
	return false
}

// AddOverrides provides an externally accessible method for
// overriding config variables.
// This method must be called before Load() to be effective.
func AddOverrides(vars map[string]interface{}) {
	for k, v := range vars {
		overrideVars[k] = v
	}
}

// applyOverrides overrides config variables.
func applyOverrides(config Config) {

}



// setNumWorkers is a helper to set the effective number of workers for
// a given config.
func setNumWorkers(config Config) {

}

// GetDogstatsdMappingProfiles returns mapping profiles used in DogStatsD mapper
func GetDogstatsdMappingProfiles() ([]MappingProfile, error) {
	return nil, nil
}

func getDogstatsdMappingProfilesConfig(config Config) ([]MappingProfile, error) {

	return nil, nil
}
