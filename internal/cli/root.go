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
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add subcommands to root command
	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(postCmd)
	RootCmd.AddCommand(runCmd)
	RootCmd.AddCommand(testCmd)
}
