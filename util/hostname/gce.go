// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package hostname

import "github.com/frankhang/doppler/util/gce"

func init() {
	RegisterHostnameProvider("gce", gce.HostnameProvider)
}