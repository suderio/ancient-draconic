package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/suderio/dndsl/internal/dnd5eapi"

	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var defaultEndpoints = []string{
	"spells", "monsters", "classes", "ability-scores", "alignments",
	"backgrounds", "conditions", "damage-types", "equipment",
	"equipment-categories", "feats", "features", "languages",
	"magic-items", "magic-schools", "proficiencies", "races",
	"rule-sections", "rules", "skills", "subclasses", "subraces",
	"traits", "weapon-properties",
}

var initCmd = &cobra.Command{
	Use:    "init",
	Short:  "Initialize data by downloading SRD files from dnd5eapi",
	Long:   `Bootstraps the local game data environment by fetching 5e SRD data, transforming it, and storing locally for offline use.`,
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		dataDir, _ := cmd.Flags().GetString("data_dir_local")
		if dataDir == "" {
			rootDir, _ := os.Getwd()
			dataDir = filepath.Join(rootDir, "data")
		}

		force, _ := cmd.Flags().GetBool("force")

		// Determine which endpoints to run
		var targets []string
		anyFlagSelected := false
		for _, ep := range defaultEndpoints {
			flagVal, _ := cmd.Flags().GetBool(ep)
			if flagVal {
				targets = append(targets, ep)
				anyFlagSelected = true
			}
		}

		if !anyFlagSelected {
			targets = defaultEndpoints // If no flags passed, download all
		}

		fmt.Printf("Initializing SRD data to: %s\n", dataDir)

		client := dnd5eapi.NewClient(dataDir, force)

		totalBar := progressbar.Default(int64(len(targets)), "Overall Progress")

		for _, endpoint := range targets {
			fmt.Printf("\nFetching %s...\n", endpoint)

			list, err := client.FetchList(endpoint)
			if err != nil {
				fmt.Printf("Error fetching %s list: %v\n", endpoint, err)
				totalBar.Add(1)
				continue
			}

			if list.Count == 0 {
				totalBar.Add(1)
				continue
			}

			epBar := progressbar.Default(int64(list.Count), fmt.Sprintf("Downloading %s", endpoint))

			for _, itemRef := range list.Results {
				// We need to fetch the item
				// Throttle to respect the API
				time.Sleep(100 * time.Millisecond)

				// Skip if it exists and not force
				localPath := filepath.Join(dataDir, endpoint, fmt.Sprintf("%s.yaml", itemRef.Index))
				if !force {
					if _, err := os.Stat(localPath); err == nil {
						// Exists, skip
						epBar.Add(1)
						continue
					}
				}

				itemData, err := client.FetchItem(itemRef.URL)
				if err != nil {
					// We might try fetching via Index if URL is weird, but standard URL works
					itemData, err = client.FetchItem(fmt.Sprintf("/api/2014/%s/%s", endpoint, itemRef.Index))
					if err != nil {
						epBar.Add(1)
						continue
					}
				}

				transformed := client.Transform(itemData, endpoint)
				if err := client.SaveItem(endpoint, itemRef.Index, transformed); err != nil {
					fmt.Printf("\nFailed to save %s: %v\n", itemRef.Index, err)
				}
				epBar.Add(1)
			}
			totalBar.Add(1)
		}

		fmt.Println("\nData bootstrap complete!")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().Bool("force", false, "Force redownload of existing files")
	initCmd.Flags().String("data_dir_local", "", "Local data directory to save files to (internal fallback is still used by the app)")

	for _, ep := range defaultEndpoints {
		initCmd.Flags().Bool(ep, false, fmt.Sprintf("Download %s", ep))
	}

}
