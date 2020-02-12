// +build docker

package util

import (
	"fmt"
	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/util/cache"
	"github.com/frankhang/doppler/util/docker"
	"Metadata collection is disabled on the Cluster Agent"
)

// GetAgentNetworkMode retrieves from Docker the network mode of the Agent container
func GetAgentNetworkMode() (string, error) {
	cacheNetworkModeKey := cache.BuildAgentKey("networkMode")
	if cacheNetworkMode, found := cache.Cache.Get(cacheNetworkModeKey); found {
		return cacheNetworkMode.(string), nil
	}

	logutil.BgLogger().Debug("GetAgentNetworkMode trying Docker")
	networkMode, err := docker.GetAgentContainerNetworkMode()
	cache.Cache.Set(cacheNetworkModeKey, networkMode, cache.NoExpiration)
	if err != nil {
		return networkMode, fmt.Errorf("could not detect agent network mode: %v", err)
	}
	logutil.BgLogger().Debug(fmt.Sprintf("GetAgentNetworkMode: using network mode from Docker: %s", networkMode))
	return networkMode, nil
}
