package main

import (
	"context"
	"github.com/hightouchio/passage/log"
	"github.com/hightouchio/passage/telemetry"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

const (
	ConfigTelemetryRuntimeMetricsEnabled = "telemetry.runtime_metrics.enabled"
	ConfigTelemetryProfilerEnabled       = "telemetry.profiler.enabled"
	ConfigTelemetryProfileTypes          = "telemetry.profiler.profile_types"
)

// runTelemetry sets up telemetry for the application
func runTelemetry(lc fx.Lifecycle, log *log.Logger, config *viper.Viper) error {
	config.SetDefault(ConfigTelemetryRuntimeMetricsEnabled, false)
	if config.GetBool(ConfigTelemetryRuntimeMetricsEnabled) {
		stopTracer := telemetry.Tracer(telemetry.TracerSettings{
			RuntimeMetrics: true,
		})

		// Stop the profiler on application shutdown
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				stopTracer()
				return nil
			},
		})
	}

	config.SetDefault(ConfigTelemetryProfilerEnabled, false)
	config.SetDefault(ConfigTelemetryProfileTypes, []string{"cpu", "heap"})

	// Enable profiling
	if config.GetBool(ConfigTelemetryProfilerEnabled) {
		// Configure profile types
		settings := telemetry.ProfilerSettings{}
		for _, profileType := range config.GetStringSlice(ConfigTelemetryProfileTypes) {
			switch profileType {
			case "cpu":
				settings.ProfileCPU = true
			case "heap":
				settings.ProfileHeap = true
			case "block":
				settings.ProfileBlock = true
			case "mutex":
				settings.ProfileMutex = true
			case "goroutine":
				settings.ProfileGoroutine = true
			default:
				log.Warnf("unknown profile type %s", profileType)
			}
		}

		// Run the profiler
		stopProfiler, err := telemetry.Profiler(settings)
		if err != nil {
			return errors.Wrap(err, "could not start profiler")
		}

		// Stop the profiler on application shutdown
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				stopProfiler()
				return nil
			},
		})
	}
}
