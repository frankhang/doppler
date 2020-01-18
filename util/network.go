package util

import (
	"fmt"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"

	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/util/cache"
)

// GetNetworkID retrieves the network_id which can be used to improve network
// connection resolution. This can be configured or detected.  The
// following sources will be queried:
// * configuration
// * GCE
// * EC2
func GetNetworkID() (string, error) {
	cacheNetworkIDKey := cache.BuildAgentKey("networkID")
	if cacheNetworkID, found := cache.Cache.Get(cacheNetworkIDKey); found {
		return cacheNetworkID.(string), nil
	}

	// the the id from configuration
	if networkID := Cfg.NetworkId; networkID != "" {
		cache.Cache.Set(cacheNetworkIDKey, networkID, cache.NoExpiration)
		logutil.BgLogger().Info("GetNetworkID: using configured network", zap.String("id", networkID))
		return networkID, nil
	}

	//log.Debugf("GetNetworkID trying GCE")
	//if networkID, err := gce.GetNetworkID(); err == nil {
	//	cache.Cache.Set(cacheNetworkIDKey, networkID, cache.NoExpiration)
	//	log.Debugf("GetNetworkID: using network ID from GCE metadata: %s", networkID)
	//	return networkID, nil
	//}

	//log.Debugf("GetNetworkID trying EC2")
	//if networkID, err := ec2.GetNetworkID(); err == nil {
	//	cache.Cache.Set(cacheNetworkIDKey, networkID, cache.NoExpiration)
	//	log.Debugf("GetNetworkID: using network ID from EC2 metadata: %s", networkID)
	//	return networkID, nil
	//}

	return "", fmt.Errorf("could not detect network ID")
}
