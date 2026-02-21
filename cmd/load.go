/*
Copyright © 2026 Paulo Suderio
*/
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/suderio/dndsl/internal/engine"
	"github.com/suderio/dndsl/internal/persistence"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load [world_name] [campaign_name]",
	Short: "Load a campaign and print the current GameState",
	Long: `Reads the log.jsonl of a specific campaign and calculates 
the GameState via the event Projector.`,
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
			fmt.Printf("Error finding campaign: %v\n", err)
			os.Exit(1)
		}
		defer store.Close()

		events, err := store.Load()
		if err != nil {
			fmt.Printf("Error reading event log: %v\n", err)
			os.Exit(1)
		}

		projector := engine.NewProjector()
		state, err := projector.Build(events)
		if err != nil {
			fmt.Printf("Error building state: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Successfully loaded campaign!\n")
		fmt.Printf("Processed %d events.\n", len(events))
		fmt.Printf("Active Entities: %d\n", len(state.Entities))
		for id, ent := range state.Entities {
			fmt.Printf("- %s (HP: %d/%d)\n", id, ent.HP, ent.MaxHP)
		}
	},
}

func init() {
	campaignCmd.AddCommand(loadCmd)
}
