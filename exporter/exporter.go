package exporter

import (
	"github.com/frankhang/util/errors"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
	"time"

	"github.com/frankhang/doppler/metrics"
	//"github.com/prometheus/client_golang/prometheus"
	//c "github.com/allegro/bigcache"
	//c "github.com/patrickmn/go-cache"
	c "github.com/goburrow/cache"

	"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promhttp"

)
var (
	Exporter *PromExporter
)
type PromExporter struct {
	cache c.LoadingCache
	//cache sync.Map
}

func NewPromExporter() (*PromExporter) {
	cache := c.NewLoadingCache(loadMetric,
		c.WithExpireAfterAccess(60*time.Second),
		c.WithRemovalListener(onRemoval),
		)

	exporter := &PromExporter{
		cache: cache,
	}
	return exporter
}

//func onRemoval(k c.Key, value c.Value) {
//	var pm *PromMetric
//	var ok bool
//	if pm, ok = k.(*PromMetric); !ok {
//		logutil.BgLogger().Error("onRemoval: Unexcepted k type")
//		return
//	}
//
//	removed := prometheus.Unregister(pm.collector)
//	logutil.BgLogger().Debug("onRemoval", zap.Bool("removed", removed))
//}


func onRemoval(key c.Key, value c.Value) {

	var collector prometheus.Collector
	var ok bool
	if collector, ok = value.(prometheus.Collector); !ok {
		logutil.BgLogger().Error("onRemoval: Unexcepted k type")
		return
	}

	removed := prometheus.Unregister(collector)
	logutil.BgLogger().Debug("onRemoval", zap.Bool("removed", removed))
}

func loadMetric(key c.Key) (value c.Value, err error) {
	var pm *PromMetric
	var ok bool
	if pm, ok = key.(*PromMetric); !ok {
		err = errors.New("loadMetric: Unexcepted k type")
		return
	}


	switch pm.Symbol {
	case GaugeSymbol:
		collector := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)
		if err =prometheus.Register(collector); err != nil {
			err = errors.Trace(err)
			return
		}
		value = collector
	case CountSymbol:
		collector := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)
		if err =prometheus.Register(collector); err != nil {
			err = errors.Trace(err)
			return
		}
		value = collector
	case HistogramSymbol:
		collector := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:pm.Name,
				Help:pm.Name,
			},
			pm.LabelNames)
		if err =prometheus.Register(collector); err != nil {
			err = errors.Trace(err)
			return
		}
		value = collector
	case SummarySymbol:
		collector := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)
		if err =prometheus.Register(collector); err != nil {
			err = errors.Trace(err)
			return
		}
		value = collector
	default:


	}


	return
}

func (e *PromExporter) Export(sample *metrics.MetricSample) (err error) {

	ps := NewPromSample(sample)

	var value interface{}


	if value, err = e.cache.Get(ps.metric); err != nil {
		err = errors.Trace(err)
		return
	}



	switch collector := value.(type) {
	case *prometheus.GaugeVec:
		collector.WithLabelValues(ps.LableValues...).Set(ps.Value)
	case *prometheus.CounterVec:
		collector.WithLabelValues(ps.LableValues...).Add(ps.Value)
	case *prometheus.HistogramVec:
		collector.WithLabelValues(ps.LableValues...).Observe(ps.Value)
	case *prometheus.SummaryVec:
		collector.WithLabelValues(ps.LableValues...).Observe(ps.Value)
	default:
		err = errors.New("export: Unexcepted collector type")

	}


	return
}
