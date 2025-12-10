/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package profiler

import (
	"log/slog"
	"runtime"

	"github.com/grafana/pyroscope-go"
	"github.com/tschaefer/finch/internal/config"
)

const (
	defaultServerAddress = "http://pyroscope:4040"
)

type Profiler interface {
	Start() error
	Stop() error
}

type profiler struct {
	instance *pyroscope.Profiler
	config   pyroscope.Config
}

func New(config *config.Config, logging bool) Profiler {
	slog.Debug("Initializing Pyroscope profiler", "serverAddress", config.Profiler(), "logging", logging)

	serverAddress := config.Profiler()
	if serverAddress == "" {
		serverAddress = defaultServerAddress
	}

	var logger pyroscope.Logger
	if logging {
		logger = pyroscope.StandardLogger
	} else {
		logger = nil
	}

	cfg := pyroscope.Config{
		ApplicationName: "finch",
		ServerAddress:   serverAddress,
		Logger:          logger,
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

func (p *profiler) Start() error {
	slog.Debug("Starting Pyroscope profiler", "config", p.config)

	runtime.SetMutexProfileFraction(5)
	runtime.SetBlockProfileRate(5)

	profiler, err := pyroscope.Start(p.config)
	if err != nil {
		p.instance = nil
		return err
	}
	p.instance = profiler

	return nil
}

func (p *profiler) Stop() error {
	slog.Debug("Stopping Pyroscope profiler")

	if p.instance == nil {
		return nil
	}

	return p.instance.Stop()
}
