package main

import (
	"fmt"
	"media-mgmt/cmd"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "media-mgmt",
	Short: "Media management and analysis tool",
	Long: `A comprehensive tool for analyzing and managing media files.
Supports video analysis, report generation, and various output formats.`,
}

func init() {
	cmd.AddCommands(rootCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}