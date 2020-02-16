// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package serializer

import (
	"encoding/json"
	"expvar"
	"fmt"
	"github.com/frankhang/util/logutil"
	"net/http"
	"regexp"

	"github.com/frankhang/doppler/config"
	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/forwarder"
	"github.com/frankhang/doppler/serializer/jsonstream"
	"github.com/frankhang/doppler/serializer/marshaler"
	"github.com/frankhang/doppler/serializer/split"
	"github.com/frankhang/doppler/util/compression"
	"github.com/frankhang/util/logutil"
)

const (
	protobufContentType                         = "application/x-protobuf"
	jsonContentType                             = "application/json"
	payloadVersionHTTPHeader                    = "DD-Agent-Payload"
	apiKeyReplacement                           = "\"apiKey\":\"*************************$1"
	maxItemCountForCreateMarshalersBySourceType = 100
)

var (
	// AgentPayloadVersion is the versions of the agent-payload repository
	// used to serialize to protobuf
	AgentPayloadVersion string

	jsonExtraHeaders                    http.Header
	protobufExtraHeaders                http.Header
	jsonExtraHeadersWithCompression     http.Header
	protobufExtraHeadersWithCompression http.Header

	expvars                                 = expvar.NewMap("serializer")
	expvarsSendEventsErrItemTooBigs         = expvar.Int{}
	expvarsSendEventsErrItemTooBigsFallback = expvar.Int{}
)

var apiKeyRegExp = regexp.MustCompile("\"apiKey\":\"*\\w+(\\w{5})")

func init() {
	expvars.Set("SendEventsErrItemTooBigs", &expvarsSendEventsErrItemTooBigs)
	expvars.Set("SendEventsErrItemTooBigsFallback", &expvarsSendEventsErrItemTooBigsFallback)
	initExtraHeaders()
}

// initExtraHeaders initializes the global extraHeaders variables.
// Not part of the `init` function body to ease testing
func initExtraHeaders() {
	jsonExtraHeaders = make(http.Header)
	jsonExtraHeaders.Set("Content-Type", jsonContentType)

	jsonExtraHeadersWithCompression = make(http.Header)
	for k := range jsonExtraHeaders {
		jsonExtraHeadersWithCompression.Set(k, jsonExtraHeaders.Get(k))
	}

	protobufExtraHeaders = make(http.Header)
	protobufExtraHeaders.Set("Content-Type", protobufContentType)
	protobufExtraHeaders.Set(payloadVersionHTTPHeader, AgentPayloadVersion)

	protobufExtraHeadersWithCompression = make(http.Header)
	for k := range protobufExtraHeaders {
		protobufExtraHeadersWithCompression.Set(k, protobufExtraHeaders.Get(k))
	}

	if compression.ContentEncoding != "" {
		jsonExtraHeadersWithCompression.Set("Content-Encoding", compression.ContentEncoding)
		protobufExtraHeadersWithCompression.Set("Content-Encoding", compression.ContentEncoding)
	}
}

// EventsStreamJSONMarshaler handles two serialization logics.
type EventsStreamJSONMarshaler interface {
	marshaler.Marshaler

	// Create a single marshaler.
	CreateSingleMarshaler() marshaler.StreamJSONMarshaler

	// If the single marshaler cannot serialize, use smaller marshalers.
	CreateMarshalersBySourceType() []marshaler.StreamJSONMarshaler
}

// MetricSerializer represents the interface of method needed by the aggregator to serialize its data
type MetricSerializer interface {
	SendEvents(e EventsStreamJSONMarshaler) error
	SendServiceChecks(sc marshaler.StreamJSONMarshaler) error
	SendSeries(series marshaler.StreamJSONMarshaler) error
	SendSketch(sketches marshaler.Marshaler) error
	SendMetadata(m marshaler.Marshaler) error
	SendJSONToV1Intake(data interface{}) error
}

