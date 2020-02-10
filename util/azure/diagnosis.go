// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package azure

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/doppler/util/log"
	"github.com/frankhang/util/errors"
)

func init() {
	diagnosis.Register("Azure Metadata availability", diagnose)
}

// diagnose the azure metadata API availability
func diagnose() error {
	_, err := GetHostAlias()
	if err != nil {
		errors.Log(errors.Trace(err))
		//log.Error(err)
	}
	return err
}
