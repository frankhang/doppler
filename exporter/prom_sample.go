package exporter

import (
	"bytes"
	"fmt"
	"github.com/frankhang/doppler/metrics"
	"github.com/frankhang/doppler/util"
	"github.com/frankhang/util/hack"

	"strings"
)

const (
	blankStr = "nil"
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

	tags := make([]string, 0, len(s.Tags)+3)
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
		var method, path string
		//sort.Strings(tags)
		tags = util.SortUniqInPlace(tags)
		for _, tag := range tags {
			tagPair := strings.Split(tag, ":")
			if len(tagPair) != 2 {
				continue
			}
			//tagName := strings.TrimSpace(tagPair[0])
			tagName := normalize(tagPair[0])
			tagValue := strings.TrimSpace(tagPair[1])
			//if len(tagName) == 0 || len(tagValue) == 0 {
			//	continue
			//}

			if len(tagName) == 0 {
				continue
			}
			if len(tagValue) == 0 {
				tagValue = blankStr
			}
			pm.LabelNames = append(pm.LabelNames, tagName)
			ps.LableValues = append(ps.LableValues, tagValue)

			if tagName == "method" {
				method = tagValue
			} else if tagName == "path" {
				path = tagValue
			}

		}
		if len(method) > 0 && len(path) > 0 {
			pm.LabelNames = append(pm.LabelNames, "apiname")
			ps.LableValues = append(ps.LableValues, fmt.Sprintf("%s %s", method, path))
		}
	}
	pm.GenerateKey()

	return ps

}

func NewPromSampleFromServiceCheck(sc *metrics.ServiceCheck) *PromSample {
	pm := &PromMetric{}
	ps := &PromSample{metric: pm}

	metricName, labelValue := generateMetricWithLabel(sc.CheckName)

	pm.Symbol = CountSymbol
	pm.Name = metricName
	ps.Value = float64(sc.Status)

	switch sc.Status {
	case metrics.ServiceCheckOK:
		ps.Value = 1
	default:
		//return nil
		ps.Value = 0
	}

	tags := make([]string, 0, len(sc.Tags)+2)
	tags = append(tags, sc.Tags...)
	tags = append(tags, fmt.Sprintf("_service_:%s", labelValue))

	host := strings.TrimSpace(sc.Host) //host of client
	if host != "" {
		tags = append(tags, fmt.Sprintf("hostname:%s", host))
	}

	if len(tags) > 0 {
		//sort.Strings(tags)
		tags = util.SortUniqInPlace(tags)
		for _, tag := range tags {
			tagPair := strings.Split(tag, ":")
			if len(tagPair) != 2 {
				continue
			}
			//tagName := strings.TrimSpace(tagPair[0])
			tagName := normalize(tagPair[0])
			tagValue := strings.TrimSpace(tagPair[1])
			//if len(tagName) == 0 || len(tagValue) == 0 {
			//	continue
			//}

			if len(tagName) == 0 {
				continue
			}
			if len(tagValue) == 0 {
				tagValue = blankStr
			}
			pm.LabelNames = append(pm.LabelNames, tagName)
			ps.LableValues = append(ps.LableValues, tagValue)

		}
	}
	pm.GenerateKey()

	return ps

}

func generateMetricWithLabel(checkName string) (string, string) {
	s1, s2 := split(checkName)

	if s1 == "datadog" {
		s1 = "doppler"
	}
	if len(s2) == 0 { //only has s1
		//return fmt.Sprintf("_sc_%s", s1), s1
		return "_sc_", s1
	}

	if len(s1) == 0 { //only has s2
		//return fmt.Sprintf("_sc_%s", s2), s2
		return "_sc_", s2
	}
	//has s1 and s2

	return fmt.Sprintf("_sc_%s", s1), s2

}

func split(s string) (string, string) {
	s = normalize(s)
	if len(s) == 0 {
		return "", ""
	}
	cn := hack.Slice(s)
	i := bytes.IndexByte(cn, '_')
	if i < 0 {
		return s, ""
	}

	return hack.String(cn[:i]), hack.String(cn[i+1:])

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
