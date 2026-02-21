package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Update represents a Telegram update
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message represents a Telegram message
type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// User represents a Telegram user
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// Client is a wrapper for the Telegram Bot API
type Client struct {
	Token      string
	APIBase    string
	HTTPClient *http.Client
}

// NewClient creates a new Telegram client
func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		APIBase:    "https://api.telegram.org",
		HTTPClient: &http.Client{},
	}
}

// GetUpdates fetches new updates from Telegram
func (c *Client) GetUpdates(offset int, timeout int) ([]Update, error) {
	u := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=%d", c.APIBase, c.Token, offset, timeout)

	resp, err := c.HTTPClient.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram API returned status: %s", resp.Status)
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram API reported error in response")
	}

	return result.Result, nil
}

// SendMessage sends a message to a specific chat
func (c *Client) SendMessage(chatID int64, text string) error {
	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", c.APIBase, c.Token)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := c.HTTPClient.Post(apiURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Log error body for debugging
		return fmt.Errorf("telegram API returned status: %v", resp.Status)
	}

	return nil
}
