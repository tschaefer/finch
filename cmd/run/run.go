/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package run

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/tschaefer/finch/internal/manager"
)

var Cmd = &cobra.Command{
	Use:   "run",
	Short: "Run Finch manager",
	Run:   runCmd,
}

func init() {
	Cmd.Flags().StringP("server.listen-address", "", "127.0.0.1:3000", "Address to listen on for traffic")
	Cmd.Flags().StringP("server.log-level", "", "info", "Log level (debug, info, warn, error)")
	Cmd.Flags().StringP("server.log-format", "", "structured", "Log format (structured, json)")
	Cmd.Flags().StringP("stack.config-file", "", "/var/lib/finch/finch.json", "Config file of the stack")

	_ = Cmd.RegisterFlagCompletionFunc("server.log-level", completeServerLogLevel)
	_ = Cmd.RegisterFlagCompletionFunc("server.log-format", completeServerLogFormat)
}

func runCmd(cmd *cobra.Command, args []string) {
	listen, _ := cmd.Flags().GetString("server.listen-address")
	config, _ := cmd.Flags().GetString("stack.config-file")
	logLevel, _ := cmd.Flags().GetString("server.log-level")
	logFormat, _ := cmd.Flags().GetString("server.log-format")

	setLogger(logLevel, logFormat)

	manager, err := manager.New(config)
	cobra.CheckErr(err)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	manager.Run(ctx, listen)
}
