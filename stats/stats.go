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

type Tags map[string]any

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

func (s Stats) SimpleEvent(title string) {
	s.Event(Event{
		Event: *statsd.NewEvent(title, ""),
	})
}

func (s Stats) ErrorEvent(title string, err error) {
	s.Event(Event{
		Event: statsd.Event{
			Title:     title,
			Text:      err.Error(),
			AlertType: statsd.Error,
		},
	})
}

func (s Stats) Event(event Event) {
	tags := mergeTags([]Tags{s.tags, s.eventTags, event.Tags})

	statsEvent := event.Event
	statsEvent.Title = joinPrefixes(s.prefix, event.Title)
	statsEvent.Tags = convertTags(tags)

	// report stats event
	s.client.Event(&statsEvent)

	// prepare for logging
	var level logrus.Level
	switch statsEvent.AlertType {
	case statsd.Error:
		level = logrus.ErrorLevel
	case statsd.Warning:
		level = logrus.WarnLevel
	default:
		level = logrus.InfoLevel
	}

	fields := logrus.Fields(tags)
	if statsEvent.AlertType == statsd.Error {
		fields["error"] = statsEvent.Text
	}

	s.logger.WithFields(logrus.Fields(tags)).Log(level, statsEvent.Title)
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
