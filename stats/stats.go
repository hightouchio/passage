package stats

import (
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"strings"
)

type Stats struct {
	client statsd.ClientInterface

	prefix    string
	tags      Tags
	eventTags Tags
}

type Tags map[string]interface{}

func New(client statsd.ClientInterface) Stats {
	return Stats{
		client:    client,
		tags:      Tags{},
		eventTags: Tags{},
	}
}

func (s Stats) WithPrefix(new string) Stats {
	s.prefix = joinPrefixes(s.prefix, new)
	return s
}

func (s Stats) WithTags(tags Tags) Stats {
	s.tags = mergeTags([]Tags{s.tags, tags})
	return s
}

func (s Stats) WithEventTags(tags Tags) Stats {
	s.eventTags = mergeTags([]Tags{s.eventTags, tags})
	return s
}

type Event struct {
	statsd.Event
	Tags Tags
}

func (s Stats) Count(name string, value int64, tags Tags, rate float64) {
	s.client.Count(joinPrefixes(s.prefix, name), value, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) Incr(name string, tags Tags, rate float64) {
	s.client.Incr(joinPrefixes(s.prefix, name), convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) Gauge(name string, value float64, tags Tags, rate float64) {
	s.client.Gauge(joinPrefixes(s.prefix, name), value, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func joinPrefixes(prefixes ...string) string {
	newPrefixes := []string{}
	for _, v := range prefixes {
		if v != "" {
			newPrefixes = append(newPrefixes, v)
		}
	}
	return strings.Join(newPrefixes, ".")
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
