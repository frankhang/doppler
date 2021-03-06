// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.
// +build !windows

package common

import (
	"go.uber.org/zap"
	"path"

	"github.com/frankhang/doppler/util/cache"

	"github.com/frankhang/util/logutil"
	gopsutilhost "github.com/shirou/gopsutil/host"
)

func getUUID() string {
	key := path.Join(CachePrefix, "uuid")
	if x, found := cache.Cache.Get(key); found {
		return x.(string)
	}

	info, err := gopsutilhost.Info()
	if err != nil {
		// don't cache and return zero value
		logutil.BgLogger().Error("failed to retrieve host info", zap.Error(err))
		return ""
	}
	cache.Cache.Set(key, info.HostID, cache.NoExpiration)
	return info.HostID
}
