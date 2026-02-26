/*
Copyright © 2026 Paulo Suderio
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suderio/ancient-draconic/internal/session"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var replCmd = &cobra.Command{
	Use:   "repl [world_name] [campaign_name]",
	Short: "Start the interactive REPL shell",
	Long: `Starts the read-eval-print loop for encountering sequences and issuing commands.
Usage:
	> roll by: Somebody dice: 3d6`,
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

		// Use session.CampaignManager for path resolution
		manager := session.NewCampaignManager(worldDir)
		campaignRoot := manager.GetCampaignPath("", campaignDir)
		campaignData := filepath.Join(campaignRoot, "data")
		worldData := filepath.Join(worldDir, "data")

		// Include root directories so manifest.yaml can be found at the campaign
		// or world root level, not only inside a /data subdirectory.
		dataDirs := []string{campaignRoot, campaignData, worldDir, worldData}

		// Store path for the new manifest event log
		storePath := manager.GetLogPath("", campaignDir)

		app, err := session.NewSession(dataDirs, storePath)
		if err != nil {
			fmt.Printf("Failed to bootstrap game session: %v\n", err)
			os.Exit(1)
		}
		defer app.Close()

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
