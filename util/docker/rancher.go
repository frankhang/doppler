// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package docker

import (
	"fmt"
	"net"

	"github.com/frankhang/util/logutil"
)

const rancherIPLabel = "io.rancher.container.ip"

// FindRancherIPInLabels looks for the `io.rancher.container.ip` label and parses it.
// Rancher 1.x containers don't have docker networks as the orchestrator provides its own CNI.
func FindRancherIPInLabels(labels map[string]string) (string, bool) {
	cidr, found := labels[rancherIPLabel]
	if found {
		ipv4Addr, _, err := net.ParseCIDR(cidr)
		if err != nil {
			logutil.BgLogger().Warn(fmt.Sprintf("error while retrieving Rancher IP: %q is not valid", cidr))
			return "", false
		}
		return ipv4Addr.String(), true
	}

	return "", false
}
