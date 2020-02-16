// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package system

import (
	"github.com/frankhang/doppler/aggregator"
	"github.com/frankhang/doppler/collector/check"
	core "github.com/frankhang/doppler/collector/corechecks"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
)

const uptimeCheckName = "uptime"

// UptimeCheck doesn't need additional fields
type UptimeCheck struct {
	core.CheckBase
}

// Run executes the check
func (c *UptimeCheck) Run() error {
	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	t, err := uptime()
	if err != nil {
		logutil.BgLogger().Error("system.UptimeCheck: could not retrieve uptime", zap.Error(err))
		return err
	}

	sender.Gauge("system.uptime", float64(t), "", nil)
	sender.Commit()

	return nil
}

func uptimeFactory() check.Check {
	return &UptimeCheck{
		CheckBase: core.NewCheckBase(uptimeCheckName),
	}
}

func init() {
	core.RegisterCheck(uptimeCheckName, uptimeFactory)
}