// Serializer serializes metrics to the correct format and routes the payloads to the correct endpoint in the Forwarder
type Serializer struct {
	Forwarder forwarder.Forwarder

	seriesPayloadBuilder *jsonstream.PayloadBuilder

	// Those variables allow users to blacklist any kind of payload
	// from being sent by the agent. This was introduced for
	// environment where, for example, events or serviceChecks
	// might collect data considered too sensitive (database IP and
	// such). By default every kind of payload is enabled since
	// almost every user won't fall into this use case.
	enableEvents                  bool
	enableSeries                  bool
	enableServiceChecks           bool
	enableSketches                bool
	enableJSONToV1Intake          bool
	enableJSONStream              bool
	enableServiceChecksJSONStream bool
	enableEventsJSONStream        bool
}

// NewSerializer returns a new Serializer initialized
func NewSerializer(forwarder forwarder.Forwarder) *Serializer {
	s := &Serializer{
		Forwarder:                     forwarder,
		seriesPayloadBuilder:          jsonstream.NewPayloadBuilder(),
		enableEvents:                  Cfg.EnablePayloadsEvents,
		enableSeries:                  Cfg.EnablePayloadsSeries,
		enableServiceChecks:           Cfg.EnablePayloadsServiceChecks,
		enableSketches:                Cfg.EnablePayloadsSketches,
		enableJSONToV1Intake:          Cfg.EnablePayloadsJsonToV1Intake,
		enableJSONStream:              jsonstream.Available && config.Datadog.GetBool("enable_stream_payload_serialization"),
		enableServiceChecksJSONStream: jsonstream.Available && config.Datadog.GetBool("enable_service_checks_stream_payload_serialization"),
		enableEventsJSONStream:        jsonstream.Available && config.Datadog.GetBool("enable_events_stream_payload_serialization"),
	}

	if !s.enableEvents {
		logutil.BgLogger().Warn("event payloads are disabled: all events will be dropped")
	}
	if !s.enableSeries {
		logutil.BgLogger().Warn("series payloads are disabled: all series will be dropped")
	}
	if !s.enableServiceChecks {
		logutil.BgLogger().Warn("service_checks payloads are disabled: all service_checks will be dropped")
	}
	if !s.enableSketches {
		logutil.BgLogger().Warn("sketches payloads are disabled: all sketches will be dropped")
	}
	if !s.enableJSONToV1Intake {
		logutil.BgLogger().Warn("JSON to V1 intake is disabled: all payloads to that endpoint will be dropped")
	}

	return s
}

func (s Serializer) serializePayload(payload marshaler.Marshaler, compress bool, useV1API bool) (forwarder.Payloads, http.Header, error) {
	var marshalType split.MarshalType
	var extraHeaders http.Header

	if useV1API {
		marshalType = split.MarshalJSON
		if compress {
			extraHeaders = jsonExtraHeadersWithCompression
		} else {
			extraHeaders = jsonExtraHeaders
		}
	} else {
		marshalType = split.Marshal
		if compress {
			extraHeaders = protobufExtraHeadersWithCompression
		} else {
			extraHeaders = protobufExtraHeaders
		}
	}

	payloads, err := split.Payloads(payload, compress, marshalType)

	if err != nil {
		return nil, nil, fmt.Errorf("could not split payload into small enough chunks: %s", err)
	}

	return payloads, extraHeaders, nil
}

func (s Serializer) serializeStreamablePayload(payload marshaler.StreamJSONMarshaler, policy jsonstream.OnErrItemTooBigPolicy) (forwarder.Payloads, http.Header, error) {
	payloads, err := s.seriesPayloadBuilder.BuildWithOnErrItemTooBigPolicy(payload, policy)
	return payloads, jsonExtraHeadersWithCompression, err
}

// As events are gathered by SourceType, the serialization logic is more complex than for the other serializations.
// We first try to use PayloadBuilder where a single item is the list of all events for the same source type.

