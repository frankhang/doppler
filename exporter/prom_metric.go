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

func (pm *PromMetric) Key() string {
	return pm.k
}

func (pm *PromMetric) Hash() string {
	return pm.Key()
}

func (pm *PromMetric) HashCode() string {
	return pm.Key()
}
func (ps *PromMetric) String() string {
	return ps.Key()
}

func (pm *PromMetric) Equal(pm2 *PromMetric) bool {
	return pm.k == pm2.k
}

func (pm *PromMetric) Equals(pm2 *PromMetric) bool {
	return pm.k == pm2.k
}
