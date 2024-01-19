package telemetry

import (
	"github.com/pkg/errors"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/profiler"
)

type ProfilerSettings struct {
	ProfileCPU       bool
	ProfileHeap      bool
	ProfileBlock     bool
	ProfileMutex     bool
	ProfileGoroutine bool
}

// Profiler starts the Datadog continuous profiler
func Profiler(settings ProfilerSettings) (stop func(), err error) {
	var profileTypes []profiler.ProfileType

	if settings.ProfileCPU {
		profileTypes = append(profileTypes, profiler.CPUProfile)
	}
	if settings.ProfileHeap {
		profileTypes = append(profileTypes, profiler.HeapProfile)
	}
	if settings.ProfileBlock {
		profileTypes = append(profileTypes, profiler.BlockProfile)
	}
	if settings.ProfileMutex {
		profileTypes = append(profileTypes, profiler.MutexProfile)
	}
	if settings.ProfileGoroutine {
		profileTypes = append(profileTypes, profiler.GoroutineProfile)
	}

	if err := profiler.Start(profiler.WithProfileTypes(profileTypes...)); err != nil {
		return nil, errors.Wrap(err, "could not start profiler")
	}

	return func() {
		profiler.Stop()
	}, nil
}

type TracerSettings struct {
	RuntimeMetrics bool
}

func Tracer(settings TracerSettings) (stop func()) {
	var options []tracer.StartOption

	if settings.RuntimeMetrics {
		options = append(options, tracer.WithRuntimeMetrics())
	}

	tracer.Start()
	return func() {
		tracer.Stop()
	}
}
