// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package status

import (
	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/metrics"
)

// InitStatus initialize a status builder
func InitStatus(sources *config.LogSources) {
	var isRunning int32 = 1
	endpoints, _ := config.BuildEndpoints()
	Init(&isRunning, endpoints, sources, metrics.LogsExpvars)
}
