// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package agent

import (
	"expvar"
	"fmt"
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
	"net"
	"strconv"
	"strings"
	"time"

	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/telemetry"
)

var (
	udpExpvars             = expvar.NewMap("agent-udp")
	udpPacketReadingErrors = expvar.Int{}
	udpPackets             = expvar.Int{}
	udpBytes               = expvar.Int{}

	tlmUDPPackets = telemetry.NewCounter("agent", "udp_packets",
		[]string{"state"}, "Agent UDP packets count")
	tlmUDPPacketsBytes = telemetry.NewCounter("agent", "udp_packets_bytes",
		[]string{}, "Agent UDP packets bytes count")
)

func init() {
	udpExpvars.Set("PacketReadingErrors", &udpPacketReadingErrors)
	udpExpvars.Set("Packets", &udpPackets)
	udpExpvars.Set("Bytes", &udpBytes)
}

// UDPListener implements the StatsdListener interface for UDP protocol.
// It listens to a given UDP address and sends back packets ready to be
// processed.
// Origin detection is not implemented for UDP.
type UDPListener struct {
	conn          net.PacketConn
	packetsBuffer *packetsBuffer
	packetBuffer  *packetBuffer
	buffer        []byte
}

// NewUDPListener returns an idle UDP Statsd listener
func NewUDPListener(packetOut chan Packets, packetPool *PacketPool) (*UDPListener, error) {
	var conn net.PacketConn
	var err error
	var url string

	if Cfg.AgentNonLocalTraffic{
		// Listen to all network interfaces
		url = fmt.Sprintf(":%d", Cfg.Port)
	} else {
		url = net.JoinHostPort(Cfg.Host, strconv.Itoa(int(Cfg.Port)))
	}

	conn, err = net.ListenPacket("udp", url)

	if err != nil {
		err := fmt.Errorf("can't listen: %s", err)
		return nil, errors.Trace(err)
	}

	if rcvbuf := Cfg.AgentSoRcvbuf; rcvbuf != 0 {
		if err := conn.(*net.UDPConn).SetReadBuffer(rcvbuf); err != nil {
			err := fmt.Errorf("could not set socket rcvbuf: %s", err)
			return nil, errors.Trace(err)
		}
	}

	bufferSize := Cfg.AgentBufferSize
	packetsBufferSize := Cfg.AgentPacketBufferSize
	flushTimeout := time.Duration(Cfg.AgentPacketBufferFlushTimeout) * time.Millisecond

	buffer := make([]byte, bufferSize)
	packetsBuffer := newPacketsBuffer(uint(packetsBufferSize), flushTimeout, packetOut)
	packetBuffer := newPacketBuffer(packetPool, flushTimeout, packetsBuffer)

	listener := &UDPListener{
		conn:          conn,
		packetsBuffer: packetsBuffer,
		packetBuffer:  packetBuffer,
		buffer:        buffer,
	}
	logutil.BgLogger().Info("agent-udp: successfully initialized", zap.String("addr", conn.LocalAddr().String()))
	return listener, nil
}

// Listen runs the intake loop. Should be called in its own goroutine
func (l *UDPListener) Listen() {
	logutil.BgLogger().Info("agent-udp: starting to listen...", zap.String("addr", l.conn.LocalAddr().String()))
	for {
		udpPackets.Add(1)
		n, _, err := l.conn.ReadFrom(l.buffer)
		if err != nil {
			// connection has been closed
			if strings.HasSuffix(err.Error(), " use of closed network connection") {
				return
			}

			logutil.BgLogger().Error("agent-udp: error reading packet", zap.Error(err))
			udpPacketReadingErrors.Add(1)
			tlmUDPPackets.Inc("error")
			continue
		}
		tlmUDPPackets.Inc("ok")
		udpBytes.Add(int64(n))

		// packetBuffer merges multiple packets together and sends them when its buffer is full
		l.packetBuffer.addMessage(l.buffer[:n])
	}
}

// Stop closes the UDP connection and stops listening
func (l *UDPListener) Stop() {
	l.packetBuffer.close()
	l.packetsBuffer.close()
	l.conn.Close()
}
