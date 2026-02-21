package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var botToken string

// botCmd represents the bot command
var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Manage global bot configurations",
}

// telegramBotCmd represents the telegram subcommand of bot
var telegramBotCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Register a global Telegram bot",
	Run: func(cmd *cobra.Command, args []string) {
		if botToken == "" {
			fmt.Println("---")
			fmt.Println("Create your Telegram Bot & Get Token")
			fmt.Println("Open Telegram and search for the official @BotFather.")
			fmt.Println("Send the /newbot command and follow the prompts to name your bot and choose a unique username.")
			fmt.Println("BotFather will provide you with an HTTP API token. Store this token securely, as it is required for all API interactions. We will need it to configure dndsl.")
			fmt.Println("For testing in a group, add the bot to a group and ensure its privacy settings allow it to read all messages (this can be configured in BotFather's settings).")
			fmt.Println("---")
			fmt.Print("token: ")

			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				botToken = strings.TrimSpace(scanner.Text())
			}
		}

		if botToken != "" {
			viper.Set("telegram_token", botToken)
			err := viper.WriteConfig()
			if err != nil {
				// If config file doesn't exist, WriteConfig typically fails.
				// For simplicity, we could try WriteConfigAs if we knew where to put it,
				// but Cobra's initConfig already sets up paths.
				err = viper.SafeWriteConfig()
				if err != nil {
					// Fallback: try to write to $HOME/.dndsl.yaml
					home, _ := os.UserHomeDir()
					err = viper.WriteConfigAs(home + "/.dndsl.yaml")
				}
			}
			if err == nil {
				fmt.Println("Telegram bot token saved successfully.")
			} else {
				fmt.Printf("Error saving configuration: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(botCmd)
	botCmd.AddCommand(telegramBotCmd)

	telegramBotCmd.Flags().StringVarP(&botToken, "token", "t", "", "Telegram bot API token")
}
