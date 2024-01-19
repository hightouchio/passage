package telemetry

import (
	"github.com/pkg/errors"
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

	if err := profiler.Start(profiler.WithProfileTypes()); err != nil {
		return nil, errors.Wrap(err, "could not start profiler")
	}

	return func() {
		profiler.Stop()
	}, nil
}
