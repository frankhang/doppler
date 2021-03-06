// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package clusteragent

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/logutil"
)

func init() {
	diagnosis.Register("Cluster Agent availability", diagnose)
}

func diagnose() error {
	_, err := GetClusterAgentClient()
	if err != nil {
		logutil.BgLogger().Error(err.Error())
	}
	return err
}
