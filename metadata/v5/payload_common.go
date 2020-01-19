// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package v5

import (
	"encoding/json"
	"fmt"

	"github.com/frankhang/doppler/metadata/common"
	"github.com/frankhang/doppler/metadata/host"
	"github.com/frankhang/doppler/metadata/resources"
)

// CommonPayload wraps Payload from the common package
type CommonPayload struct {
	common.Payload
}

// HostPayload wraps Payload from the host package
type HostPayload struct {
	host.Payload
}

// ResourcesPayload wraps Payload from the resources package
type ResourcesPayload struct {
	resources.Payload `json:"resources,omitempty"`
}

// MarshalJSON serialization a Payload to JSON
func (p *Payload) MarshalJSON() ([]byte, error) {
	// use an alias to avoid infinite recursion while serializing
	type PayloadAlias Payload

	return json.Marshal((*PayloadAlias)(p))
}

// Marshal not implemented
func (p *Payload) Marshal() ([]byte, error) {
	return nil, fmt.Errorf("V5 Payload serialization is not implemented")
}