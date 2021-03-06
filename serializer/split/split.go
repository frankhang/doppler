// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package split

import (
	"expvar"
	"fmt"
	"go.uber.org/zap"

	"github.com/frankhang/doppler/forwarder"
	"github.com/frankhang/doppler/serializer/marshaler"
	"github.com/frankhang/doppler/telemetry"
	"github.com/frankhang/doppler/util/compression"

	"github.com/frankhang/util/logutil"
)

// the backend accepts payloads up to 3MB, but being conservative is okay
var maxPayloadSize = 2 * 1024 * 1024

// MarshalType is the type of marshaler to use
type MarshalType int

// Enumeration of the existing marshal types
const (
	MarshalJSON MarshalType = iota
	Marshal
)

var (
	// TODO(remy): could probably be removed as not used in the status page
	splitterExpvars      = expvar.NewMap("splitter")
	splitterNotTooBig    = expvar.Int{}
	splitterTooBig       = expvar.Int{}
	splitterTotalLoops   = expvar.Int{}
	splitterPayloadDrops = expvar.Int{}

	tlmSplitterNotTooBig = telemetry.NewCounter("splitter", "not_too_big",
		nil, "Splitter 'not too big' occurrences")
	tlmSplitterTooBig = telemetry.NewCounter("splitter", "too_big",
		nil, "Splitter 'too big' occurrences")
	tlmSplitterTotalLoops = telemetry.NewCounter("splitter", "total_loops",
		nil, "Splitter total loops run")
	tlmSplitterPayloadDrops = telemetry.NewCounter("splitter", "payload_drops",
		nil, "Splitter payload drops")
)

func init() {
	splitterExpvars.Set("NotTooBig", &splitterNotTooBig)
	splitterExpvars.Set("TooBig", &splitterTooBig)
	splitterExpvars.Set("TotalLoops", &splitterTotalLoops)
	splitterExpvars.Set("PayloadDrops", &splitterPayloadDrops)

}

// CheckSizeAndSerialize Check the size of a payload and marshall it (optionally compress it)
// The dual role makes sense as you will never serialize without checking the size of the payload
func CheckSizeAndSerialize(m marshaler.Marshaler, compress bool, mType MarshalType) (bool, []byte, []byte, error) {
	compressedPayload, payload, err := serializeMarshaller(m, compress, mType)
	if err != nil {
		return false, nil, nil, err
	}
	return checkSize(compressedPayload), compressedPayload, payload, nil
}

// Payloads serializes a metadata payload and sends it to the forwarder
func Payloads(m marshaler.Marshaler, compress bool, mType MarshalType) (forwarder.Payloads, error) {
	marshallers := []marshaler.Marshaler{m}
	smallEnoughPayloads := forwarder.Payloads{}
	nottoobig, payload, _, err := CheckSizeAndSerialize(m, compress, mType)
	if err != nil {
		return smallEnoughPayloads, err
	}
	// If the payload's size is fine, just return it
	if nottoobig {
		logutil.BgLogger().Debug("The payload was not too big, returning the full payload")
		splitterNotTooBig.Add(1)
		tlmSplitterNotTooBig.Inc()
		smallEnoughPayloads = append(smallEnoughPayloads, &payload)
		return smallEnoughPayloads, nil
	}
	splitterTooBig.Add(1)
	tlmSplitterTooBig.Inc()
	toobig := !nottoobig
	loops := 0
	// Do not attempt to split payloads forever, if a payload cannot be split then abandon the task
	// the function will return all the payloads that were able to be split
	for toobig && loops < 3 {
		splitterTotalLoops.Add(1)
		tlmSplitterTotalLoops.Inc()
		// create a temporary slice, the other array will be reused to keep track of the payloads that have yet to be split
		tempSlice := make([]marshaler.Marshaler, len(marshallers))
		copy(tempSlice, marshallers)
		marshallers = []marshaler.Marshaler{}
		for _, toSplit := range tempSlice {
			var e error
			// we have to do this every time to get the proper payload
			payload, compressedPayload, e := serializeMarshaller(toSplit, compress, mType)
			if e != nil {
				return smallEnoughPayloads, e
			}
			payloadSize := len(payload)
			compressedSize := len(compressedPayload)
			// Attempt to account for the compression when estimating the number of chunks that will be needed
			// This is the same function used in dd-agent
			compressionRatio := float64(payloadSize) / float64(compressedSize)
			numChunks := compressedSize/maxPayloadSize + 1 + int(compressionRatio/2)
			logutil.BgLogger().Debug(fmt.Sprintf("split the payload into into %d chunks", numChunks))
			chunks, err := toSplit.SplitPayload(numChunks)
			logutil.BgLogger().Debug(fmt.Sprintf("payload was split into %d chunks", len(chunks)))
			if err != nil {
				logutil.BgLogger().Warn("Some payloads could not be split, dropping them")
				splitterPayloadDrops.Add(1)
				tlmSplitterPayloadDrops.Inc()
				return smallEnoughPayloads, err
			}
			// after the payload has been split, loop through the chunks
			for _, chunk := range chunks {
				// serialize the payload
				smallEnough, payload, _, err := CheckSizeAndSerialize(chunk, compress, mType)
				if err != nil {
					logutil.BgLogger().Debug("Error serializing a chunk", zap.Error(err))
					continue
				}
				if smallEnough {
					// if the payload is small enough, return it straight away
					smallEnoughPayloads = append(smallEnoughPayloads, &payload)
					logutil.BgLogger().Debug(fmt.Sprintf("chunk was small enough: %v, smallEnoughPayloads are of length: %v", len(payload), len(smallEnoughPayloads)))
				} else {
					// if it is not, append it to the list of payloads
					marshallers = append(marshallers, chunk)
					logutil.BgLogger().Debug(fmt.Sprintf("chunk was not small enough: %v, marshallers are of length: %v", len(payload), len(marshallers)))
				}
			}
		}
		if len(marshallers) == 0 {
			logutil.BgLogger().Debug("marshallers was empty, breaking out of the loop")
			toobig = false
		} else {
			logutil.BgLogger().Debug("marshallers was not empty, running around the loop again")
			loops++
		}
	}
	if len(marshallers) != 0 {
		logutil.BgLogger().Warn("Some payloads could not be split, dropping them")
		splitterPayloadDrops.Add(1)
		tlmSplitterPayloadDrops.Inc()
	}

	return smallEnoughPayloads, nil
}

// serializeMarshaller serializes the marshaller and returns both the compressed and uncompressed payloads
func serializeMarshaller(m marshaler.Marshaler, compress bool, mType MarshalType) ([]byte, []byte, error) {
	var payload []byte
	var compressedPayload []byte
	var err error
	payload, err = marshal(m, mType)
	compressedPayload = payload
	if err != nil {
		return nil, nil, err
	}
	if compress {
		compressedPayload, err = compression.Compress(nil, payload)
		if err != nil {
			return nil, nil, err
		}
	}
	return compressedPayload, payload, nil
}

func checkSize(payload []byte) bool {
	if len(payload) >= maxPayloadSize {
		return false
	}
	return true
}

func marshal(m marshaler.Marshaler, mType MarshalType) ([]byte, error) {
	switch mType {
	case MarshalJSON:
		return m.MarshalJSON()
	case Marshal:
		return m.Marshal()
	default:
		return m.MarshalJSON()
	}
}

// GetPayloadDrops returns the number of times we dropped some payloads because we couldn't split them.
func GetPayloadDrops() int64 {
	return splitterPayloadDrops.Value()
}
