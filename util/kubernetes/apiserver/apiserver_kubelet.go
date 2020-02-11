// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2018-2020 Datadog, Inc.

// +build kubeapiserver,kubelet

package apiserver

import (
	"fmt"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/frankhang/doppler/util/cache"
	"github.com/frankhang/doppler/util/kubernetes/kubelet"
	"github.com/frankhang/util/logutil"
)

// NodeMetadataMapping only fetch the endpoints from Kubernetes apiserver and add the metadataMapper of the
// node to the cache
// Only called when the node agent computes the metadata mapper locally and does not rely on the DCA.
func (c *APIClient) NodeMetadataMapping(nodeName string, pods []*kubelet.Pod) error {
	endpointList, err := c.Cl.CoreV1().Endpoints("").List(metav1.ListOptions{TimeoutSeconds: &c.timeoutSeconds})
	if err != nil {
		logutil.BgLogger().Error("Could not collect endpoints from the API Server", zap.Error(err))
		return err
	}
	if endpointList.Items == nil {
		logutil.BgLogger().Debug("No endpoints collected from the API server")
		return nil
	}
	logutil.BgLogger().Debug("Successfully collected endpoints")

	var node v1.Node
	var nodeList v1.NodeList
	node.Name = nodeName

	nodeList.Items = append(nodeList.Items, node)

	processKubeServices(&nodeList, pods, endpointList)
	return nil
}

// processKubeServices adds services to the metadataMapper cache, pointer parameters must be non nil
func processKubeServices(nodeList *v1.NodeList, pods []*kubelet.Pod, endpointList *v1.EndpointsList) {
	if nodeList.Items == nil || len(pods) == 0 || endpointList.Items == nil {
		return
	}
	logutil.BgLogger().Debug(fmt.Sprintf("Identified: %d node, %d pod, %d endpoints", len(nodeList.Items), len(pods), len(endpointList.Items)))
	for _, node := range nodeList.Items {
		nodeName := node.Name
		nodeNameCacheKey := cache.BuildAgentKey(metadataMapperCachePrefix, nodeName)
		freshness := cache.BuildAgentKey(metadataMapperCachePrefix, nodeName, "freshness")

		cacheData, found := cache.Cache.Get(nodeNameCacheKey)        // We get the old one with the dead pods. if diff reset metabundle and deleted key. Then compute again.
		freshnessCache, freshnessFound := cache.Cache.Get(freshness) // if expired, freshness not found deal with that

		newMetaBundle := newMetadataMapperBundle()
		if !found {
			cache.Cache.Set(freshness, len(pods), metadataMapExpire)
		}

		// We want to churn the cache every `metadataMapExpire` and if the number of entries varies between 2 runs..
		// If a pod is killed and rescheduled during a run, we will only keep the old entry for another run, which is acceptable.
		if found && freshnessCache != len(pods) || !freshnessFound {
			cache.Cache.Set(freshness, len(pods), metadataMapExpire)
			logutil.BgLogger().Debug(fmt.Sprintf("Refreshing cache for %s", nodeNameCacheKey))
		} else {
			oldMetadataBundle, ok := cacheData.(*metadataMapperBundle)
			if ok {
				newMetaBundle.DeepCopy(oldMetadataBundle)
			}
		}

		if err := newMetaBundle.mapServices(nodeName, pods, *endpointList); err != nil {
			logutil.BgLogger().Error(fmt.Sprintf("Could not map the services on node %s", node.Name), zap.Error(err))
			continue
		}
		cache.Cache.Set(nodeNameCacheKey, newMetaBundle, metadataMapExpire)
	}
}
