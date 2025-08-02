package cmd

import "github.com/spf13/cobra"

func AddCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(analyzeCmd)
	rootCmd.AddCommand(transcodeCmd)
}