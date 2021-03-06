// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build docker

package docker

import (
	"fmt"
	"go.uber.org/zap"
	"math"
	"sync"
	"time"

	"github.com/docker/docker/client"

	"github.com/frankhang/doppler/logs/auditor"
	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/pipeline"
	"github.com/frankhang/doppler/logs/restart"
	"github.com/frankhang/doppler/logs/service"
	"github.com/frankhang/doppler/tagger"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/doppler/util/retry"
)

const (
	backoffInitialDuration = 1 * time.Second
	backoffMaxDuration     = 60 * time.Second
)

// A Launcher starts and stops new tailers for every new containers discovered by autodiscovery.
type Launcher struct {
	initRetry          retry.Retrier
	pipelineProvider   pipeline.Provider
	addedSources       chan *config.LogSource
	removedSources     chan *config.LogSource
	addedServices      chan *service.Service
	removedServices    chan *service.Service
	activeSources      []*config.LogSource
	pendingContainers  map[string]*Container
	tailers            map[string]*Tailer
	cli                *client.Client
	registry           auditor.Registry
	stop               chan struct{}
	erroredContainerID chan string
	lock               *sync.Mutex
	collectAllSource   *config.LogSource
	dockerRetrierStop  chan struct{}
}

// NewLauncher returns a new launcher
func NewLauncher(sources *config.LogSources, services *service.Services, pipelineProvider pipeline.Provider, registry auditor.Registry, shouldRetry bool) (*Launcher, error) {
	launcher := &Launcher{
		pipelineProvider:   pipelineProvider,
		tailers:            make(map[string]*Tailer),
		pendingContainers:  make(map[string]*Container),
		registry:           registry,
		stop:               make(chan struct{}),
		erroredContainerID: make(chan string),
		lock:               &sync.Mutex{},
		dockerRetrierStop:  make(chan struct{}, 1),
	}

	if err := launcher.retrySetupInBackground(shouldRetry); err != nil {
		return nil, err
	}

	// FIXME(achntrl): Find a better way of choosing the right launcher
	// between Docker and Kubernetes
	launcher.addedSources = sources.GetAddedForType(config.DockerType)
	launcher.removedSources = sources.GetRemovedForType(config.DockerType)
	launcher.addedServices = services.GetAddedServicesForType(config.DockerType)
	launcher.removedServices = services.GetRemovedServicesForType(config.DockerType)
	return launcher, nil
}

// setup initializes the docker client and the tagger,
// returns an error if it fails.
func (l *Launcher) setup() error {
	var err error
	// create a new docker client
	l.cli, err = NewClient()
	if err != nil {
		return err
	}
	// initialize the tagger
	tagger.Init()
	return nil
}

func (l *Launcher) retrySetupInBackground(shouldRetry bool) error {
	var retryStrategy retry.Strategy
	if shouldRetry {
		retryStrategy = retry.RetryCount
	} else {
		retryStrategy = retry.OneTry
	}
	l.initRetry.SetupRetrier(&retry.Config{
		Name:          "docker Launcher setup",
		AttemptMethod: func() error { return l.setup() },
		Strategy:      retryStrategy,
		RetryCount:    math.MaxInt32,
		RetryDelay:    30 * time.Second,
	})

	if err := l.initRetry.TriggerRetry(); err != nil && err.RetryStatus == retry.PermaFail {
		return err
	}

	go func() {
		for l.initRetry.RetryStatus() == retry.FailWillRetry {
			select {
			case <-time.After(time.Until(l.initRetry.NextRetry())):
				l.initRetry.TriggerRetry()
			case <-l.dockerRetrierStop:
				return
			}
		}
	}()

	return nil
}

// Start starts the Launcher
func (l *Launcher) Start() {
	go l.run()
}

// Stop stops the Launcher and its tailers in parallel,
// this call returns only when all the tailers are stopped.
func (l *Launcher) Stop() {
	l.dockerRetrierStop <- struct{}{}
	l.stop <- struct{}{}
	stopper := restart.NewParallelStopper()
	for _, tailer := range l.tailers {
		stopper.Add(tailer)
		l.removeTailer(tailer.ContainerID)
	}
	stopper.Stop()
}

// run starts and stops new tailers when it receives a new source
// or a new service which is mapped to a container.
func (l *Launcher) run() {
	for {
		select {
		case service := <-l.addedServices:
			// detected a new container running on the host,
			dockerContainer, err := GetContainer(l.cli, service.Identifier)
			if err != nil {
				logutil.BgLogger().Warn("Could not find container with id", zap.Error(err))
				continue
			}
			container := NewContainer(dockerContainer, service)
			source := container.FindSource(l.activeSources)
			switch {
			case source != nil:
				// a source matches with the container, start a new tailer
				l.startTailer(container, source)
			default:
				// no source matches with the container but a matching source may not have been
				// emitted yet or the container may contain an autodiscovery identifier
				// so it's put in a cache until a matching source is found.
				l.pendingContainers[service.Identifier] = container
			}
		case source := <-l.addedSources:
			// detected a new source that has been created either from a configuration file,
			// a docker label or a pod annotation.
			l.activeSources = append(l.activeSources, source)
			pendingContainers := make(map[string]*Container)
			for _, container := range l.pendingContainers {
				if container.IsMatch(source) {
					// found a container matching the new source, start a new tailer
					l.startTailer(container, source)
				} else {
					// keep the container in cache until
					pendingContainers[container.service.Identifier] = container
				}
			}
			// keep the containers that have not found any source yet for next iterations
			l.pendingContainers = pendingContainers
		case source := <-l.removedSources:
			for i, src := range l.activeSources {
				if src == source {
					// no need to stop any tailer here, it will be stopped after receiving a
					// "remove service" event.
					l.activeSources = append(l.activeSources[:i], l.activeSources[i+1:]...)
					break
				}
			}
		case service := <-l.removedServices:
			// detected that a container has been stopped.
			containerID := service.Identifier
			l.stopTailer(containerID)
			delete(l.pendingContainers, containerID)
		case containerID := <-l.erroredContainerID:
			go l.restartTailer(containerID)
		case <-l.stop:
			// no docker container should be tailed anymore
			return
		}
	}
}

