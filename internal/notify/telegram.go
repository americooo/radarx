package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// Telegram sends alerts through the Telegram Bot API so the operator gets a
// push on their phone even when away from the machine — exactly what you want
// for continuous recon.
//
// Credentials are supplied by the caller, never hardcoded. Prefer environment
// variables:
//
//	RADARX_TG_TOKEN    bot token from @BotFather
//	RADARX_TG_CHAT_ID  your chat id (message the bot once, then look it up)
type Telegram struct {
	Token  string
	ChatID string
	client *http.Client
}

// NewTelegram builds a Telegram notifier. Returns (nil, false) if either
// credential is missing, so the caller can silently skip the channel.
func NewTelegram(token, chatID string) (*Telegram, bool) {
	if token == "" || chatID == "" {
		return nil, false
	}
	return &Telegram{
		Token:  token,
		ChatID: chatID,
		client: &http.Client{Timeout: 12 * time.Second},
	}, true
}

// NewTelegramFromEnv reads credentials from the environment.
func NewTelegramFromEnv() (*Telegram, bool) {
	return NewTelegram(os.Getenv("RADARX_TG_TOKEN"), os.Getenv("RADARX_TG_CHAT_ID"))
}

func (Telegram) Name() string { return "telegram" }

func (t *Telegram) Notify(d model.DiffResult) error {
	return t.send(FormatText(d))
}

// SendRaw posts an arbitrary message (used for startup pings / test messages).
func (t *Telegram) SendRaw(text string) error { return t.send(text) }

func (t *Telegram) send(text string) error {
	payload := map[string]any{
		"chat_id":                  t.ChatID,
		"text":                     text,
		"disable_web_page_preview": true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Description string `json:"description"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return fmt.Errorf("telegram API %d: %s", resp.StatusCode, apiErr.Description)
	}
	return nil
}
