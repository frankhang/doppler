package exporter

import (
	"github.com/frankhang/util/hack"
)

const (
	GaugeSymbol     = 'g'
	CountSymbol     = 'c'
	HistogramSymbol = 'h'
	SummarySymbol   = 's'
)

type PromMetric struct {
	k string

	Symbol byte
	Name   string

	LabelNames []string

	//collector prometheus.Collector
}

func (pm *PromMetric) GenerateKey() {

	buf := make([]byte, 0, 64)
	buf = append(buf, pm.Symbol)
	buf = append(buf, '|')
	buf = append(buf, hack.Slice(pm.Name)...)

	labels := pm.LabelNames

	if len(labels) > 0 {
		for _, label := range labels {
			buf = append(buf, '|')
			buf = append(buf, hack.Slice(label)...)
		}
	}

	pm.k = hack.String(buf)
}


func (pm *PromMetric) String() string {
	return pm.k
}

