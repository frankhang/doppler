// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"expvar"
	"fmt"
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"

	"net"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	. "github.com/frankhang/doppler/config"
	"github.com/frankhang/doppler/mapper"
	"github.com/frankhang/doppler/metrics"
	"github.com/frankhang/doppler/status/health"
	"github.com/frankhang/doppler/tagger"
	"github.com/frankhang/doppler/telemetry"
	"github.com/frankhang/doppler/util"

	l "github.com/sirupsen/logrus"
)

var (
	dogstatsdExpvars                 = expvar.NewMap("dogstatsd")
	dogstatsdServiceCheckParseErrors = expvar.Int{}
	dogstatsdServiceCheckPackets     = expvar.Int{}
	dogstatsdEventParseErrors        = expvar.Int{}
	dogstatsdEventPackets            = expvar.Int{}
	dogstatsdMetricParseErrors       = expvar.Int{}
	dogstatsdMetricPackets           = expvar.Int{}
	dogstatsdPacketsLastSec          = expvar.Int{}

	tlmProcessed = telemetry.NewCounter("dogstatsd", "processed",
		[]string{"message_type", "state"}, "Count of service checks/events/metrics processed by dogstatsd")
)

func init() {
	dogstatsdExpvars.Set("ServiceCheckParseErrors", &dogstatsdServiceCheckParseErrors)
	dogstatsdExpvars.Set("ServiceCheckPackets", &dogstatsdServiceCheckPackets)
	dogstatsdExpvars.Set("EventParseErrors", &dogstatsdEventParseErrors)
	dogstatsdExpvars.Set("EventPackets", &dogstatsdEventPackets)
	dogstatsdExpvars.Set("MetricParseErrors", &dogstatsdMetricParseErrors)
	dogstatsdExpvars.Set("MetricPackets", &dogstatsdMetricPackets)
}

// Server represent a Dogstatsd server
type Server struct {
	listeners             []StatsdListener
	packetsIn             chan Packets
	samplePool            *metrics.MetricSamplePool
	samplesOut            chan<- []metrics.MetricSample
	eventsOut             chan<- []*metrics.Event
	servicesCheckOut      chan<- []*metrics.ServiceCheck
	Statistics            *util.Stats
	Started               bool
	packetPool            *PacketPool
	stopChan              chan bool
	health                *health.Handle
	metricPrefix          string
	metricPrefixBlacklist []string
	defaultHostname       string
	histToDist            bool
	histToDistPrefix      string
	extraTags             []string
	debugMetricsStats     bool
	metricsStats          map[string]metricStat
	statsLock             sync.Mutex
	mapper                *mapper.MetricMapper
}

// metricStat holds how many times a metric has been
// processed and when was the last time.
type metricStat struct {
	Count    uint64    `json:"count"`
	LastSeen time.Time `json:"last_seen"`
}

// NewServer returns a running Dogstatsd server
func NewServer(samplePool *metrics.MetricSamplePool, samplesOut chan<- []metrics.MetricSample, eventsOut chan<- []*metrics.Event, servicesCheckOut chan<- []*metrics.ServiceCheck) (*Server, error) {
	var stats *util.Stats
	if Cfg.AgentStatsEnable {
		buff := Cfg.AgentStatsBuffer
		s, err := util.NewStats(uint32(buff))
		if err != nil {
			logutil.BgLogger().Error("Agent: unable to start statistics facilities")
		}
		stats = s
		dogstatsdExpvars.Set("PacketsLastSecond", &dogstatsdPacketsLastSec)
	}

	var metricsStats bool
	if Cfg.MetricsStatsEnable {
		logutil.BgLogger().Info("Agent: metrics statistics will be stored")
		metricsStats = true
	}

	packetsChannel := make(chan Packets, Cfg.AgentQueueSize)
	packetPool := NewPacketPool(Cfg.AgentBufferSize)
	tmpListeners := make([]StatsdListener, 0, 2)

	//socketPath := config.Datadog.GetString("dogstatsd_socket")
	//if len(socketPath) > 0 {
	//	unixListener, err := NewUDSListener(packetsChannel, packetPool)
	//	if err != nil {
	//		log.Errorf(err.Error())
	//	} else {
	//		tmpListeners = append(tmpListeners, unixListener)
	//	}
	//}
	if Cfg.Port > 0 {
		udpListener, err := NewUDPListener(packetsChannel, packetPool)
		if err != nil {
			return nil, errors.Trace(err)
		} else {
			tmpListeners = append(tmpListeners, udpListener)
		}
	}

	if len(tmpListeners) == 0 {
		err := fmt.Errorf("listening on neither udp nor socket, please check your configuration")
		return nil, errors.Trace(err)
	}

	// check configuration for custom namespace
	metricPrefix := Cfg.MetricNamespace
	if metricPrefix != "" && !strings.HasSuffix(metricPrefix, ".") {
		metricPrefix = metricPrefix + "."
	}
	metricPrefixBlacklist := Cfg.MetricNamespaceBlacklist

	defaultHostname, err := util.GetHostname()
	if err != nil {
		return nil, errors.Trace(err)
	}

	histToDist := Cfg.HistogramCopyToDistribution
	histToDistPrefix := Cfg.HistogramCopyToDistributionPrefix

	extraTags := Cfg.AgentTags

	s := &Server{
		Started:               true,
		Statistics:            stats,
		samplePool:            samplePool,
		packetsIn:             packetsChannel,
		samplesOut:            samplesOut,
		eventsOut:             eventsOut,
		servicesCheckOut:      servicesCheckOut,
		listeners:             tmpListeners,
		packetPool:            packetPool,
		stopChan:              make(chan bool),
		health:                health.Register("agent-main"),
		metricPrefix:          metricPrefix,
		metricPrefixBlacklist: metricPrefixBlacklist,
		defaultHostname:       defaultHostname,
		histToDist:            histToDist,
		histToDistPrefix:      histToDistPrefix,
		extraTags:             extraTags,
		debugMetricsStats:     metricsStats,
		metricsStats:          make(map[string]metricStat),
	}

	forwardHost := Cfg.ForwardHost
	forwardPort := Cfg.ForwardPort

	if forwardHost != "" && forwardPort != 0 {

		forwardAddress := fmt.Sprintf("%s:%d", forwardHost, forwardPort)

		con, err := net.Dial("udp", forwardAddress)

		if err != nil {
			logutil.BgLogger().Warn("Could not connect to statsd forward host", zap.Error(err))
		} else {
			s.packetsIn = make(chan Packets, Cfg.AgentQueueSize)
			go s.forwarder(con, packetsChannel)
		}
	}

	s.handleMessages()

	cacheSize := Cfg.CacheSize

	mappings, err := GetDogstatsdMappingProfiles()
	if err != nil {
		logutil.BgLogger().Warn("Could not parse mapping profiles", zap.Error(err))
	} else if len(mappings) != 0 {
		mapperInstance, err := mapper.NewMetricMapper(mappings, cacheSize)
		if err != nil {
			logutil.BgLogger().Warn("Could not create metric mapper", zap.Error(err))
		} else {
			s.mapper = mapperInstance
		}
	}
	return s, nil
}

