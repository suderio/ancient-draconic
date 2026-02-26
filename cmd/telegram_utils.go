package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/suderio/ancient-draconic/internal/session"
	"github.com/suderio/ancient-draconic/internal/telegram"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// maybeStartBot checks if telegram is configured and starts the background worker
func maybeStartBot(session *session.Session, worldDir, campaignDir string) {
	token := viper.GetString("telegram_token")
	if token == "" {
		return
	}

	// Campaign Path resolution (simplified as we already have worldDir and campaignDir)
	campaignPath := filepath.Join(worldDir, campaignDir)
	configPath := filepath.Join(campaignPath, "telegram.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	f, err := os.Open(configPath)
	if err != nil {
		return
	}
	defer f.Close()

	var config TelegramCampaignConfig
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		return
	}

	if config.ChatID == "" {
		return
	}

	chatID, err := strconv.ParseInt(config.ChatID, 10, 64)
	if err != nil {
		return
	}

	userMap := make(map[int64]string)
	for idStr, actorID := range config.Users {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err == nil {
			userMap[id] = actorID
		}
	}

	bot := telegram.NewBot(token, chatID, userMap, &botAdapter{session})

	// Run in background
	go bot.Start()
	fmt.Printf("[Telegram Bot] Active for chat %d\n", chatID)
}

// botAdapter bridges session.Session to the telegram.Executor interface.
type botAdapter struct {
	session *session.Session
}

func (a *botAdapter) Execute(input string) (*telegram.CommandResult, error) {
	events, err := a.session.Execute(input)
	if err != nil {
		return nil, err
	}
	result := &telegram.CommandResult{}
	for _, evt := range events {
		if msg := evt.Message(); msg != "" {
			result.Messages = append(result.Messages, msg)
		}
	}
	return result, nil
}