// This method may lead to item than can be too big to be serialized. In this case we try the following method.
// If the count of source type is less than maxItemCountForCreateMarshalersBySourceType then we use a
// of PayloadBuilder for each source type where an item is a single event. We limit to maxItemCountForCreateMarshalersBySourceType
// for performance reasons.
//
// If none of the previous methods work, we fallback to the old serialization method (Serializer.serializePayload).
func (s Serializer) serializeEventsStreamJSONMarshalerPayload(
	eventsStreamJSONMarshaler EventsStreamJSONMarshaler, useV1API bool) (forwarder.Payloads, http.Header, error) {
	marshaler := eventsStreamJSONMarshaler.CreateSingleMarshaler()
	eventPayloads, extraHeaders, err := s.serializeStreamablePayload(marshaler, jsonstream.FailOnErrItemTooBig)

	if err == jsonstream.ErrItemTooBig {
		expvarsSendEventsErrItemTooBigs.Add(1)

		// Do not use CreateMarshalersBySourceType when there are too many source types (Performance issue).
		if marshaler.Len() > maxItemCountForCreateMarshalersBySourceType {
			expvarsSendEventsErrItemTooBigsFallback.Add(1)
			eventPayloads, extraHeaders, err = s.serializePayload(eventsStreamJSONMarshaler, true, useV1API)
		} else {
			eventPayloads = nil
			for _, v := range eventsStreamJSONMarshaler.CreateMarshalersBySourceType() {
				var eventPayloadsForSourceType forwarder.Payloads
				eventPayloadsForSourceType, extraHeaders, err = s.serializeStreamablePayload(v, jsonstream.DropItemOnErrItemTooBig)
				if err != nil {
					return nil, nil, err
				}
				eventPayloads = append(eventPayloads, eventPayloadsForSourceType...)
			}
		}
	}
	return eventPayloads, extraHeaders, err
}

// SendEvents serializes a list of event and sends the payload to the forwarder
func (s *Serializer) SendEvents(e EventsStreamJSONMarshaler) error {
	if !s.enableEvents {
		logutil.BgLogger().Debug("events payloads are disabled: dropping it")
		return nil
	}

	useV1API := !config.Datadog.GetBool("use_v2_api.events")
	var eventPayloads forwarder.Payloads
	var extraHeaders http.Header
	var err error

	if useV1API && s.enableEventsJSONStream {
		eventPayloads, extraHeaders, err = s.serializeEventsStreamJSONMarshalerPayload(e, useV1API)
	} else {
		eventPayloads, extraHeaders, err = s.serializePayload(e, true, useV1API)
	}
	if err != nil {
		return fmt.Errorf("dropping event payload: %s", err)
	}

	if useV1API {
		return s.Forwarder.SubmitV1Intake(eventPayloads, extraHeaders)
	}
	return s.Forwarder.SubmitEvents(eventPayloads, extraHeaders)
}

// SendServiceChecks serializes a list of serviceChecks and sends the payload to the forwarder
func (s *Serializer) SendServiceChecks(sc marshaler.StreamJSONMarshaler) error {
	if !s.enableServiceChecks {
		logutil.BgLogger().Debug("service_checks payloads are disabled: dropping it")
		return nil
	}

	useV1API := !config.Datadog.GetBool("use_v2_api.service_checks")

	var serviceCheckPayloads forwarder.Payloads
	var extraHeaders http.Header
	var err error

	if useV1API && s.enableServiceChecksJSONStream {
		serviceCheckPayloads, extraHeaders, err = s.serializeStreamablePayload(sc, jsonstream.DropItemOnErrItemTooBig)
	} else {
		serviceCheckPayloads, extraHeaders, err = s.serializePayload(sc, true, useV1API)
	}
	if err != nil {
		return fmt.Errorf("dropping service check payload: %s", err)
	}

	if useV1API {
		return s.Forwarder.SubmitV1CheckRuns(serviceCheckPayloads, extraHeaders)
	}
	return s.Forwarder.SubmitServiceChecks(serviceCheckPayloads, extraHeaders)
}