func (s *Server) handleMessages() {
	if s.Statistics != nil {
		go s.Statistics.Process()
		go s.Statistics.Update(&dogstatsdPacketsLastSec)
	}

	for _, l := range s.listeners {
		go l.Listen()
	}

	// Run min(2, GoMaxProcs-2) workers, we dedicate a core to the
	// listener goroutine and another to aggregator + forwarder
	workers := runtime.GOMAXPROCS(-1) - 2
	if workers < 2 {
		workers = 2
	}

	for i := 0; i < workers; i++ {
		go s.worker()
	}
}

func (s *Server) forwarder(fcon net.Conn, packetsChannel chan Packets) {
	for {
		select {
		case <-s.stopChan:
			return
		case packets := <-packetsChannel:
			for _, packet := range packets {
				_, err := fcon.Write(packet.Contents)

				if err != nil {
					logutil.BgLogger().Warn("Forwarding packet failed", zap.Error(err))
				}
			}
			s.packetsIn <- packets
		}
	}
}

func (s *Server) worker() {
	batcher := newBatcher(s.samplePool, s.samplesOut, s.eventsOut, s.servicesCheckOut)
	for {
		select {
		case <-s.stopChan:
			return
		case <-s.health.C:
		case packets := <-s.packetsIn:
			s.parsePackets(batcher, packets)
		}
	}
}

func nextMessage(packet *[]byte) (message []byte) {
	if len(*packet) == 0 {
		return nil
	}

	advance, message, err := bufio.ScanLines(*packet, true)
	if err != nil || len(message) == 0 {
		return nil
	}

	*packet = (*packet)[advance:]
	return message
}




func (s *Server) parsePackets(batcher *batcher, packets []*Packet) {
	for _, packet := range packets {
		originTags := findOriginTags(packet.Origin)
		if l.GetLevel() >= l.DebugLevel {
			logutil.BgLogger().Debug("Agent receive", zap.ByteString("packet", packet.Contents))
		}

		//logutil.BgLogger().Info("Agent receive", zap.ByteString("packet", packet.Contents))


		for {
			message := nextMessage(&packet.Contents)
			if message == nil {
				break
			}

			logutil.BgLogger().Debug("nextMessage", zap.ByteString("message", message))
			if s.Statistics != nil {
				s.Statistics.StatEvent(1)
			}
			messageType := findMessageType(message)

			switch messageType {
			case serviceCheckType:
				serviceCheck, err := s.parseServiceCheckMessage(message)
				if err != nil {
					logutil.BgLogger().Error("Agent: error parsing service check", zap.Error(err))
					continue
				}
				serviceCheck.Tags = append(serviceCheck.Tags, originTags...)
				batcher.appendServiceCheck(serviceCheck)
			case eventType:
				event, err := s.parseEventMessage(message)
				if err != nil {
					logutil.BgLogger().Error("Agent: error parsing event", zap.Error(err))
					continue
				}
				event.Tags = append(event.Tags, originTags...)
				batcher.appendEvent(event)
			case metricSampleType:
				sample, err := s.parseMetricMessage(message)
				if err != nil {
					logutil.BgLogger().Error("Agent: error parsing metrics", zap.Error(err))
					continue
				}
				if s.debugMetricsStats {
					s.storeMetricStats(sample.Name)
				}
				sample.Tags = append(sample.Tags, originTags...)
				batcher.appendSample(sample)
				if s.histToDist && sample.Mtype == metrics.HistogramType {
					distSample := sample.Copy()
					distSample.Name = s.histToDistPrefix + distSample.Name
					distSample.Mtype = metrics.DistributionType
					batcher.appendSample(*distSample)
				}
			}
		}
	}
	batcher.flush()
}

