/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT License, see LICENSE file in the project root for details.
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/tschaefer/finch/cmd/run"
	"github.com/tschaefer/finch/internal/version"
)

var rootCmd = &cobra.Command{
	Use:   "finch",
	Short: "Finch log stack manager.",
	Run: func(cmd *cobra.Command, args []string) {
		v, _ := cmd.Flags().GetBool("version")
		if v {
			version.Print()
			return
		}
		_ = cmd.Help()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			version.Print()
		},
	}

	rootCmd.AddCommand(run.Cmd)
	rootCmd.AddCommand(versionCmd)
}
