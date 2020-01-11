package main

type MetricType int

const (
	gaugeMetric = iota + 1
	countMetric
	histogramMetric
	distributionMetric
	setMetric
	timingMetric
)

var (
	//gaugeSymbol        = []byte("g")
	//countSymbol        = []byte("c")
	//histogramSymbol    = []byte("h")
	//distributionSymbol = []byte("d")
	//setSymbol          = []byte("s")
	//timingSymbol       = []byte("ms")

	metricSymbol2Code = map[byte]MetricType{
		'g': gaugeMetric,
		'c': countMetric,
		'h': histogramMetric,
		'd': distributionMetric,
		's': setMetric,
		't': timingMetric,
	}
)
