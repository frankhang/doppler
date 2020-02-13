// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build kubeapiserver,kubelet

package collectors

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/errors"
	"github.com/frankhang/doppler/util/clusteragent"
	"github.com/frankhang/doppler/util/kubernetes/apiserver"
	"github.com/frankhang/doppler/util/kubernetes/kubelet"
	"github.com/frankhang/doppler/util/retry"
)

const (
	kubeMetadataCollectorName = "kube-metadata-collector"
)

type KubeMetadataCollector struct {
	kubeUtil  kubelet.KubeUtilInterface
	apiClient *apiserver.APIClient
	infoOut   chan<- []*TagInfo
	dcaClient clusteragent.DCAClientInterface
	// used to set a custom delay
	lastUpdate time.Time
	updateFreq time.Duration

	clusterAgentEnabled bool
}

// Detect tries to connect to the kubelet and the API Server if the DCA is not used or the DCA.
func (c *KubeMetadataCollector) Detect(out chan<- []*TagInfo) (CollectionMode, error) {
	if config.Datadog.GetBool("kubernetes_collect_metadata_tags") == false {
		logutil.BgLogger().Info("The metadata mapper was configured to be disabled, not collecting metadata for the pods from the API Server")
		return NoCollection, fmt.Errorf("collection disabled by the configuration")
	}

	var err, errDCA error
	c.kubeUtil, err = kubelet.GetKubeUtil()
	if err != nil {
		return NoCollection, err
	}
	// if DCA is enabled and can't communicate with the DCA, let the tagger retry.
	if config.Datadog.GetBool("cluster_agent.enabled") {
		c.clusterAgentEnabled = false
		c.dcaClient, errDCA = clusteragent.GetClusterAgentClient()
		if errDCA != nil {
			logutil.BgLogger().Error("Could not initialise the communication with the cluster agent", zap.Error(errDCA))
			// continue to retry while we can
			if retry.IsErrWillRetry(errDCA) {
				return NoCollection, errDCA
			}
			// we return the permanent fail only if fallback is disabled
			if retry.IsErrPermaFail(errDCA) && !config.Datadog.GetBool("cluster_agent.tagging_fallback") {
				return NoCollection, errDCA
			}
			logutil.BgLogger().Error("Permanent failure in communication with the cluster agent, will fallback to local service mapper")
		} else {
			c.clusterAgentEnabled = true
		}
	}
	// Fallback to local metamapper if DCA not enabled, or in permafail state with fallback enabled.
	if !config.Datadog.GetBool("cluster_agent.enabled") || errDCA != nil {
		c.apiClient, err = apiserver.GetAPIClient()
		if err != nil {
			return NoCollection, err
		}
	}
	c.infoOut = out
	c.updateFreq = time.Duration(config.Datadog.GetInt("kubernetes_metadata_tag_update_freq")) * time.Second
	return PullCollection, nil
}

// Pull implements an additional time constraints to avoid exhausting the kube-apiserver
func (c *KubeMetadataCollector) Pull() error {
	// Time constraints, get the delta in seconds to display it in the logs:
	timeDelta := c.lastUpdate.Add(c.updateFreq).Unix() - time.Now().Unix()
	if timeDelta > 0 {
		logutil.BgLogger().Debug(fmt.Sprintf("skipping, next effective Pull will be in %d seconds", timeDelta))
		return nil
	}

	pods, err := c.kubeUtil.GetLocalPodList()
	if err != nil {
		return err
	}
	if !c.isClusterAgentEnabled() {
		// If the DCA is not used, each agent stores a local cache of the MetadataMap.
		err = c.addToCacheMetadataMapping(pods)
		if err != nil {
			logutil.BgLogger().Debug("Cannot add the metadataMapping to cache", zap.Error(err))
		}
	}
	c.infoOut <- c.getTagInfos(pods)
	c.lastUpdate = time.Now()
	return nil
}

// Fetch fetches tags for a given entity by iterating on the whole podlist and
// the metadataMapper
func (c *KubeMetadataCollector) Fetch(entity string) ([]string, []string, []string, error) {
	var lowCards, orchestratorCards, highCards []string

	pod, err := c.kubeUtil.GetPodForEntityID(entity)
	if err != nil {
		return lowCards, orchestratorCards, highCards, err
	}

	if kubelet.IsPodReady(pod) == false {
		return lowCards, orchestratorCards, highCards, errors.NewNotFound(entity)
	}

	pods := []*kubelet.Pod{pod}
	if !c.isClusterAgentEnabled() {
		// If the DCA is not used, each agent stores a local cache of the MetadataMap.
		err = c.addToCacheMetadataMapping(pods)
		if err != nil {
			logutil.BgLogger().Debug("Cannot add the metadataMapping to cache", zap.Error(err))
		}
	}

	tagInfos := c.getTagInfos(pods)
	c.infoOut <- tagInfos
	for _, info := range tagInfos {
		if info.Entity == entity {
			return info.LowCardTags, info.OrchestratorCardTags, info.HighCardTags, nil
		}
	}
	return lowCards, orchestratorCards, highCards, errors.NewNotFound(entity)
}

func (c *KubeMetadataCollector) isClusterAgentEnabled() bool {
	if c.clusterAgentEnabled && c.dcaClient != nil {
		v := c.dcaClient.Version()
		if v.String() != "0.0.0" { // means not initialized
			return true
		}
	}
	return false
}

func kubernetesFactory() Collector {
	return &KubeMetadataCollector{}
}

func init() {
	registerCollector(kubeMetadataCollectorName, kubernetesFactory, ClusterOrchestrator)
}
