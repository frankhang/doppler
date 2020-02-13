// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build kubeapiserver

package apiserver

import (
	"fmt"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
)

// DetectOpenShiftAPILevel looks at known endpoints to detect if OpenShift
// APIs are available on this apiserver. OpenShift transitioned from a
// non-standard `/oapi` URL prefix to standard api groups under the `/apis`
// prefix in 3.6. Detecting both, with a preference for the new prefix.
func (c *APIClient) DetectOpenShiftAPILevel() OpenShiftAPILevel {
	err := c.Cl.CoreV1().RESTClient().Get().AbsPath("/apis/quota.openshift.io").Do().Error()
	if err == nil {
		logutil.BgLogger().Debug(fmt.Sprintf("Found %s", OpenShiftAPIGroup))
		return OpenShiftAPIGroup
	}
	logutil.BgLogger().Debug(fmt.Sprintf("Cannot access %s", OpenShiftAPIGroup), zap.Error(err))

	err = c.Cl.CoreV1().RESTClient().Get().AbsPath("/oapi").Do().Error()
	if err == nil {
		logutil.BgLogger().Debug(fmt.Sprintf("Found %s", OpenShiftOAPI))
		return OpenShiftOAPI
	}
	logutil.BgLogger().Debug(fmt.Sprintf("Cannot access %s", OpenShiftOAPI), zap.Error(err))

	// Fallback to NotOpenShift
	return NotOpenShift
}
