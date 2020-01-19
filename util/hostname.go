// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package util

import (
	"expvar"
	"fmt"
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
	"net"
	"os"
	"runtime"

	"github.com/frankhang/doppler/metadata/inventories"
	"github.com/frankhang/doppler/util/containers"
	"github.com/frankhang/doppler/util/log"

	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/util/cache"
	"github.com/frankhang/doppler/util/ec2"
	"github.com/frankhang/doppler/util/ecs"
	"github.com/frankhang/doppler/util/hostname"
	"github.com/frankhang/doppler/util/hostname/validate"
)

var (
	hostnameExpvars  = expvar.NewMap("hostname")
	hostnameProvider = expvar.String{}
	hostnameErrors   = expvar.Map{}
)

func init() {
	hostnameErrors.Init()
	hostnameExpvars.Set("provider", &hostnameProvider)
	hostnameExpvars.Set("errors", &hostnameErrors)
}

// Fqdn returns the FQDN for the host if any
func Fqdn(hostname string) string {
	addrs, err := net.LookupIP(hostname)
	if err != nil {
		return hostname
	}

	for _, addr := range addrs {
		if ipv4 := addr.To4(); ipv4 != nil {
			ip, err := ipv4.MarshalText()
			if err != nil {
				return hostname
			}
			hosts, err := net.LookupAddr(string(ip))
			if err != nil || len(hosts) == 0 {
				return hostname
			}
			return hosts[0]
		}
	}
	return hostname
}

func setHostnameProvider(name string) {
	hostnameProvider.Set(name)
	inventories.SetAgentMetadata("hostname_source", name)
}

// isOSHostnameUsable returns `false` if it has the certainty that the agent is running
// in a non-root UTS namespace because in that case, the OS hostname characterizes the
// identity of the agent container and not the one of the nodes it is running on.
// There can be some cases where the agent is running in a non-root UTS namespace that are
// not detected by this function (systemd-nspawn containers, manual `unshare -u`…)
// In those uncertain cases, it returns `true`.
func isOSHostnameUsable() (osHostnameUsable bool) {
	// If the agent is not containerized, just skip all this detection logic
	if !IsContainerized() {
		return true
	}

	// Check UTS namespace from docker
	utsMode, err := GetAgentUTSMode()
	if err == nil && (utsMode != containers.HostUTSMode && utsMode != containers.UnknownUTSMode) {
		log.Debug("Agent is running in a docker container without host UTS mode: OS-provided hostnames cannot be used for hostname resolution.")
		return false
	}

	// Check hostNetwork from kubernetes
	// because kubernetes sets UTS namespace to host if and only if hostNetwork = true:
	// https://github.com/kubernetes/kubernetes/blob/cf16e4988f58a5b816385898271e70c3346b9651/pkg/kubelet/dockershim/security_context.go#L203-L205
	hostNetwork, err := isAgentKubeHostNetwork()
	if err == nil && !hostNetwork {
		logutil.BgLogger().Info("Agent is running in a POD without hostNetwork: OS-provided hostnames cannot be used for hostname resolution.")
		return false
	}

	return true
}

// GetHostname retrieves the host name from GetHostnameData
func GetHostname() (string, error) {
	hostnameData, err := GetHostnameData()
	return hostnameData.Hostname, err
}

// HostnameProviderConfiguration is the key for the hostname provider associated to datadog.yaml
const HostnameProviderConfiguration = "configuration"

// HostnameData contains hostname and the hostname provider
type HostnameData struct {
	Hostname string
	Provider string
}

// saveHostnameData creates a HostnameData struct, saves it in the cache under cacheHostnameKey
// and calls setHostnameProvider with the provider if it is not empty.
func saveHostnameData(cacheHostnameKey string, hostname string, provider string) HostnameData {
	hostnameData := HostnameData{Hostname: hostname, Provider: provider}
	cache.Cache.Set(cacheHostnameKey, hostnameData, cache.NoExpiration)
	if provider != "" {
		setHostnameProvider(provider)
	}
	return hostnameData
}

