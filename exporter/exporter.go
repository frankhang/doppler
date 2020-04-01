package exporter

import (
	"fmt"
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
	cache c.Cache
	//cache sync.Map
	bucketsForMilliseconds []float64
}

func NewPromExporter() *PromExporter {
	cache := c.New(
		c.WithExpireAfterAccess(60*time.Minute),
		c.WithRemovalListener(onRemoval),
	)

	exporter := &PromExporter{
		cache: cache,
		bucketsForMilliseconds: prometheus.ExponentialBuckets(0.1, 1.6, 32),
	}
	return exporter
}

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

func (e *PromExporter) loadMetric(key c.Key) (value c.Value, err error) {
	var pm *PromMetric
	var ok bool
	if pm, ok = key.(*PromMetric); !ok {
		err = errors.New("loadMetric: Unexcepted k type")
		return
	}

	var collector prometheus.Collector
	switch pm.Symbol {
	case GaugeSymbol:
		collector = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)

		err = prometheus.Register(collector)

	case CountSymbol:
		collector = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)
		err = prometheus.Register(collector)
	case HistogramSymbol:
		collector = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    pm.Name,
				Help:    pm.Name,
				Buckets: e.bucketsForMilliseconds,
			},
			pm.LabelNames)
		err = prometheus.Register(collector)
	case SummarySymbol:
		collector = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name: pm.Name,
				Help: pm.Name,
			},
			pm.LabelNames)
		err = prometheus.Register(collector)
	default:
		err = errors.New(fmt.Sprintf("loadMetric: Unsupported symbol, %s", pm))
	}

	if err == nil { //register successfully
		value = collector

	} else { //register error

		if reg, already := err.(prometheus.AlreadyRegisteredError); already {
			logutil.BgLogger().Info("loadMetric: already registered", zap.Reflect("collector", reg.ExistingCollector))

			value = reg.ExistingCollector
			err = nil
		} else {
			logutil.BgLogger().Error("loadMetric: register error", zap.Error(err))
			err = errors.Trace(err)
		}
	}

	return
}

func (e *PromExporter) Export(sample *metrics.MetricSample) (err error) {

	ps := NewPromSample(sample)

	var value interface{}
	var ok bool

	key := ps.metric.String()
	if value, ok = e.cache.GetIfPresent(key); !ok {
		if value, err = e.loadMetric(ps.metric); err != nil {
			err = errors.Trace(err)
			return
		}
		e.cache.Put(key, value)
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
