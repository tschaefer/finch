/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package run

import (
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
	Cmd.Flags().StringP("stack.config-file", "", "/var/lib/finch/finch.json", "Config file of the stack")
}

func runCmd(cmd *cobra.Command, args []string) {
	listen, _ := cmd.Flags().GetString("server.listen-address")
	config, _ := cmd.Flags().GetString("stack.config-file")

	manager, err := manager.New(config)
	cobra.CheckErr(err)

	manager.Run(listen)
}
