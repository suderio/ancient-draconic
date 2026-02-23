/*
Copyright © 2026 Paulo Suderio
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suderio/ancient-draconic/internal/persistence"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [world_name] [campaign_name]",
	Short: "Create a new local persistence layer in a world",
	Long: `Bootstraps a fresh append-only log record log.jsonl and dedicated 
data directories under worlds/<world_name>/<campaign_name> to 
securely track the history of an isolated encounter state.`,
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
		store, err := manager.Create("", campaignDir)
		if err != nil {
			fmt.Printf("Error creating campaign: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		fmt.Printf("Successfully created campaign!\n")
		fmt.Printf("Log file stored at: %s/log.jsonl\n", manager.GetCampaignPath("", campaignDir))
	},
}

func init() {
	campaignCmd.AddCommand(createCmd)
}
