package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/frankhang/util/logutil"
	"github.com/frankhang/util/tcp"
	l "github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

//tierHandler implements Hanlder
type Handler struct {
	*PacketIO

	driver *Driver

	ctl *Controller
}

func NewHandler(tierPacketIO *PacketIO, driver *Driver) *Handler {
	handler := &Handler{PacketIO: tierPacketIO, driver: driver}
	handler.ctl = &Controller{PacketIO: tierPacketIO}
	return handler
}

func (th *Handler) Handle(ctx context.Context, cc *tcp.ClientConn, header []byte, data []byte) (err error) {

	var delim byte

	if l.GetLevel() >= l.DebugLevel {
		logutil.Logger(ctx).Debug("Packet received",
			zap.Int("size", len(data)),
			zap.String("packet", fmt.Sprintf("%s", data)),

		)
	}

	ctl := th.ctl
	ctl.cc = cc

	s := 0 // search start index
	var i int

	delim = ':'
	if i = bytes.IndexByte(data[s:], delim); i < 0 {
		return ErrMissingDelim.FastGenByArgs(delim, "name")
	}
	name := data[s : s+i]
	s += i + 1

	delim = '|'
	if i = bytes.IndexByte(data[s:], delim); i < 0 {
		return ErrMissingDelim.FastGenByArgs(delim, "value")
	}
	valueSlice := data[s : s+i]
	s += i + 1

	if len(data[s:]) == 0 {
		return ErrNoMetricType
	}

	var metricType MetricType
	var ok bool
	symbol := data[s]
	if metricType, ok = metricSymbol2Code[symbol]; !ok {
		return ErrBadMetricType.FastGenByArgs(symbol)
	}
	s++

	if metricType == timingMetric {
		s++
		if len(data[s:]) == 0 {
			return ErrBadFormat.FastGenByArgs(data)
		}
		if data[s] != 's' {
			return ErrBadFormat.FastGenByArgs(data)
		}
	}

	var rateSlice []byte
	var tagSlice []byte
	var splitSlice [][]byte
	if len(data[s:]) >= 2 && data[s] == '|' && data[s+1] == '@' {
		s += 2
		if len(data[s:]) == 0 {
			return ErrBadFormat.FastGenByArgs(data)
		}

		i = bytes.IndexByte(data[s:], '|')
		if i == 0 {
			return ErrBadFormat.FastGenByArgs(data)
		}
		if i < 0 { //no tags
			rateSlice = data

			if l.GetLevel() >= l.DebugLevel {
				logutil.Logger(ctx).Debug("Handle",
					zap.String("name", string(name)),
					zap.String("value", string(valueSlice)),
					zap.String("symbol", string(symbol)),
					zap.String("rate", string(rateSlice)),
				)
			}

		} else { //i > 0 with tags
			rateSlice = data[s : s+i]
			s += i + 1

			if l.GetLevel() >= l.DebugLevel {
				logutil.Logger(ctx).Debug("Handle",
					zap.String("name", string(name)),
					zap.String("value", string(valueSlice)),
					zap.String("symbol", string(symbol)),
					zap.String("rate", string(rateSlice)),
				)
			}

			//read tags
			if !(len(data[s:]) >= 2 && data[0] == '#') {
				return ErrBadFormat.FastGenByArgs(data)
			}
			s++

			tagSlice = data[s:]

			//parse tags
			splitSlice = bytes.Split(tagSlice, []byte(","))
			for _, tag := range splitSlice {
				tagPair := bytes.Split(tag, []byte(":"))

				if l.GetLevel() >= l.DebugLevel {
					logutil.Logger(ctx).Debug("Tag",
						zap.String("name", string(tagPair[0])),
						zap.String("value", string(tagPair[1])),

					)
				}

			}

		}

		//todo: parse result
		if len(valueSlice) > 0 {

		}
		if len(rateSlice) > 0 {

		}

	}
	
	return nil
}
