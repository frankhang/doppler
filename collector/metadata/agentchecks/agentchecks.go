// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package agentchecks

import (
	"encoding/json"
	"fmt"
	"go.uber.org/zap"

	"github.com/frankhang/doppler/collector"
	"github.com/frankhang/doppler/collector/runner"
	"github.com/frankhang/doppler/metadata/common"
	"github.com/frankhang/doppler/metadata/externalhost"
	"github.com/frankhang/doppler/metadata/host"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/util/logutil"
)

// GetPayload builds a payload of all the agentchecks metadata
func GetPayload() *Payload {
	agentChecksPayload := ACPayload{}
	hostnameData, _ := util.GetHostnameData()
	hostname := hostnameData.Hostname
	checkStats := runner.GetCheckStats()

	for _, stats := range checkStats {
		for _, s := range stats {
			var status []interface{}
			if s.LastError != "" {
				status = []interface{}{
					s.CheckName, s.CheckName, s.CheckID, "ERROR", s.LastError, "",
				}
			} else if len(s.LastWarnings) != 0 {
				status = []interface{}{
					s.CheckName, s.CheckName, s.CheckID, "WARNING", s.LastWarnings, "",
				}
			} else {
				status = []interface{}{
					s.CheckName, s.CheckName, s.CheckID, "OK", "", "",
				}
			}
			if status != nil {
				agentChecksPayload.AgentChecks = append(agentChecksPayload.AgentChecks, status)
			}
		}
	}

	loaderErrors := collector.GetLoaderErrors()

	for check, errs := range loaderErrors {
		jsonErrs, err := json.Marshal(errs)
		if err != nil {
			logutil.BgLogger().Warn(fmt.Sprintf("Error formatting loader error from check %s", check), zap.Error(err))
		}
		status := []interface{}{
			check, check, "initialization", "ERROR", string(jsonErrs),
		}
		agentChecksPayload.AgentChecks = append(agentChecksPayload.AgentChecks, status)
	}

	// Grab the non agent checks information
	metaPayload := host.GetMeta(hostnameData)
	metaPayload.Hostname = hostname
	cp := common.GetPayload(hostname)
	ehp := externalhost.GetPayload()
	payload := &Payload{
		CommonPayload{*cp},
		MetaPayload{*metaPayload},
		agentChecksPayload,
		ExternalHostPayload{*ehp},
	}

	return payload
}
