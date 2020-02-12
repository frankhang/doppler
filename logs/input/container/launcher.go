// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package container

import (
	"github.com/frankhang/doppler/logs/auditor"
	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/input/docker"
	"github.com/frankhang/doppler/logs/input/kubernetes"
	"github.com/frankhang/doppler/logs/pipeline"
	"github.com/frankhang/doppler/logs/restart"
	"github.com/frankhang/doppler/logs/service"
	"go.uber.org/zap"

	"github.com/frankhang/util/logutil"
)

// NewLauncher returns a new container launcher depending on the environment.
// By default returns a docker launcher if the docker socket is mounted and fallback to
// a kubernetes launcher if '/var/log/pods' is mounted ; this behaviour is reversed when
// collectFromFiles is enabled.
// If none of those volumes are mounted, returns a lazy docker launcher with a retrier to handle the cases
// where docker is started after the agent.
func NewLauncher(collectAll bool, collectFromFiles bool, sources *config.LogSources, services *service.Services, pipelineProvider pipeline.Provider, registry auditor.Registry) restart.Restartable {
	var (
		launcher restart.Restartable
		err      error
	)

	if collectFromFiles {
		launcher, err = kubernetes.NewLauncher(sources, services, collectAll)
		if err == nil {
			logutil.BgLogger().Info("Kubernetes launcher initialized")
			return launcher
		}
		logutil.BgLogger().Info("Could not setup the kubernetes launcher", zap.Error(err))

		launcher, err = docker.NewLauncher(sources, services, pipelineProvider, registry, false)
		if err == nil {
			logutil.BgLogger().Info("Docker launcher initialized")
			return launcher
		}
		logutil.BgLogger().Info("Could not setup the docker launcher", zap.Error(err))
	} else {
		launcher, err = docker.NewLauncher(sources, services, pipelineProvider, registry, false)
		if err == nil {
			logutil.BgLogger().Info("Docker launcher initialized")
			return launcher
		}
		logutil.BgLogger().Info("Could not setup the docker launcher", zap.Error(err))

		launcher, err = kubernetes.NewLauncher(sources, services, collectAll)
		if err == nil {
			logutil.BgLogger().Info("Kubernetes launcher initialized")
			return launcher
		}
		logutil.BgLogger().Info("Could not setup the kubernetes launcher", zap.Error(err))
	}

	launcher, err = docker.NewLauncher(sources, services, pipelineProvider, registry, true)
	if err != nil {
		logutil.BgLogger().Warn("Could not setup the docker launcher. Will not be able to collect container logs", zap.Error(err))
		return nil
	}

	logutil.BgLogger().Info("Container logs won't be collected unless a docker daemon is eventually started")

	return launcher
}
