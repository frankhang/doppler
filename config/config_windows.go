// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package config

import (
	"path/filepath"

	"github.com/frankhang/doppler/util/winutil"
)

var (
	defaultConfdPath            = "c:\\programdata\\datadog\\conf.d"
	defaultAdditionalChecksPath = "c:\\programdata\\datadog\\checks.d"
	defaultRunPath              = "c:\\programdata\\datadog\\run"
	defaultSyslogURI            = ""
	defaultGuiPort              = 5002
)

// ServiceName is the name that'll be used to register the Agent
const ServiceName = "DatadogAgent"

func osinit() {
	pd, err := winutil.GetProgramDataDir()
	if err == nil {
		defaultConfdPath = filepath.Join(pd, "conf.d")
		defaultAdditionalChecksPath = filepath.Join(pd, "checks.d")
		defaultRunPath = filepath.Join(pd, "run")
	} else {
		winutil.LogEventViewer(ServiceName, 0x8000000F, defaultConfdPath)
	}
}

// NewAssetFs  Should never be called on non-android
func setAssetFs(config Config) {}
