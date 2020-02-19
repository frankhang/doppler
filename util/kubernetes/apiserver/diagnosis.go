// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build kubeapiserver

package apiserver

import (
	"fmt"
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/logutil"
)

func init() {
	diagnosis.Register("Kubernetes API Server availability", diagnose)
}

// diagnose the API server availability
func diagnose() error {
	isConnectVerbose = true
	c, err := GetAPIClient()
	isConnectVerbose = false
	if err != nil {
		logutil.BgLogger().Error(err.Error())
		return err
	}
	logutil.BgLogger().Info(fmt.Sprintf("Detecting OpenShift APIs: %s available", c.DetectOpenShiftAPILevel()))
	return nil
}
