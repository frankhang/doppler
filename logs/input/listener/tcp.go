// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package listener

import (
	"fmt"
	"go.uber.org/zap"
	"net"
	"sync"
	"time"

	"github.com/frankhang/util/logutil"

	"github.com/frankhang/doppler/logs/config"
	"github.com/frankhang/doppler/logs/pipeline"
	"github.com/frankhang/doppler/logs/restart"
)

// defaultTimeout represents the time after which a connection is closed when no data is read
const defaultTimeout = time.Minute

// A TCPListener listens and accepts TCP connections and delegates the read operations to a tailer.
type TCPListener struct {
	pipelineProvider pipeline.Provider
	source           *config.LogSource
	frameSize        int
	listener         net.Listener
	tailers          []*Tailer
	mu               sync.Mutex
	stop             chan struct{}
}

// NewTCPListener returns an initialized TCPListener
func NewTCPListener(pipelineProvider pipeline.Provider, source *config.LogSource, frameSize int) *TCPListener {
	return &TCPListener{
		pipelineProvider: pipelineProvider,
		source:           source,
		frameSize:        frameSize,
		tailers:          []*Tailer{},
		stop:             make(chan struct{}, 1),
	}
}

// Start starts the listener to accepts new incoming connections.
func (l *TCPListener) Start() {
	logutil.BgLogger().Info(fmt.Sprintf("Starting TCP forwarder on port %d, with read buffer size: %d", l.source.Config.Port, l.frameSize))
	err := l.startListener()
	if err != nil {
		logutil.BgLogger().Error(fmt.Sprintf("Can't start TCP forwarder on port %d", l.source.Config.Port), zap.Error(err))
		l.source.Status.Error(err)
		return
	}
	l.source.Status.Success()
	go l.run()
}

// Stop stops the listener from accepting new connections and all the activer tailers.
func (l *TCPListener) Stop() {
	logutil.BgLogger().Info(fmt.Sprintf("Stopping TCP forwarder on port %d", l.source.Config.Port))
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stop <- struct{}{}
	l.listener.Close()
	stopper := restart.NewParallelStopper()
	for _, tailer := range l.tailers {
		stopper.Add(tailer)
	}
	stopper.Stop()
}

// run accepts new TCP connections and create a dedicated tailer for each.
func (l *TCPListener) run() {
	defer l.listener.Close()
	for {
		select {
		case <-l.stop:
			// stop accepting new connections.
			return
		default:
			conn, err := l.listener.Accept()
			switch {
			case err != nil && isClosedConnError(err):
				return
			case err != nil:
				// an error occurred, restart the listener.
				logutil.BgLogger().Warn(fmt.Sprintf("Can't listen on port %d, restarting a listener", l.source.Config.Port), zap.Error(err))
				l.listener.Close()
				err := l.startListener()
				if err != nil {
					logutil.BgLogger().Error(fmt.Sprintf("Can't restart listener on port %d", l.source.Config.Port), zap.Error(err))
					l.source.Status.Error(err)
					return
				}
				l.source.Status.Success()
				continue
			default:
				l.startTailer(conn)
				l.source.Status.Success()
			}
		}
	}
}

// startListener starts a new listener, returns an error if it failed.
func (l *TCPListener) startListener() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", l.source.Config.Port))
	if err != nil {
		return err
	}
	l.listener = listener
	return nil
}

// read reads data from connection, returns an error if it failed and stop the tailer.
func (l *TCPListener) read(tailer *Tailer) ([]byte, error) {
	tailer.conn.SetReadDeadline(time.Now().Add(defaultTimeout))
	frame := make([]byte, l.frameSize)
	n, err := tailer.conn.Read(frame)
	if err != nil {
		l.source.Status.Error(err)
		go l.stopTailer(tailer)
		return nil, err
	}
	return frame[:n], nil
}

// startTailer creates and starts a new tailer that reads from the connection.
func (l *TCPListener) startTailer(conn net.Conn) {
	l.mu.Lock()
	defer l.mu.Unlock()
	tailer := NewTailer(l.source, conn, l.pipelineProvider.NextPipelineChan(), l.read)
	l.tailers = append(l.tailers, tailer)
	tailer.Start()
}

// stopTailer stops the tailer.
func (l *TCPListener) stopTailer(tailer *Tailer) {
	tailer.Stop()
	l.mu.Lock()
	defer l.mu.Unlock()
	for i, t := range l.tailers {
		if t == tailer {
			l.tailers = append(l.tailers[:i], l.tailers[i+1:]...)
			break
		}
	}
}
