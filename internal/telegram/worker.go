package telegram

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// CommandResult holds the output of a command execution.
type CommandResult struct {
	Messages []string
}

// Executor defines the interface for running DSL commands.
type Executor interface {
	Execute(input string) (*CommandResult, error)
}

// Bot handles the integration between Telegram and the Ancient Draconic session
type Bot struct {
	client       *Client
	executor     Executor
	chatID       int64
	userMap      map[int64]string // telegram_user_id -> draconic_actor_id
	lastUpdateID int
}

// NewBot initializes a new follower bot
func NewBot(token string, chatID int64, userMap map[int64]string, exec Executor) *Bot {
	return &Bot{
		client:       NewClient(token),
		executor:     exec,
		chatID:       chatID,
		userMap:      userMap,
		lastUpdateID: viper.GetInt("tg_last_update_id"),
	}
}

// Start launches the long-polling loop
func (b *Bot) Start() {
	log.Printf("Telegram bot started for chat %d", b.chatID)
	for {
		updates, err := b.client.GetUpdates(b.lastUpdateID+1, 25)
		if err != nil {
			log.Printf("Error fetching updates: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		for _, update := range updates {
			if update.UpdateID > b.lastUpdateID {
				b.lastUpdateID = update.UpdateID
				// Persist last_update_id
				viper.Set("tg_last_update_id", b.lastUpdateID)
				_ = viper.WriteConfig() // Ignore error if config file doesn't exist yet
			}

			if update.Message != nil {
				b.handleMessage(update.Message)
			}
		}
	}
}

func (b *Bot) handleMessage(msg *Message) {
	// 1. Verify Chat ID
	if msg.Chat.ID != b.chatID {
		// Ignore messages from other chats as per requirement
		return
	}

	// 2. Ignore non-commands
	if !strings.HasPrefix(msg.Text, "/") {
		return
	}

	// 3. Command Translation
	rawText := strings.TrimPrefix(msg.Text, "/")
	parts := strings.Fields(rawText)
	if len(parts) == 0 {
		return
	}

	actorID, ok := b.userMap[msg.From.ID]
	if !ok {
		b.client.SendMessage(b.chatID, fmt.Sprintf("User %s (%d) is not registered in this campaign.", msg.From.FirstName, msg.From.ID))
		return
	}

	// Inject by: <actor> after the first word (command)
	translatedCmd := parts[0] + " by: " + actorID + " " + strings.Join(parts[1:], " ")

	// 4. Execution
	result, err := b.executor.Execute(translatedCmd)
	if err != nil {
		b.client.SendMessage(b.chatID, fmt.Sprintf("Error: %v", err))
		return
	}

	for _, msg := range result.Messages {
		if msg != "" {
			b.client.SendMessage(b.chatID, fmt.Sprintf("*%s*", msg))
		}
	}
}
