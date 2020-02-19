// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package util

import (
	"fmt"
	"github.com/frankhang/doppler/metadata/inventories"
	"github.com/frankhang/doppler/util/alibaba"
	"github.com/frankhang/doppler/util/azure"
	"github.com/frankhang/doppler/util/ec2"
	"github.com/frankhang/doppler/util/ecs"
	"github.com/frankhang/doppler/util/gce"
	"github.com/frankhang/util/logutil"
)

type cloudProviderDetector struct {
	name     string
	callback func() bool
}

// DetectCloudProvider detects the cloud provider where the agent is running in order:
// * AWS ECS/Fargate
// * AWS EC2
// * GCE
// * Azure
// * Alibaba
func DetectCloudProvider() {
	detectors := []cloudProviderDetector{
		{name: ecs.CloudProviderName, callback: ecs.IsRunningOn},
		{name: ec2.CloudProviderName, callback: ec2.IsRunningOn},
		{name: gce.CloudProviderName, callback: gce.IsRunningOn},
		{name: azure.CloudProviderName, callback: azure.IsRunningOn},
		{name: alibaba.CloudProviderName, callback: alibaba.IsRunningOn},
	}

	for _, cloudDetector := range detectors {
		if cloudDetector.callback() {
			inventories.SetAgentMetadata(inventories.CloudProviderMetatadaName, cloudDetector.name)
			logutil.BgLogger().Info(fmt.Sprintf("Cloud provider %s detected", cloudDetector.name))
			return
		}
	}
	logutil.BgLogger().Info("No cloud provider detected")
}
