package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/suderio/ancient-draconic/internal/data"
	"github.com/suderio/ancient-draconic/internal/persistence"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var (
	tgChatID    string
	tgUserPairs []string
)

type TelegramCampaignConfig struct {
	ChatID string            `yaml:"chat_id"`
	Users  map[string]string `yaml:"users"` // user_id -> actor_id
}

var telegramCmd = &cobra.Command{
	Use:   "telegram [world_name] [campaign_name]",
	Short: "Configure Telegram settings for a campaign",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		worldDir, _ := campaignCmd.PersistentFlags().GetString("world_dir")
		campaignDir, _ := campaignCmd.PersistentFlags().GetString("campaign_dir")

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
		campaignPath := manager.GetCampaignPath("", campaignDir)
		if _, err := os.Stat(campaignPath); os.IsNotExist(err) {
			fmt.Printf("Error: campaign directory %s does not exist. Run 'campaign create' first.\n", campaignPath)
			os.Exit(1)
		}

		configPath := filepath.Join(campaignPath, "telegram.yaml")
		config := TelegramCampaignConfig{
			Users: make(map[string]string),
		}

		// Load existing config if it exists
		if _, err := os.Stat(configPath); err == nil {
			f, _ := os.Open(configPath)
			yaml.NewDecoder(f).Decode(&config)
			f.Close()
		}

		if tgChatID == "" {
			fmt.Println("---")
			fmt.Println("How to get your Telegram Chat ID:")
			fmt.Println("1. Add your bot to the group.")
			fmt.Println("2. Send a message in the group (e.g., /start).")
			fmt.Println("3. Access https://api.telegram.org/bot<TOKEN>/getUpdates in your browser.")
			fmt.Println("4. Look for the 'chat' object and its 'id' field (it usually starts with a minus sign).")
			fmt.Println("---")
			fmt.Print("chat_id: ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				tgChatID = strings.TrimSpace(scanner.Text())
			}
		}

		if tgChatID != "" {
			config.ChatID = tgChatID
		}
		if len(tgUserPairs) > 0 {
			campaignData := filepath.Join(campaignPath, "data")
			worldData := filepath.Join(worldDir, "data")
			loader := data.NewLoader([]string{campaignPath, campaignData, worldDir, worldData})

			for _, pair := range tgUserPairs {
				parts := strings.Split(pair, ":")
				if len(parts) != 2 {
					fmt.Printf("Warning: invalid user pair format '%s'. Expected 'username:user_id'\n", pair)
					continue
				}
				username := parts[0]
				userID := parts[1]

				// Validate username
				_, err := loader.LoadCharacter(username)
				if err != nil {
					_, err = loader.LoadMonster(username)
				}

				if err != nil {
					fmt.Printf("Warning: character or monster '%s' not found in data directories. Users may be unable to command it.\n", username)
				}

				config.Users[userID] = username
			}
		}

		// Save config
		f, err := os.Create(configPath)
		if err != nil {
			fmt.Printf("Error creating config file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		encoder := yaml.NewEncoder(f)
		if err := encoder.Encode(config); err != nil {
			fmt.Printf("Error encoding config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Telegram campaign configuration saved to %s\n", configPath)
	},
}

func init() {
	campaignCmd.AddCommand(telegramCmd)
	telegramCmd.Flags().StringVarP(&tgChatID, "chat_id", "c", "", "Telegram group chat ID")
	telegramCmd.Flags().StringSliceVarP(&tgUserPairs, "user", "u", []string{}, "Map Telegram user_id to actor_id (format: username:user_id)")
}
