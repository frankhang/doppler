// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package providers

import (
	"fmt"
	"go.uber.org/zap"
	"time"

	"github.com/frankhang/doppler/autodiscovery/integration"
	"github.com/frankhang/doppler/autodiscovery/providers/names"
	"github.com/frankhang/doppler/clusteragent/clusterchecks/types"
	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/doppler/util/clusteragent"
	"github.com/frankhang/util/logutil"
)

const defaultGraceDuration = 60 * time.Second

// ClusterChecksConfigProvider implements the ConfigProvider interface
// for the cluster check feature.
type ClusterChecksConfigProvider struct {
	dcaClient      clusteragent.DCAClientInterface
	graceDuration  time.Duration
	heartbeat      time.Time
	lastChange     int64
	nodeName       string
	flushedConfigs bool
}

// NewClusterChecksConfigProvider returns a new ConfigProvider collecting
// cluster check configurations from the cluster-agent.
// Connectivity is not checked at this stage to allow for retries, Collect will do it.
func NewClusterChecksConfigProvider(cfg config.ConfigurationProviders) (ConfigProvider, error) {
	c := &ClusterChecksConfigProvider{
		graceDuration: defaultGraceDuration,
	}

	c.nodeName, _ = util.GetHostname()
	if cfg.GraceTimeSeconds > 0 {
		c.graceDuration = time.Duration(cfg.GraceTimeSeconds) * time.Second
	}

	// Register in the cluster agent as soon as possible
	c.IsUpToDate()

	return c, nil
}

func (c *ClusterChecksConfigProvider) initClient() error {
	dcaClient, err := clusteragent.GetClusterAgentClient()
	if err == nil {
		c.dcaClient = dcaClient
	}
	return err
}

// String returns a string representation of the ClusterChecksConfigProvider
func (c *ClusterChecksConfigProvider) String() string {
	return names.ClusterChecks
}

func (c *ClusterChecksConfigProvider) withinGracePeriod() bool {
	return c.heartbeat.Add(c.graceDuration).After(time.Now())
}

// IsUpToDate queries the cluster-agent to update its status and
// query if new configurations are available
func (c *ClusterChecksConfigProvider) IsUpToDate() (bool, error) {
	if c.dcaClient == nil {
		err := c.initClient()
		if err != nil {
			return false, err
		}
	}

	status := types.NodeStatus{
		LastChange: c.lastChange,
	}

	reply, err := c.dcaClient.PostClusterCheckStatus(c.nodeName, status)
	if err != nil {
		if c.withinGracePeriod() {
			// Return true to keep the configs during the grace period
			logutil.BgLogger().Debug("Catching error during grace period", zap.Error(err))
			return true, nil
		}
		// Return false, the next Collect will flush the configs
		return false, err
	}

	c.heartbeat = time.Now()
	if reply.IsUpToDate {
		logutil.BgLogger().Debug(fmt.Sprintf("Up to date with change %d", c.lastChange))
	} else {
		logutil.BgLogger().Debug(fmt.Sprintf("Not up to date with change %d", c.lastChange))
	}
	return reply.IsUpToDate, nil
}

// Collect retrieves configurations the cluster-agent dispatched to this agent
func (c *ClusterChecksConfigProvider) Collect() ([]integration.Config, error) {
	if c.dcaClient == nil {
		err := c.initClient()
		if err != nil {
			return nil, err
		}
	}

	reply, err := c.dcaClient.GetClusterCheckConfigs(c.nodeName)
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
	c.lastChange = reply.LastChange
	logutil.BgLogger().Debug(fmt.Sprintf("Storing last change %d", c.lastChange))
	return reply.Configs, nil
}

func init() {
	RegisterProvider("clusterchecks", NewClusterChecksConfigProvider)
}
