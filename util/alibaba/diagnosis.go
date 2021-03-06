// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package alibaba

import (
	"github.com/frankhang/doppler/diagnose/diagnosis"
	"github.com/frankhang/util/errors"
)

func init() {
	diagnosis.Register("Alibaba Metadata availability", diagnose)
}

// diagnose the alibaba metadata API availability
func diagnose() error {
	_, err := GetHostAlias()
	if err != nil {
		//log.Error(err)
		errors.Log(errors.Trace(err))
	}
	return err
}
