package agent

import (
	"fmt"
	"github.com/frankhang/util/logutil"
	"go.uber.org/zap"
	"strings"

	"github.com/frankhang/doppler/metrics"
	"github.com/frankhang/doppler/tagger"
	"github.com/frankhang/doppler/tagger/collectors"
	"github.com/frankhang/doppler/util/kubernetes/kubelet"
)

type tagRetriever func(entity string, cardinality collectors.TagCardinality) ([]string, error)

var (
	hostTagPrefix     = "host:"
	entityIDTagPrefix = "dd.internal.entity_id:"

	getTags tagRetriever = tagger.Tag
)

func enrichTags(tags []string, defaultHostname string) ([]string, string) {
	if len(tags) == 0 {
		return nil, defaultHostname
	}

	extraTags := make([]string, 0, 8)
	host := defaultHostname

	n := 0
	for _, tag := range tags {
		if strings.HasPrefix(tag, hostTagPrefix) {
			host = tag[len(hostTagPrefix):]
		} else if strings.HasPrefix(tag, entityIDTagPrefix) {
			// currently only supported for pods
			entity := kubelet.KubePodTaggerEntityPrefix + tag[len(entityIDTagPrefix):]
			entityTags, err := getTags(entity, tagger.DogstatsdCardinality)
			if err != nil {
				logutil.BgLogger().Debug(fmt.Sprintf("Cannot get tags for entity %s", entity), zap.Error(err))
				continue
			}
			extraTags = append(extraTags, entityTags...)
		} else {
			tags[n] = tag
			n++
		}
	}
	tags = tags[:n]
	tags = append(tags, extraTags...)
	return tags, host
}

func enrichMetricType(dogstatsdMetricType metricType) metrics.MetricType {
	switch dogstatsdMetricType {
	case gaugeType:
		return metrics.GaugeType
	case countType:
		return metrics.CounterType
	case distributionType:
		return metrics.DistributionType
	case histogramType:
		return metrics.HistogramType
	case setType:
		return metrics.SetType
	case timingType:
		return metrics.HistogramType
	}
	return metrics.GaugeType
}

func enrichMetricSample(metricSample dogstatsdMetricSample, namespace string, namespaceBlacklist []string, defaultHostname string) metrics.MetricSample {
	metricName := metricSample.name
	if namespace != "" {
		blacklisted := false
		for _, prefix := range namespaceBlacklist {
			if strings.HasPrefix(metricName, prefix) {
				blacklisted = true
			}
		}
		if !blacklisted {
			metricName = namespace + metricName
		}
	}

	tags, hostname := enrichTags(metricSample.tags, defaultHostname)

	return metrics.MetricSample{
		Host:       hostname,
		Name:       metricName,
		Tags:       tags,
		Mtype:      enrichMetricType(metricSample.metricType),
		Value:      metricSample.value,
		SampleRate: metricSample.sampleRate,
		RawValue:   metricSample.setValue,
	}
}

func enrichEventPriority(priority eventPriority) metrics.EventPriority {
	switch priority {
	case priorityNormal:
		return metrics.EventPriorityNormal
	case priorityLow:
		return metrics.EventPriorityLow
	}
	return metrics.EventPriorityNormal
}

func enrichEventAlertType(dogstatsdAlertType alertType) metrics.EventAlertType {
	switch dogstatsdAlertType {
	case alertTypeSuccess:
		return metrics.EventAlertTypeSuccess
	case alertTypeInfo:
		return metrics.EventAlertTypeInfo
	case alertTypeWarning:
		return metrics.EventAlertTypeWarning
	case alertTypeError:
		return metrics.EventAlertTypeError
	}
	return metrics.EventAlertTypeSuccess
}

func enrichEvent(event dogstatsdEvent, defaultHostname string) *metrics.Event {
	tags, hostFromTags := enrichTags(event.tags, defaultHostname)

	enrichedEvent := &metrics.Event{
		Title:          event.title,
		Text:           event.text,
		Ts:             event.timestamp,
		Priority:       enrichEventPriority(event.priority),
		Tags:           tags,
		AlertType:      enrichEventAlertType(event.alertType),
		AggregationKey: event.aggregationKey,
		SourceTypeName: event.sourceType,
	}

	if len(event.hostname) != 0 {
		enrichedEvent.Host = event.hostname
	} else {
		enrichedEvent.Host = hostFromTags
	}
	return enrichedEvent
}

func enrichServiceCheckStatus(status serviceCheckStatus) metrics.ServiceCheckStatus {
	switch status {
	case serviceCheckStatusUnknown:
		return metrics.ServiceCheckUnknown
	case serviceCheckStatusOk:
		return metrics.ServiceCheckOK
	case serviceCheckStatusWarning:
		return metrics.ServiceCheckWarning
	case serviceCheckStatusCritical:
		return metrics.ServiceCheckCritical
	}
	return metrics.ServiceCheckUnknown
}

func enrichServiceCheck(serviceCheck dogstatsdServiceCheck, defaultHostname string) *metrics.ServiceCheck {
	tags, hostFromTags := enrichTags(serviceCheck.tags, defaultHostname)

	enrichedServiceCheck := &metrics.ServiceCheck{
		CheckName: serviceCheck.name,
		Ts:        serviceCheck.timestamp,
		Status:    enrichServiceCheckStatus(serviceCheck.status),
		Message:   serviceCheck.message,
		Tags:      tags,
	}

	if len(serviceCheck.hostname) != 0 {
		enrichedServiceCheck.Host = serviceCheck.hostname
	} else {
		enrichedServiceCheck.Host = hostFromTags
	}
	return enrichedServiceCheck
}
