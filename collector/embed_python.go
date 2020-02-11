// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build python

package collector

import (
	"github.com/frankhang/doppler/collector/python"
	"github.com/frankhang/doppler/config"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
)

func pySetup(paths ...string) (pythonVersion, pythonHome, pythonPath string) {
	if err := python.Initialize(paths...); err != nil {
		logutil.BgLogger().Error("Could not initialize Python", zap.Error(err))
	}
	return python.PythonVersion, python.PythonHome, python.PythonPath
}

func pyPrepareEnv() error {
	if config.Datadog.IsSet("procfs_path") {
		procfsPath := config.Datadog.GetString("procfs_path")
		return python.SetPythonPsutilProcPath(procfsPath)
	}
	return nil
}

func pyTeardown() {
	python.Destroy()
}
