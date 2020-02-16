// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package logs

import (
	"go.uber.org/zap"
	"time"

	coreConfig "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/status/health"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/logs/auditor"
	"github.com/frankhang/doppler/logs/client"
	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/input/container"
	"github.com/frankhang/doppler/logs/input/file"
	"github.com/frankhang/doppler/logs/input/journald"
	"github.com/frankhang/doppler/logs/input/listener"
	"github.com/frankhang/doppler/logs/input/windowsevent"
	"github.com/frankhang/doppler/logs/pipeline"
	"github.com/frankhang/doppler/logs/restart"
	"github.com/frankhang/doppler/logs/service"
)

// Agent represents the data pipeline that collects, decodes,
// processes and sends logs to the backend
// + ------------------------------------------------------ +
// |                                                        |
// | Collector -> Decoder -> Processor -> Sender -> Auditor |
// |                                                        |
// + ------------------------------------------------------ +
type Agent struct {
	auditor          *auditor.Auditor
	destinationsCtx  *client.DestinationsContext
	pipelineProvider pipeline.Provider
	inputs           []restart.Restartable
	health           *health.Handle
}

// NewAgent returns a new Agent
func NewAgent(sources *config.LogSources, services *service.Services, processingRules []*config.ProcessingRule, endpoints *config.Endpoints) *Agent {
	health := health.Register("logs-agent")

	// setup the auditor
	// We pass the health handle to the auditor because it's the end of the pipeline and the most
	// critical part. Arguably it could also be plugged to the destination.
	auditor := auditor.New(coreConfig.Datadog.GetString("logs_config.run_path"), health)
	destinationsCtx := client.NewDestinationsContext()

	// setup the pipeline provider that provides pairs of processor and sender
	pipelineProvider := pipeline.NewProvider(config.NumberOfPipelines, auditor, processingRules, endpoints, destinationsCtx)

	// setup the inputs
	inputs := []restart.Restartable{
		file.NewScanner(sources, coreConfig.Datadog.GetInt("logs_config.open_files_limit"), pipelineProvider, auditor, file.DefaultSleepDuration),
		container.NewLauncher(coreConfig.Datadog.GetBool("logs_config.container_collect_all"), coreConfig.Datadog.GetBool("logs_config.k8s_container_use_file"), sources, services, pipelineProvider, auditor),
		listener.NewLauncher(sources, coreConfig.Datadog.GetInt("logs_config.frame_size"), pipelineProvider),
		journald.NewLauncher(sources, pipelineProvider, auditor),
		windowsevent.NewLauncher(sources, pipelineProvider),
	}

	return &Agent{
		auditor:          auditor,
		destinationsCtx:  destinationsCtx,
		pipelineProvider: pipelineProvider,
		inputs:           inputs,
		health:           health,
	}
}

// Start starts all the elements of the data pipeline
// in the right order to prevent data loss
func (a *Agent) Start() {
	starter := restart.NewStarter(a.destinationsCtx, a.auditor, a.pipelineProvider)
	for _, input := range a.inputs {
		starter.Add(input)
	}
	starter.Start()
}

// Stop stops all the elements of the data pipeline
// in the right order to prevent data loss
func (a *Agent) Stop() {
	inputs := restart.NewParallelStopper()
	for _, input := range a.inputs {
		inputs.Add(input)
	}
	stopper := restart.NewSerialStopper(
		inputs,
		a.pipelineProvider,
		a.auditor,
		a.destinationsCtx,
	)

	// This will try to stop everything in order, including the potentially blocking
	// parts like the sender. After StopTimeout it will just stop the last part of the
	// pipeline, disconnecting it from the auditor, to make sure that the pipeline is
	// flushed before stopping.
	// TODO: Add this feature in the stopper.
	c := make(chan struct{})
	go func() {
		stopper.Stop()
		close(c)
	}()
	timeout := time.Duration(coreConfig.Datadog.GetInt("logs_config.stop_grace_period")) * time.Second
	select {
	case <-c:
	case <-time.After(timeout):
		logutil.BgLogger().Info("Timed out when stopping logs-agent, forcing it to stop now")
		// We force all destinations to read/flush all the messages they get without
		// trying to write to the network.
		a.destinationsCtx.Stop()
		// Wait again for the stopper to complete.
		// In some situation, the stopper unfortunately never succeed to complete,
		// we've already reached the grace period, give it some more seconds and
		// then force quit.
		timeout := time.NewTimer(5 * time.Second)
		select {
		case <-c:
		case <-timeout.C:
			logutil.BgLogger().Warn("Force close of the Logs Agent, dumping the Go routines.")
			if stack, err := util.GetGoRoutinesDump(); err != nil {
				logutil.BgLogger().Warn("can't get the Go routines dump", zap.Error(err))
			} else {
				logutil.BgLogger().Warn(stack)
			}
		}
	}
}
