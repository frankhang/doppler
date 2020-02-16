// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

// +build !kubeapiserver

package status

import (
	"github.com/frankhang/util/logutil"
)

func getLeaderElectionDetails() map[string]string {
	logutil.BgLogger().Info("Not implemented")
	return nil
}

func getDCAStatus() map[string]string {
	logutil.BgLogger().Info("Not implemented")
	return nil
}
