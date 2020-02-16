// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build kubelet

package providers

import (
	"github.com/frankhang/doppler/autodiscovery/integration"
	"github.com/frankhang/doppler/autodiscovery/providers/names"
	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/util/clusteragent"
	"github.com/frankhang/doppler/util/kubernetes/kubelet"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
)

// EndpointsChecksConfigProvider implements the ConfigProvider interface
// for the endpoints check feature.
type EndpointsChecksConfigProvider struct {
	dcaClient      clusteragent.DCAClientInterface
	nodeName       string
	flushedConfigs bool
}

// NewEndpointsChecksConfigProvider returns a new ConfigProvider collecting
// endpoints check configurations from the cluster-agent.
// Connectivity is not checked at this stage to allow for retries, Collect will do it.
func NewEndpointsChecksConfigProvider(cfg config.ConfigurationProviders) (ConfigProvider, error) {
	c := &EndpointsChecksConfigProvider{}
	var err error
	c.nodeName, err = getNodename()
	if err != nil {
		logutil.BgLogger().Error("Cannot get node name", zap.Error(err))
		return nil, err
	}
	err = c.initClient()
	if err != nil {
		logutil.BgLogger().Warn("Cannot get dca client", zap.Error(err))
	}
	return c, nil
}

// String returns a string representation of the EndpointsChecksConfigProvider
func (c *EndpointsChecksConfigProvider) String() string {
	return names.EndpointsChecks
}

// IsUpToDate updates the list of AD templates versions in the Agent's cache and checks the list is up to date compared to Kubernetes's data.
func (c *EndpointsChecksConfigProvider) IsUpToDate() (bool, error) {
	return false, nil
}

// Collect retrieves endpoints checks configurations the cluster-agent dispatched to this agent
func (c *EndpointsChecksConfigProvider) Collect() ([]integration.Config, error) {
	if c.dcaClient == nil {
		err := c.initClient()
		if err != nil {
			return nil, err
		}
	}
	reply, err := c.dcaClient.GetEndpointsCheckConfigs(c.nodeName)
	if err != nil {
		if !c.flushedConfigs {
			// On first error after grace period, mask the error once
			// to delete the configurations and de-schedule the checks
			c.flushedConfigs = true
			return nil, nil
		}
		return nil, err
	}
	c.flushedConfigs = false
	return reply.Configs, nil
}

// getNodename retrieves current node name from kubelet.
func getNodename() (string, error) {
	ku, err := kubelet.GetKubeUtil()
	if err != nil {
		logutil.BgLogger().Error("Cannot get kubeUtil object", zap.Error(err))
		return "", err
	}
	return ku.GetNodename()
}

// initClient initializes a cluster agent client.
func (c *EndpointsChecksConfigProvider) initClient() error {
	dcaClient, err := clusteragent.GetClusterAgentClient()
	if err == nil {
		c.dcaClient = dcaClient
	}
	return err
}

func init() {
	RegisterProvider("endpointschecks", NewEndpointsChecksConfigProvider)
}
