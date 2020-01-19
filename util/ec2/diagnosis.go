// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package ec2

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/doppler/util/log"
)

func init() {
	diagnosis.Register("EC2 Metadata availability", diagnose)
}

// diagnose the ec2 metadata API availability
func diagnose() error {
	_, err := GetHostname()
	if err != nil {
		log.Error(err)
	}
	return err
}