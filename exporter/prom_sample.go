package exporter

import (
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
	pm.Name = s.Name
	ps.Value = s.Value

	tags := s.Tags

	if s.Mtype == metrics.SetType {
		tags = append(tags, s.RawValue)
		ps.Value = 1
	}

	if len(tags) > 0 {
		sort.Strings(tags)
		for _, tag := range tags {
			tagPair := strings.Split(tag, ":")
			if len(tagPair) != 2 {
				continue
			}
			tagName := strings.TrimSpace(tagPair[0])
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
