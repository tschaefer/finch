/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"log/slog"
	"os"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/tschaefer/finch/internal/config"
)

type Profiler interface {
	Run() error
	Stop()
}

type profiler struct {
	instance *pyroscope.Profiler
	config   pyroscope.Config
}

func New(config config.Config, logging bool) Profiler {
	slog.Debug("Initializing Pyroscope profiler", "serverAddress", config.Profiler(), "logging", logging)

	var logger pyroscope.Logger
	if logging {
		logger = pyroscope.StandardLogger
	} else {
		logger = nil
	}

	cfg := pyroscope.Config{
		ApplicationName: "finch",
		ServerAddress:   config.Profiler(),
		Logger:          logger,
		Tags:            map[string]string{"hostname": os.Getenv("HOSTNAME")},
		ProfileTypes: []pyroscope.ProfileType{
			pyroscope.ProfileCPU,
			pyroscope.ProfileAllocObjects,
			pyroscope.ProfileAllocSpace,
			pyroscope.ProfileInuseObjects,
			pyroscope.ProfileInuseSpace,
			pyroscope.ProfileGoroutines,
			pyroscope.ProfileMutexCount,
			pyroscope.ProfileMutexDuration,
			pyroscope.ProfileBlockCount,
			pyroscope.ProfileBlockDuration,
		},
	}
	return &profiler{config: cfg}
}

func (p *profiler) Run() error {
	slog.Debug("Starting Pyroscope profiler", "config", p.config)

	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	profiler, err := pyroscope.Start(p.config)
	p.instance = profiler

	return err
}

func (p *profiler) Stop() {
	slog.Debug("Stopping Pyroscope profiler")
	if p.instance != nil {
		_ = p.instance.Stop()
	}
}
