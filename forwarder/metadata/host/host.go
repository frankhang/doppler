// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package host

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"path"
	"sync"
	"time"

	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/metadata/common"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/doppler/util/alibaba"
	"github.com/frankhang/doppler/util/cache"
	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/metadata/host/container"
	"github.com/frankhang/doppler/util/azure"
	"github.com/frankhang/doppler/util/cloudfoundry"
	"github.com/frankhang/doppler/util/ec2"
	"github.com/frankhang/doppler/util/gce"
	kubelet "github.com/frankhang/doppler/util/hostname/kubelet"
)

const packageCachePrefix = "host"

// GetPayload builds a metadata payload every time is called.
// Some data is collected only once, some is cached, some is collected at every call.
func GetPayload(hostnameData util.HostnameData) *Payload {
	meta := getMeta(hostnameData)
	meta.Hostname = hostnameData.Hostname

	p := &Payload{
		Os:            osName,
		PythonVersion: GetPythonVersion(),
		SystemStats:   getSystemStats(),
		Meta:          meta,
		HostTags:      getHostTags(),
		ContainerMeta: getContainerMeta(1 * time.Second),
		NetworkMeta:   getNetworkMeta(),
	}

	// Cache the metadata for use in other payloads
	key := buildKey("payload")
	cache.Cache.Set(key, p, cache.NoExpiration)

	return p
}

// GetPayloadFromCache returns the payload from the cache if it exists, otherwise it creates it.
// The metadata reporting should always grab it fresh. Any other uses, e.g. status, should use this
func GetPayloadFromCache(hostnameData util.HostnameData) *Payload {
	key := buildKey("payload")
	if x, found := cache.Cache.Get(key); found {
		return x.(*Payload)
	}
	return GetPayload(hostnameData)
}

// GetMeta grabs the metadata from the cache and returns it,
// if the cache is empty, then it queries the information directly
func GetMeta(hostnameData util.HostnameData) *Meta {
	key := buildKey("meta")
	if x, found := cache.Cache.Get(key); found {
		return x.(*Meta)
	}
	return getMeta(hostnameData)
}

// GetPythonVersion returns the version string as provided by the embedded Python
// interpreter.
func GetPythonVersion() string {
	// retrieve the Python version from the Agent cache
	if x, found := cache.Cache.Get(cache.BuildAgentKey("pythonVersion")); found {
		return x.(string)
	}

	return "n/a"
}

// getHostAliases returns the hostname aliases from different provider
// This should include GCE, Azure, Cloud foundry, kubernetes
func getHostAliases() []string {
	aliases := []string{}

	alibabaAlias, err := alibaba.GetHostAlias()
	if err != nil {
		logutil.BgLogger().Debug("no Alibaba Host Alias", zap.Error(err))
	} else if alibabaAlias != "" {
		aliases = append(aliases, alibabaAlias)
	}

	azureAlias, err := azure.GetHostAlias()
	if err != nil {
		logutil.BgLogger().Debug("no Azure Host Alias", zap.Error(err))
	} else if azureAlias != "" {
		aliases = append(aliases, azureAlias)
	}

	gceAlias, err := gce.GetHostAlias()
	if err != nil {
		logutil.BgLogger().Debug("no GCE Host Alias", zap.Error(err))
	} else {
		aliases = append(aliases, gceAlias)
	}

	cfAliases, err := cloudfoundry.GetHostAliases()
	if err != nil {
		logutil.BgLogger().Debug("no Cloud Foundry Host Alias", zap.Error(err))
	} else if cfAliases != nil {
		aliases = append(aliases, cfAliases...)
	}

	k8sAlias, err := kubelet.GetHostAlias()
	if err != nil {
		logutil.BgLogger().Debug("no Kubernetes Host Alias (through kubelet API)", zap.Error(err))
	} else if k8sAlias != "" {
		aliases = append(aliases, k8sAlias)
	}
	return aliases
}

// getMeta grabs the information and refreshes the cache
func getMeta(hostnameData util.HostnameData) *Meta {
	hostname, _ := os.Hostname()
	tzname, _ := time.Now().Zone()
	ec2Hostname, _ := ec2.GetHostname()
	instanceID, _ := ec2.GetInstanceID()

	var agentHostname string

	if config.Datadog.GetBool("hostname_force_config_as_canonical") &&
		hostnameData.Provider == util.HostnameProviderConfiguration {
		agentHostname = hostnameData.Hostname
	}

	m := &Meta{
		SocketHostname: hostname,
		Timezones:      []string{tzname},
		SocketFqdn:     util.Fqdn(hostname),
		EC2Hostname:    ec2Hostname,
		HostAliases:    getHostAliases(),
		InstanceID:     instanceID,
		AgentHostname:  agentHostname,
	}

	// Cache the metadata for use in other payload
	key := buildKey("meta")
	cache.Cache.Set(key, m, cache.NoExpiration)

	return m
}

func getNetworkMeta() *NetworkMeta {
	nid, err := util.GetNetworkID()
	if err != nil {
		logutil.BgLogger().Info("could not get network metadata", zap.Error(err))
		return nil
	}
	return &NetworkMeta{ID: nid}
}

func getContainerMeta(timeout time.Duration) map[string]string {
	wg := sync.WaitGroup{}
	containerMeta := make(map[string]string)
	// protecting the above map from concurrent access
	mutex := &sync.Mutex{}

	for provider, getMeta := range container.DefaultCatalog {
		wg.Add(1)
		go func(provider string, getMeta container.MetadataProvider) {
			defer wg.Done()
			meta, err := getMeta()
			if err != nil {
				logutil.BgLogger().Debug(fmt.Sprintf("Unable to get %s metadata", provider), zap.Error(err))
				return
			}
			mutex.Lock()
			for k, v := range meta {
				containerMeta[k] = v
			}
			mutex.Unlock()
		}(provider, getMeta)
	}
	// we want to timeout even if the wait group is not done yet
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return containerMeta
	case <-time.After(timeout):
		// in this case the map might be incomplete so return a copy to avoid race
		incompleteMeta := make(map[string]string)
		mutex.Lock()
		for k, v := range containerMeta {
			incompleteMeta[k] = v
		}
		mutex.Unlock()
		return incompleteMeta
	}
}

func buildKey(key string) string {
	return path.Join(common.CachePrefix, packageCachePrefix, key)
}