// overrideSource create a new source with the image short name if the source is ContainerCollectAll
func (l *Launcher) overrideSource(container *Container, source *config.LogSource) *config.LogSource {
	if source.Name != config.ContainerCollectAll {
		return source
	}

	if l.collectAllSource == nil {
		l.collectAllSource = source
	}

	shortName, err := container.getShortImageName()
	if err != nil {
		containerID := container.service.Identifier
		logutil.BgLogger().Warn(fmt.Sprintf("Could not get short image name for container %v", ShortContainerID(containerID)), zap.Error(err))
		return source
	}

	overridenSource := config.NewLogSource(config.ContainerCollectAll, &config.LogsConfig{
		Type:    config.DockerType,
		Service: shortName,
		Source:  shortName,
	})
	overridenSource.Status = source.Status
	return overridenSource
}

// startTailer starts a new tailer for the container matching with the source.
func (l *Launcher) startTailer(container *Container, source *config.LogSource) {
	containerID := container.service.Identifier
	if _, isTailed := l.tailers[containerID]; isTailed {
		logutil.BgLogger().Warn(fmt.Sprintf("Can't tail twice the same container: %v", ShortContainerID(containerID)))
		return
	}

	// overridenSource == source if the containerCollectAll option is not activated or the container has AD labels
	overridenSource := l.overrideSource(container, source)
	tailer := NewTailer(l.cli, containerID, overridenSource, l.pipelineProvider.NextPipelineChan(), l.erroredContainerID)

	// compute the offset to prevent from missing or duplicating logs
	since, err := Since(l.registry, tailer.Identifier(), container.service.CreationTime)
	if err != nil {
		logutil.BgLogger().Warn(fmt.Sprintf("Could not recover tailing from last committed offset %v", ShortContainerID(containerID)), zap.Error(err))
	}

	// start the tailer
	err = tailer.Start(since)
	if err != nil {
		logutil.BgLogger().Warn(fmt.Sprintf("Could not start tailer %s: %v", containerID), zap.Error(err))
		return
	}
	source.AddInput(containerID)

	// keep the tailer in track to stop it later on
	l.addTailer(containerID, tailer)
}

// stopTailer stops the tailer matching the containerID.
func (l *Launcher) stopTailer(containerID string) {
	if tailer, isTailed := l.tailers[containerID]; isTailed {
		// No-op if the tailer source came from AD
		if l.collectAllSource != nil {
			l.collectAllSource.RemoveInput(containerID)
		}
		go tailer.Stop()
		l.removeTailer(containerID)
	}
}

func (l *Launcher) restartTailer(containerID string) {
	backoffDuration := backoffInitialDuration
	cumulatedBackoff := 0 * time.Second
	var source *config.LogSource

	oldTailer, exists := l.tailers[containerID]
	if exists {
		source = oldTailer.source
		if l.collectAllSource != nil {
			l.collectAllSource.RemoveInput(containerID)
		}
		oldTailer.Stop()
		l.removeTailer(containerID)
	}

	tailer := NewTailer(l.cli, containerID, source, l.pipelineProvider.NextPipelineChan(), l.erroredContainerID)

	// compute the offset to prevent from missing or duplicating logs
	since, err := Since(l.registry, tailer.Identifier(), service.Before)
	if err != nil {
		logutil.BgLogger().Warn(fmt.Sprintf("Could not recover last committed offset for container %v", ShortContainerID(containerID)), zap.Error(err))
	}

	for {
		if backoffDuration > backoffMaxDuration {
			logutil.BgLogger().Warn(fmt.Sprintf("Could not resume tailing container %v", ShortContainerID(containerID)))
			return
		}

		// start the tailer
		err = tailer.Start(since)
		if err != nil {
			logutil.BgLogger().Warn(fmt.Sprintf("Could not start tailer for container %v", ShortContainerID(containerID)), zaperr)
			time.Sleep(backoffDuration)
			cumulatedBackoff += backoffDuration
			backoffDuration *= 2
			continue
		}
		// keep the tailer in track to stop it later on
		l.addTailer(containerID, tailer)
		source.AddInput(containerID)
		return
	}
}

func (l *Launcher) addTailer(containerID string, tailer *Tailer) {
	l.lock.Lock()
	l.tailers[containerID] = tailer
	l.lock.Unlock()
}

func (l *Launcher) removeTailer(containerID string) {
	l.lock.Lock()
	delete(l.tailers, containerID)
	l.lock.Unlock()
}
