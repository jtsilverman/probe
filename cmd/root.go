package cmd

import (
	"github.com/spf13/cobra"

	"github.com/jtsilverman/probe/internal/config"
)

var cfg config.Config

var rootCmd = &cobra.Command{
	Use:   "probe",
	Short: "AI-powered code review CLI",
	Long:  "Zero-config CLI that reviews code diffs using Claude. Catches bugs, security issues, and AI-generated code anti-patterns that linters miss.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.Load()
	},
}

func Execute() error {
	return rootCmd.Execute()
}
