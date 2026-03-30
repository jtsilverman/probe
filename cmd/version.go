package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.2.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print probe version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("probe v%s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