// SendSeries serializes a list of serviceChecks and sends the payload to the forwarder
func (s *Serializer) SendSeries(series marshaler.StreamJSONMarshaler) error {
	if !s.enableSeries {
		logutil.BgLogger().Debug("series payloads are disabled: dropping it")
		return nil
	}

	useV1API := !config.Datadog.GetBool("use_v2_api.series")

	var seriesPayloads forwarder.Payloads
	var extraHeaders http.Header
	var err error

	if useV1API && s.enableJSONStream {
		seriesPayloads, extraHeaders, err = s.serializeStreamablePayload(series, jsonstream.DropItemOnErrItemTooBig)
	} else {
		seriesPayloads, extraHeaders, err = s.serializePayload(series, true, useV1API)
	}

	if err != nil {
		return fmt.Errorf("dropping series payload: %s", err)
	}

	if useV1API {
		return s.Forwarder.SubmitV1Series(seriesPayloads, extraHeaders)
	}
	return s.Forwarder.SubmitSeries(seriesPayloads, extraHeaders)
}

// SendSketch serializes a list of SketSeriesList and sends the payload to the forwarder
func (s *Serializer) SendSketch(sketches marshaler.Marshaler) error {
	if !s.enableSketches {
		logutil.BgLogger().Debug("sketches payloads are disabled: dropping it")
		return nil
	}

	compress := true
	useV1API := false // Sketches only have a v2 endpoint
	splitSketches, extraHeaders, err := s.serializePayload(sketches, compress, useV1API)
	if err != nil {
		return fmt.Errorf("dropping sketch payload: %s", err)
	}

	return s.Forwarder.SubmitSketchSeries(splitSketches, extraHeaders)
}

// SendMetadata serializes a metadata payload and sends it to the forwarder
func (s *Serializer) SendMetadata(m marshaler.Marshaler) error {
	smallEnough, compressedPayload, payload, err := split.CheckSizeAndSerialize(m, true, split.MarshalJSON)
	if err != nil {
		return fmt.Errorf("could not determine size of metadata payload: %s", err)
	}

	logutil.BgLogger().Debug(fmt.Sprintf("Sending metadata payload, content: %v", apiKeyRegExp.ReplaceAllString(string(payload), apiKeyReplacement)))

	if !smallEnough {
		return fmt.Errorf("metadata payload was too big to send (%d bytes compressed), metadata payloads cannot be split", len(compressedPayload))
	}

	if err := s.Forwarder.SubmitV1Intake(forwarder.Payloads{&compressedPayload}, jsonExtraHeadersWithCompression); err != nil {
		return err
	}

	logutil.BgLogger().Info(fmt.Sprintf("Sent metadata payload, size (raw/compressed): %d/%d bytes.", len(payload), len(compressedPayload)))
	return nil
}

// SendJSONToV1Intake serializes a payload and sends it to the forwarder. Some code sends
// arbitrary payload the v1 API.
func (s *Serializer) SendJSONToV1Intake(data interface{}) error {
	if !s.enableJSONToV1Intake {
		logutil.BgLogger().Debug("JSON to V1 intake endpoint payloads are disabled: dropping it")
		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not serialize v1 payload: %s", err)
	}
	if err := s.Forwarder.SubmitV1Intake(forwarder.Payloads{&payload}, jsonExtraHeaders); err != nil {
		return err
	}

	logutil.BgLogger().Info(fmt.Sprintf("Sent processes metadata payload, size: %d bytes.", len(payload)))
	logutil.BgLogger().Debug(fmt.Sprintf("Sent processes metadata payload, content: %v", apiKeyRegExp.ReplaceAllString(string(payload), apiKeyReplacement)))
	return nil
}
