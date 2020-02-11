// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017-2020 Datadog, Inc.

// +build docker

package ecs

import (
	"fmt"
	"go.uber.org/zap"
	"net"
	"time"

	"github.com/frankhang/doppler/util/containers"
	"github.com/frankhang/doppler/util/containers/metrics"
	"github.com/frankhang/doppler/util/ecs/metadata"
	"github.com/frankhang/util/logutil"

	v2 "github.com/frankhang/doppler/util/ecs/metadata/v2"
)

// ListContainersInCurrentTask returns internal container representations (with
// their metrics) for the current task by collecting that information from the
// ECS metadata v2 API.
func ListContainersInCurrentTask() ([]*containers.Container, error) {
	var cList []*containers.Container

	task, err := metadata.V2().GetTask()
	if err != nil || len(task.Containers) == 0 {
		logutil.BgLogger().Error("Unable to get the container list from ecs")
		return cList, err
	}
	for _, c := range task.Containers {
		cList = append(cList, convertMetaV2Container(c))
	}

	err = UpdateContainerMetrics(cList)
	return cList, err
}

// UpdateContainerMetrics updates performance metrics for a list of internal
// container representations based on stats collected from the ECS metadata v2 API
func UpdateContainerMetrics(cList []*containers.Container) error {
	for _, ctr := range cList {
		stats, err := metadata.V2().GetContainerStats(ctr.ID)
		if err != nil {
			logutil.BgLogger().Debug("Unable to get stats from ECS for container", zap.String("id", ctr.ID), zap.Error(err))
			continue
		}

		stats.IO.ReadBytes = sumStats(stats.IO.BytesPerDeviceAndKind, "Read")
		stats.IO.WriteBytes = sumStats(stats.IO.BytesPerDeviceAndKind, "Write")

		// TODO: add metrics - complete for https://github.com/DataDog/datadog-process-agent/blob/970729924e6b2b6fe3a912b62657c297621723cc/checks/container_rt.go#L110-L128
		// start with a hack (translate ecs stats to docker cgroup stuff)
		// then support ecs stats natively
		cpu, mem, io, memLimit := convertMetaV2ContainerStats(stats)
		ctr.CPU = &cpu
		ctr.Memory = &mem
		ctr.IO = &io

		if ctr.MemLimit == 0 {
			ctr.MemLimit = memLimit
		}
	}
	return nil
}

// convertMetaV2Container returns an internal container representation from an
// ECS metadata v2 container object.
func convertMetaV2Container(c v2.Container) *containers.Container {
	container := &containers.Container{
		Type:        "ECS",
		ID:          c.DockerID,
		EntityID:    containers.BuildTaggerEntityName(c.DockerID),
		Name:        c.DockerName,
		Image:       c.Image,
		ImageID:     c.ImageID,
		AddressList: parseContainerNetworkAddresses(c.Ports, c.Networks, c.DockerName),
	}

	createdAt, err := time.Parse(time.RFC3339, c.CreatedAt)
	if err != nil {
		logutil.BgLogger().Error("Unable to determine creation time for container", zap.Strings("id", c.DockerID), zap.Error(err))
	} else {
		container.Created = createdAt.Unix()
	}
	startedAt, err := time.Parse(time.RFC3339, c.StartedAt)
	if err != nil {
		logutil.BgLogger().Error("Unable to determine creation time for container", zap.String("id", c.DockerID), zap.Error(err))
	} else {
		container.StartedAt = startedAt.Unix()
	}

	if l, found := c.Limits["cpu"]; found && l > 0 {
		container.CPULimit = float64(l)
	} else {
		container.CPULimit = 100
	}
	if l, found := c.Limits["memory"]; found && l > 0 {
		container.MemLimit = l
	}

	return container
}

// convertMetaV2Container returns internal metrics representations from an ECS
// metadata v2 container stats object.
func convertMetaV2ContainerStats(s *v2.ContainerStats) (cpu metrics.CgroupTimesStat, mem metrics.CgroupMemStat, io metrics.CgroupIOStat, memLimit uint64) {
	// CPU
	cpu.User = s.CPU.Usage.Usermode
	cpu.System = s.CPU.Usage.Kernelmode
	cpu.SystemUsage = s.CPU.System

	// Memory
	mem.Cache = s.Memory.Details.Cache
	mem.MemUsageInBytes = s.Memory.Usage
	mem.Pgfault = s.Memory.Details.PgFault
	mem.RSS = s.Memory.Details.RSS
	memLimit = s.Memory.Limit

	// IO
	io.ReadBytes = s.IO.ReadBytes
	io.WriteBytes = s.IO.WriteBytes

	return
}

// parseContainerNetworkAddresses converts ECS container ports
// and networks into a list of NetworkAddress
func parseContainerNetworkAddresses(ports []v2.Port, networks []v2.Network, container string) []containers.NetworkAddress {
	addrList := []containers.NetworkAddress{}
	if networks == nil {
		logutil.BgLogger().Debug("No network settings available in ECS metadata")
		return addrList
	}
	for _, network := range networks {
		for _, addr := range network.IPv4Addresses { // one-element list
			IP := net.ParseIP(addr)
			if IP == nil {
				logutil.BgLogger().Warn(fmt.Sprintf("Unable to parse IP: %v for container: %s", addr, container))
				continue
			}
			if len(ports) > 0 {
				// Ports is not nil, get ports and protocols
				for _, port := range ports {
					addrList = append(addrList, containers.NetworkAddress{
						IP:       IP,
						Port:     int(port.ContainerPort),
						Protocol: port.Protocol,
					})
				}
			} else {
				// Ports is nil (omitted by the ecs api if there are no ports exposed).
				// Keep the container IP anyway.
				addrList = append(addrList, containers.NetworkAddress{
					IP: IP,
				})
			}
		}
	}
	return addrList
}

// sumStats adds up values across devices for an operation kind.
func sumStats(ops []v2.OPStat, kind string) uint64 {
	var res uint64
	for _, op := range ops {
		if op.Kind == kind {
			res += op.Value
		}
	}
	return res
}
