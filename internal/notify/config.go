package notify

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config persists Telegram credentials the operator enters through the GUI,
// so they don't have to set environment variables to use continuous
// monitoring alerts. It lives at ~/.radarx/config.json with 0600 permissions
// (readable only by the owner) — this file holds a bot token, never commit
// or transmit it anywhere else.
type Config struct {
	TelegramToken  string `json:"telegram_token,omitempty"`
	TelegramChatID string `json:"telegram_chat_id,omitempty"`
}

// ConfigPath returns ~/.radarx/config.json. Exported so the CLI and GUI both
// point at the same file without duplicating the join logic.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".radarx", "config.json"), nil
}

// LoadConfig reads the local config file. A missing file is not an error —
// it returns a zero Config, matching the "nothing configured yet" case.
func LoadConfig() (Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, err
	}
	return c, nil
}

// SaveConfig writes c to disk, creating ~/.radarx if needed. The file is
// created 0600 (owner read/write only) since it may contain a bot token.
func SaveConfig(c Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Credentials resolves the Telegram token/chat id to use: environment
// variables (RADARX_TG_TOKEN / RADARX_TG_CHAT_ID) take priority — useful for
// CI, scripting, or power users — falling back to the local config file
// saved from the GUI's Settings screen. ok is false if either half is
// missing from both sources.
func Credentials() (token, chatID string, ok bool) {
	token, chatID = os.Getenv("RADARX_TG_TOKEN"), os.Getenv("RADARX_TG_CHAT_ID")
	if token != "" && chatID != "" {
		return token, chatID, true
	}
	cfg, err := LoadConfig()
	if err != nil {
		return "", "", false
	}
	if token == "" {
		token = cfg.TelegramToken
	}
	if chatID == "" {
		chatID = cfg.TelegramChatID
	}
	return token, chatID, token != "" && chatID != ""
}
