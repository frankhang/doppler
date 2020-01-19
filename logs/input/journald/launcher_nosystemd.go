// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build !systemd

package journald

import (
	"github.com/frankhang/doppler/logs/auditor"
	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/pipeline"
)

// Launcher is not supported on no systemd environment.
type Launcher struct{}

// NewLauncher returns a new Launcher
func NewLauncher(sources *config.LogSources, pipelineProvider pipeline.Provider, registry auditor.Registry) *Launcher {
	return &Launcher{}
}

// Start does nothing
func (l *Launcher) Start() {}

// Stop does nothing
func (l *Launcher) Stop() {}