// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build docker

package docker

import (
	"fmt"
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
)

func init() {
	diagnosis.Register("Docker availability", diagnose)
}

// diagnose the docker availability on the system
func diagnose() error {
	_, err := GetDockerUtil()
	if err != nil {
		logutil.BgLogger().Error(string(err))
	} else {
		logutil.BgLogger().Info("successfully connected to docker")
	}

	hostname, err := HostnameProvider()
	if err != nil {
		logutil.BgLogger().Error(fmt.Sprintf("returned hostname %q", hostname), zap.Error(err))
	} else {
		logutil.BgLogger().Info(fmt.Sprintf("successfully got hostname %q from docker", hostname))
	}
	return err
}