func (s *Server) parseMetricMessage(message []byte) (metrics.MetricSample, error) {
	sample, err := parseMetricSample(message)
	if err != nil {
		dogstatsdMetricParseErrors.Add(1)
		tlmProcessed.Inc("metrics", "error")
		return metrics.MetricSample{}, err
	}
	if s.mapper != nil && len(sample.tags) == 0 {
		mapResult := s.mapper.Map(sample.name)
		if mapResult != nil {
			sample.name = mapResult.Name
			sample.tags = append(sample.tags, mapResult.Tags...)
		}
	}
	metricSample := enrichMetricSample(sample, s.metricPrefix, s.metricPrefixBlacklist, s.defaultHostname)
	metricSample.Tags = append(metricSample.Tags, s.extraTags...)
	dogstatsdMetricPackets.Add(1)
	tlmProcessed.Inc("metrics", "ok")
	return metricSample, nil
}

func (s *Server) parseEventMessage(message []byte) (*metrics.Event, error) {
	sample, err := parseEvent(message)
	if err != nil {
		dogstatsdEventParseErrors.Add(1)
		tlmProcessed.Inc("events", "error")
		return nil, err
	}
	event := enrichEvent(sample, s.defaultHostname)
	event.Tags = append(event.Tags, s.extraTags...)
	tlmProcessed.Inc("events", "ok")
	dogstatsdEventPackets.Add(1)
	return event, nil
}

func (s *Server) parseServiceCheckMessage(message []byte) (*metrics.ServiceCheck, error) {
	sample, err := parseServiceCheck(message)
	if err != nil {
		dogstatsdServiceCheckParseErrors.Add(1)
		tlmProcessed.Inc("service_checks", "error")
		return nil, err
	}
	serviceCheck := enrichServiceCheck(sample, s.defaultHostname)
	serviceCheck.Tags = append(serviceCheck.Tags, s.extraTags...)
	dogstatsdServiceCheckPackets.Add(1)
	tlmProcessed.Inc("service_checks", "ok")
	return serviceCheck, nil
}

func findOriginTags(origin string) []string {
	var tags []string
	if origin != NoOrigin {
		originTags, err := tagger.Tag(origin, tagger.DogstatsdCardinality)
		if err != nil {
			logutil.BgLogger().Error(err.Error())
		} else {
			tags = append(tags, originTags...)
		}
	}
	return tags
}

// Stop stops a running Dogstatsd server
func (s *Server) Stop() {
	close(s.stopChan)
	for _, l := range s.listeners {
		l.Stop()
	}
	if s.Statistics != nil {
		s.Statistics.Stop()
	}
	s.health.Deregister()
	s.Started = false
}

func (s *Server) storeMetricStats(name string) {
	now := time.Now()
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	ms := s.metricsStats[name]
	ms.Count++
	ms.LastSeen = now
	s.metricsStats[name] = ms
}

// GetJSONDebugStats returns jsonified debug statistics.
func (s *Server) GetJSONDebugStats() ([]byte, error) {
	s.statsLock.Lock()
	defer s.statsLock.Unlock()
	return json.Marshal(s.metricsStats)
}

// FormatDebugStats returns a printable version of debug stats.
func FormatDebugStats(stats []byte) (string, error) {
	var dogStats map[string]metricStat
	if err := json.Unmarshal(stats, &dogStats); err != nil {
		return "", err
	}

	// put tags in order: first is the more frequent
	order := make([]string, len(dogStats))
	i := 0
	for tag := range dogStats {
		order[i] = tag
		i++
	}

	sort.Slice(order, func(i, j int) bool {
		return dogStats[order[i]].Count > dogStats[order[j]].Count
	})

	// write the response
	buf := bytes.NewBuffer(nil)

	header := fmt.Sprintf("%-40s | %-10s | %-20s\n", "Metric", "Count", "Last Seen")
	buf.Write([]byte(header))
	buf.Write([]byte(strings.Repeat("-", len(header)) + "\n"))

	for _, metric := range order {
		stats := dogStats[metric]
		buf.Write([]byte(fmt.Sprintf("%-40s | %-10d | %-20v\n", metric, stats.Count, stats.LastSeen)))
	}

	if len(dogStats) == 0 {
		buf.Write([]byte("No metrics processed yet."))
	}

	return buf.String(), nil
}
