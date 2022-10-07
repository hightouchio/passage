package main

import (
	"context"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

const healthcheckTimeout = 10 * time.Second

type healthcheck func(ctx context.Context) error

type healthcheckManager struct {
	healthchecks map[string]healthcheck
}

func newHealthcheckManager() *healthcheckManager {
	return &healthcheckManager{healthchecks: make(map[string]healthcheck)}
}

func (m *healthcheckManager) AddCheck(name string, h healthcheck) {
	m.healthchecks[name] = h
}

func (m *healthcheckManager) CheckHealth(ctx context.Context) error {
	cerr := make(chan error)
	go func() {
		for name, check := range m.healthchecks {
			if err := check(ctx); err != nil {
				cerr <- errors.Wrapf(err, "%s is unhealthy", name)
				return
			}
		}
		cerr <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-cerr:
		return err
	}
}

func (m *healthcheckManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// run healthchecks
	if err := m.CheckHealth(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
}
