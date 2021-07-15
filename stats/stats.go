package stats

import (
	"fmt"
	"github.com/DataDog/datadog-go/statsd"
	"github.com/sirupsen/logrus"
	"strings"
)

type Stats struct {
	client statsd.ClientInterface
	logger *logrus.Logger

	prefix    string
	tags      Tags
	eventTags Tags
}

type Tags map[string]interface{}

func New(client statsd.ClientInterface, logger *logrus.Logger) Stats {
	return Stats{
		client:    client,
		logger:    logger,
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
	*statsd.Event
	Tags Tags
}

func (s Stats) Incr(name string, tags Tags, rate float64) {
	s.client.Incr(joinPrefixes(s.prefix, name), convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) Gauge(name string, value float64, tags Tags, rate float64) {
	s.client.Gauge(joinPrefixes(s.prefix, name), value, convertTags(mergeTags([]Tags{s.tags, tags})), rate)
}

func (s Stats) SimpleEvent(title string) {
	s.Event(Event{
		Event: statsd.NewEvent(title, ""),
	})
}

func (s Stats) Event(event Event) {
	statsEvent := event.GetEvent(s.prefix, s.tags, s.eventTags)
	s.client.Event(statsEvent)
	logFields := logrus.Fields(s.tags)
	logFields["status"] = event.AlertType
	s.logger.WithFields(logFields).Log(logrus.InfoLevel, event.Title)
}

func (e Event) GetEvent(prefix string, oldTags ...Tags) *statsd.Event {
	realEvent := e.Event
	realEvent.Title = joinPrefixes(prefix, e.Title)
	realEvent.Tags = convertTags(mergeTags(append(oldTags, e.Tags)))
	return realEvent
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
