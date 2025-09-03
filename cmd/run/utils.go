/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package run

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

func completeServerLogLevel(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{"debug", "info", "warn", "error"}, cobra.ShellCompDirectiveDefault
}

func completeServerLogFormat(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
	return []cobra.Completion{"structured", "json", "text"}, cobra.ShellCompDirectiveDefault
}

func setLogger(logLevel string, logFormat string) {
	var leveler slog.Leveler
	switch logLevel {
	case "debug":
		leveler = slog.LevelDebug
	case "info":
		leveler = slog.LevelInfo
	case "warn":
		leveler = slog.LevelWarn
	case "error":
		leveler = slog.LevelError
	default:
		cobra.CheckErr("unknown log level")
	}

	opts := &slog.HandlerOptions{
		Level: leveler,
	}

	var logger *slog.Logger
	switch logFormat {
	case "structured":
		logger = slog.New(slog.NewTextHandler(os.Stdout, opts))
	case "json":
		logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	case "text":
		return
	default:
		cobra.CheckErr("unknown log format")
	}
	slog.SetDefault(logger)
}
