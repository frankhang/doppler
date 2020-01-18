// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build linux windows darwin

package v5

import (
	"github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/metadata/common"
	"github.com/frankhang/doppler/metadata/gohai"
	"github.com/frankhang/doppler/metadata/host"
	"github.com/frankhang/doppler/metadata/resources"
	"github.com/frankhang/doppler/util"
)

// GetPayload returns the complete metadata payload as seen in Agent v5
func GetPayload(hostnameData util.HostnameData) *Payload {
	cp := common.GetPayload(hostnameData.Hostname)
	hp := host.GetPayload(hostnameData)
	rp := resources.GetPayload(hostnameData.Hostname)

	p := &Payload{
		CommonPayload: CommonPayload{*cp},
		HostPayload:   HostPayload{*hp},
	}

	if rp != nil {
		p.ResourcesPayload = ResourcesPayload{*rp}
	}

	if config.Datadog.GetBool("enable_gohai") {
		p.GohaiPayload = GohaiPayload{MarshalledGohaiPayload{*gohai.GetPayload()}}
	}

	return p
}
