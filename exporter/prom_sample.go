package exporter

import (
	"fmt"
	"github.com/frankhang/doppler/metrics"
	"sort"
	"strings"
)

type PromSample struct {
	metric      *PromMetric
	Value       float64
	LableValues []string
}

func NewPromSample(s *metrics.MetricSample) *PromSample {
	pm := &PromMetric{}
	ps := &PromSample{metric: pm}

	pm.Symbol = getMetricSymbol(s)
	pm.Name = normalize(s.Name)
	ps.Value = s.Value

	tags := make([]string, 0, len(s.Tags) + 3)
	tags = append(tags, s.Tags...)

	if s.Mtype == metrics.SetType {
		rawValue := strings.TrimSpace(s.RawValue)
		if rawValue != "" {
			tags = append(tags, fmt.Sprintf("_setOf_%s:1", rawValue))
		}
		ps.Value = 1
	}

	host := strings.TrimSpace(s.Host)
	if host != "" {
		tags = append(tags, fmt.Sprintf("_agent_:%s", host))
	}
	tags = append(tags, fmt.Sprintf("_rate_:%.3f", s.SampleRate))

	if len(tags) > 0 {
		sort.Strings(tags)
		for _, tag := range tags {
			tagPair := strings.Split(tag, ":")
			if len(tagPair) != 2 {
				continue
			}
			//tagName := strings.TrimSpace(tagPair[0])
			tagName := normalize(tagPair[0])
			tagValue := strings.TrimSpace(tagPair[1])
			if len(tagName) == 0 || len(tagValue) == 0 {
				continue
			}

			pm.LabelNames = append(pm.LabelNames, tagName)
			ps.LableValues = append(ps.LableValues, tagValue)

		}
	}
	pm.GenerateKey()

	return ps

}

func normalize(s string) string {
	t := strings.ReplaceAll(s, ".", "_")
	return strings.TrimSpace(t)

}

func getMetricSymbol(sample *metrics.MetricSample) byte {

	var symbol byte
	switch sample.Mtype {
	case metrics.GaugeType:
		symbol = GaugeSymbol
	case metrics.CountType, metrics.CounterType, metrics.MonotonicCountType, metrics.SetType:
		symbol = CountSymbol
	case metrics.HistogramType, metrics.HistorateType, metrics.DistributionType:
		symbol = HistogramSymbol
	default:
		symbol = GaugeSymbol
	}

	return symbol

}
