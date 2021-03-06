// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package gce

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/errors"
)

func init() {
	diagnosis.Register("GCE Metadata availability", diagnose)
}

// diagnose the GCE metadata API availability
func diagnose() error {
	_, err := GetHostname()
	if err != nil {
		errors.Log(errors.Trace(err))
		//log.Error(err)
	}
	return err
}
