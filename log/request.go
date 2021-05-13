package log

import "github.com/sirupsen/logrus"

// Request logs request/response operations on the server
func Request(log logrus.FieldLogger, eventName string, request interface{}, response interface{}, err error) {
	log = log.WithField("request", request)

	if err != nil {
		log = log.WithError(err)
	}
	if response != nil {
		log = log.WithField("response", response)
	}

	log.Info(eventName)
}
