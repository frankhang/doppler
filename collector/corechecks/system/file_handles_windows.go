// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.
// +build windows

package system

import (
	"fmt"
	"github.com/frankhang/doppler/autodiscovery/integration"
	"github.com/frankhang/doppler/collector/check"
	core "github.com/frankhang/doppler/collector/corechecks"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/doppler/util/winutil/pdhutil"
	"go.uber.org/zap"

	"github.com/frankhang/doppler/aggregator"
)

const fileHandlesCheckName = "file_handle"

type fhCheck struct {
	core.CheckBase
	counter *pdhutil.PdhMultiInstanceCounterSet
}

// Run executes the check
func (c *fhCheck) Run() error {

	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}
	vals, err := c.counter.GetAllValues()
	if err != nil {
		logutil.BgLogger().Warn("Error getting handle value", zap.Error(err))
		return err
	}
	val := vals["_Total"]
	logutil.BgLogger().Debug(fmt.Sprintf("Submitting system.fs.file_handles_in_use %v", val))
	sender.Gauge("system.fs.file_handles.in_use", float64(val), "", nil)
	sender.Commit()

	return nil
}

// The check doesn't need configuration
func (c *fhCheck) Configure(data integration.Data, initConfig integration.Data, source string) (err error) {
	if err := c.CommonConfigure(data, source); err != nil {
		return err
	}

	c.counter, err = pdhutil.GetMultiInstanceCounter("Process", "Handle Count", &[]string{"_Total"}, nil)
	return err
}

func fhFactory() check.Check {
	return &fhCheck{
		CheckBase: core.NewCheckBase(fileHandlesCheckName),
	}
}

func init() {
	core.RegisterCheck(fileHandlesCheckName, fhFactory)
}
