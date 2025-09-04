/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package controller

import (
	"fmt"
	"log/slog"

	"github.com/tschaefer/finch/internal/config"
	"github.com/tschaefer/finch/internal/model"
)

type Controller interface {
	ControllerAgent
}

type controller struct {
	config config.Config
	model  model.Model
}

func New(model model.Model, cfg config.Config) Controller {
	slog.Debug("Initializing Controller", "model", fmt.Sprintf("%+T", model), "config", fmt.Sprintf("%+T", cfg))

	return &controller{
		model:  model,
		config: cfg,
	}
}
