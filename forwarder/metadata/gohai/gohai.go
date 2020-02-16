// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package gohai

import (
	"github.com/DataDog/gohai/cpu"
	"github.com/DataDog/gohai/filesystem"
	"github.com/DataDog/gohai/memory"
	"github.com/DataDog/gohai/network"
	"github.com/DataDog/gohai/platform"
	"go.uber.org/zap"

	"github.com/frankhang/doppler/config"
	"github.com/frankhang/util/logutil"
)

// GetPayload builds a payload of every metadata collected with gohai except processes metadata.
func GetPayload() *Payload {
	return &Payload{
		Gohai: getGohaiInfo(),
	}
}

func getGohaiInfo() *gohai {
	res := new(gohai)

	cpuPayload, err := new(cpu.Cpu).Collect()
	if err == nil {
		res.CPU = cpuPayload
	} else {
		logutil.BgLogger().Error("Failed to retrieve cpu metadata", zap.Error(err))
	}

	fileSystemPayload, err := new(filesystem.FileSystem).Collect()
	if err == nil {
		res.FileSystem = fileSystemPayload
	} else {
		logutil.BgLogger().Error("Failed to retrieve filesystem metadata", zap.Error(err))
	}

	memoryPayload, err := new(memory.Memory).Collect()
	if err == nil {
		res.Memory = memoryPayload
	} else {
		logutil.BgLogger().Error("Failed to retrieve memory metadata", zap.Error(err))
	}

	if !config.IsContainerized() {
		networkPayload, err := new(network.Network).Collect()
		if err == nil {
			res.Network = networkPayload
		} else {
			logutil.BgLogger().Error("Failed to retrieve network metadata", zap.Error(err))
		}
	}

	platformPayload, err := new(platform.Platform).Collect()
	if err == nil {
		res.Platform = platformPayload
	} else {
		logutil.BgLogger().Error("Failed to retrieve platform metadata", zap.Error(err))
	}

	return res
}
