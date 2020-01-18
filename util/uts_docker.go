// +build docker

package util

import (
	"fmt"

	"github.com/frankhang/doppler/util/cache"
	"github.com/frankhang/doppler/util/containers"
	"github.com/frankhang/doppler/util/docker"
	"github.com/frankhang/doppler/util/log"
)

// GetAgentUTSMode retrieves from Docker the UTS mode of the Agent container
func GetAgentUTSMode() (containers.UTSMode, error) {
	cacheUTSModeKey := cache.BuildAgentKey("utsMode")
	if cacheUTSMode, found := cache.Cache.Get(cacheUTSModeKey); found {
		return cacheUTSMode.(containers.UTSMode), nil
	}

	log.Debugf("GetAgentUTSMode trying docker")
	utsMode, err := docker.GetAgentContainerUTSMode()
	cache.Cache.Set(cacheUTSModeKey, utsMode, cache.NoExpiration)
	if err != nil {
		return utsMode, fmt.Errorf("could not detect agent UTS mode: %v", err)
	}
	log.Debugf("GetAgentUTSMode: using UTS mode from Docker: %s", utsMode)
	return utsMode, nil
}
