package stats

import (
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
)

type Stats struct {
	client statsd.ClientInterface

	prefix string
	tags   Tags
}

type Tags map[string]interface{}

func New(client statsd.ClientInterface) Stats {
	return Stats{
		client: client,
		tags:   Tags{},
	}
}

func (s Stats) WithTags(tags Tags) Stats {
	s.tags = mergeTags([]Tags{s.tags, tags})
	return s
}

func (s Stats) Count(name string, value int64, tags Tags, rate float64) {
	s.client.Count(name, value, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) Incr(name string, tags Tags, rate float64) {
	s.client.Incr(name, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) Gauge(name string, value float64, tags Tags, rate float64) {
	s.client.Gauge(name, value, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func mergeTags(tags []Tags) Tags {
	mergedTags := make(Tags, 0)
	for _, tagGroup := range tags {
		if tagGroup == nil {
			continue
		}
		for k, v := range tagGroup {
			if v == nil {
				continue
			}
			mergedTags[k] = v
		}
	}
	return mergedTags
}

func convertTags(tags Tags) []string {
	var newTags []string
	for k, v := range tags {
		newTags = append(newTags, fmt.Sprintf("%s:%v", k, v))
	}
	return newTags
}
