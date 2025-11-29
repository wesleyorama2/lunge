// Copyright (c) 2025, Wesley Brown
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "lunge",
	Short:   "A powerful yet simple terminal-based HTTP client",
	Version: version,
	Long: `Lunge is a powerful yet simple terminal-based HTTP client written in Go
that combines curl's simplicity with Postman/Insomnia's power, with a
special emphasis on testing capabilities and response validation.`,
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, print help
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

// ExecuteWithExit is the same as Execute but exits the process on error.
// This is used by the main function to maintain backward compatibility.
func ExecuteWithExit() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Add subcommands to root command
	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(postCmd)
	RootCmd.AddCommand(putCmd)
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(testCmd)
	RootCmd.AddCommand(perfCmd)
}
