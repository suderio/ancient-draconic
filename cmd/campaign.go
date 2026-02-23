/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// campaignCmd represents the campaign command
var campaignCmd = &cobra.Command{
	Use:   "campaign",
	Short: "Manage draconic world states and logging",
	Long: `The campaign command allows you to construct isolated event-sourced
journals for varied D&D environments.

Use subcommands 'create' and 'load' to manipulate targeted JSONL logs
isolated within a specific world's namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("campaign called")
	},
}

func init() {
	rootCmd.AddCommand(campaignCmd)

	campaignCmd.PersistentFlags().StringP("world_dir", "w", "", "Location of the world directory (can be relative or absolute path)")
	campaignCmd.PersistentFlags().StringP("campaign_dir", "c", "", "Name of the campaign directory inside the world directory")
}
