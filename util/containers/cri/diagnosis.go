// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build cri

package cri

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/logutil"
)

func init() {
	diagnosis.Register("CRI availability", diagnose)
}

// diagnose the CRI socket connectivity
func diagnose() error {
	_, err := GetUtil()
	if err != nil {
		logutil.BgLogger().Error(string(err))
	}
	return err
}
