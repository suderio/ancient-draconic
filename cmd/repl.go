/*
Copyright © 2026 Paulo Suderio
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"dndsl/internal/persistence"
	"dndsl/internal/session"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var replCmd = &cobra.Command{
	Use:   "repl [world_name] [campaign_name]",
	Short: "Start the interactive REPL shell",
	Long: `Starts the read-eval-print loop for encountering sequences and issuing commands.
Usage:
	> roll :by Somebody 3d6k1+1`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		worldDir, _ := cmd.Flags().GetString("world_dir")
		campaignDir, _ := cmd.Flags().GetString("campaign_dir")

		worldName := ""
		campaignName := ""
		if len(args) >= 1 {
			worldName = args[0]
		}
		if len(args) >= 2 {
			campaignName = args[1]
		}

		if worldDir == "" {
			if worldName == "" {
				fmt.Println("Error: must specify either [world_name] argument or --world_dir flag")
				os.Exit(1)
			}
			worldsDir := viper.GetString("worlds_dir")
			if worldsDir == "" {
				worldsDir = "./worlds"
			}
			worldDir = filepath.Join(worldsDir, worldName)
		}

		if campaignDir == "" {
			if campaignName == "" {
				fmt.Println("Error: must specify either [campaign_name] argument or --campaign_dir flag")
				os.Exit(1)
			}
			campaignDir = campaignName
		}

		manager := persistence.NewCampaignManager(worldDir)
		store, err := manager.Load("", campaignDir)
		if err != nil {
			fmt.Printf("Error: %v\nDid you run `campaign create` first?\n", err)
			os.Exit(1)
		}
		defer store.Close()

		campaignData := filepath.Join(manager.GetCampaignPath("", campaignDir), "data")
		worldData := filepath.Join(worldDir, "data")

		rootData := viper.GetString("data_dir")
		if rootData == "" {
			rootDir, _ := os.Getwd()
			rootData = filepath.Join(rootDir, "data")
		}

		dataDirs := []string{campaignData, worldData, rootData}

		app, err := session.NewSession(dataDirs, store)
		if err != nil {
			fmt.Printf("Failed to bootstrap game session: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Starting REPL for '%s/%s'...\nType 'exit' or 'quit' to leave.\n\n", worldDir, campaignDir)

		maybeStartBot(app, worldDir, campaignDir)

		if err := RunTUI(app, worldDir, campaignDir); err != nil {
			fmt.Printf("Fatal TUI Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(replCmd)
	replCmd.Flags().StringP("world_dir", "w", "", "Location of the world directory (can be relative or absolute path)")
	replCmd.Flags().StringP("campaign_dir", "c", "", "Name of the campaign directory inside the world directory")
}