// GetHostnameData retrieves the host name for the Agent and hostname provider, trying to query these
// environments/api, in order:
// * GCE
// * Docker
// * kubernetes
// * os
// * EC2
func GetHostnameData() (HostnameData, error) {
	cacheHostnameKey := cache.BuildAgentKey("hostname")
	if cacheHostname, found := cache.Cache.Get(cacheHostnameKey); found {
		return cacheHostname.(HostnameData), nil
	}

	var hostName string
	var err error
	var provider string

	// try the name provided in the configuration file
	configName := Cfg.HostName
	err = validate.ValidHostname(configName)
	if err == nil {
		hostnameData := saveHostnameData(cacheHostnameKey, configName, HostnameProviderConfiguration)
		if !isHostnameCanonicalForIntake(configName) && !Cfg.HostnameForceConfigAsCanonical {
			logutil.BgLogger().Warn("Hostname defined in configuration will not be used as the in-app hostname. For more information: https://dtdg.co/agent-hostname-force-config-as-canonical", zap.String("hostname", configName))
		}
		return hostnameData, nil
	}

	expErr := new(expvar.String)
	expErr.Set(err.Error())
	hostnameErrors.Set("configuration/environment", expErr)

	logutil.BgLogger().Info("Unable to get the hostname from the config file", zap.Error(err))
	logutil.BgLogger().Info("Trying to determine a reliable host name automatically...")

	// if fargate we strip the hostname
	if ecs.IsFargateInstance() {
		hostnameData := saveHostnameData(cacheHostnameKey, "", "")
		return hostnameData, nil
	}


	// FQDN
	var fqdn string
	canUseOSHostname := isOSHostnameUsable()
	if canUseOSHostname {
		logutil.BgLogger().Info("GetHostname trying FQDN/`hostname -f`...")
		fqdn, err = getSystemFQDN()
		if Cfg.HostNameFqdn && err == nil {
			hostName = fqdn
			provider = "fqdn"
		} else {
			if err != nil {
				expErr := new(expvar.String)
				expErr.Set(err.Error())
				hostnameErrors.Set("fqdn", expErr)
			}
			logutil.BgLogger().Info("Unable to get FQDN from system: ", zap.Error(err))
		}
	}

	isContainerized, containerName := getContainerHostname()
	if isContainerized {
		if containerName != "" {
			hostName = containerName
			provider = "container"
		} else {
			expErr := new(expvar.String)
			expErr.Set("Unable to get hostname from container API")
			hostnameErrors.Set("container", expErr)
		}
	}

	if canUseOSHostname && hostName == "" {
		// os
		logutil.BgLogger().Info("GetHostname trying os...")
		systemName, err := os.Hostname()
		if err == nil {
			hostName = systemName
			provider = "os"
		} else {
			expErr := new(expvar.String)
			expErr.Set(err.Error())
			hostnameErrors.Set("os", expErr)
			logutil.BgLogger().Info("Unable to get hostname from OS", zap.Error(err))
		}
	}

	/* at this point we've either the hostname from the os or an empty string */

	// We use the instance id if we're on an ECS cluster or we're on EC2
	// and the hostname is one of the default ones
	if getEC2Hostname, found := hostname.ProviderCatalog["ec2"]; found {
		logutil.BgLogger().Info("GetHostname trying EC2 metadata...")
		if ecs.IsECSInstance() || ec2.IsDefaultHostname(hostName) {
			instanceID, err := getEC2Hostname()
			if err == nil {
				err = validate.ValidHostname(instanceID)
				if err == nil {
					hostName = instanceID
					provider = "aws"
				} else {
					expErr := new(expvar.String)
					expErr.Set(err.Error())
					hostnameErrors.Set("aws", expErr)
					logutil.BgLogger().Info("EC2 instance ID is not a valid hostname: ", zap.Error(err))
				}
			} else {
				expErr := new(expvar.String)
				expErr.Set(err.Error())
				hostnameErrors.Set("aws", expErr)
				logutil.BgLogger().Info("Unable to determine hostname from EC2", zap.Error(err))
			}
		} else {
			err := fmt.Errorf("not retrieving hostname from AWS: the host is not an ECS instance, and other providers already retrieve non-default hostnames")
			logutil.BgLogger().Info(err.Error())
			expErr := new(expvar.String)
			expErr.Set(err.Error())
			hostnameErrors.Set("aws", expErr)
		}
	}

	h, err := os.Hostname()
	if err == nil && !Cfg.HostNameFqdn && fqdn != "" && hostName == h && h != fqdn {
		if runtime.GOOS != "windows" {
			// REMOVEME: This should be removed when the default `hostname_fqdn` is set to true
			logutil.BgLogger().Warn(fmt.Sprintf("DEPRECATION NOTICE: The agent resolved your hostname as '%s'. However in a future version, it will be resolved as '%s' by default. To enable the future behavior, please enable the `hostname_fqdn` flag in the configuration. For more information: https://dtdg.co/flag-hostname-fqdn", h, fqdn))
		} else { // OS is Windows
			logutil.BgLogger().Warn(fmt.Sprintf("The agent resolved your hostname as '%s', and will be reported this way to maintain compatibility with version 5. To enable reporting as '%s', please enable the `hostname_fqdn` flag in the configuration. For more information: https://dtdg.co/flag-hostname-fqdn", h, fqdn))
		}
	}

	// If at this point we don't have a name, bail out
	if hostName == "" {
		err = fmt.Errorf("unable to reliably determine the host name. You can define one in the agent config file or in your hosts file")
	} else {
		// we got a hostname, residual errors are irrelevant now
		err = nil
	}

	hostnameData := saveHostnameData(cacheHostnameKey, hostName, provider)
	if err != nil {
		expErr := new(expvar.String)
		expErr.Set(fmt.Sprintf(err.Error()))
		hostnameErrors.Set("all", expErr)
	}
	return hostnameData, errors.Trace(err)
}

// isHostnameCanonicalForIntake returns true if the intake will use the hostname as canonical hostname.
func isHostnameCanonicalForIntake(hostname string) bool {
	// Intake uses instance id for ec2 default hostname except for Windows.
	if ec2.IsDefaultHostnameForIntake(hostname) {
		_, err := ec2.GetInstanceID()
		return err != nil
	}
	return true
}