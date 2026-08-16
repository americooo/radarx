package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg != (Config{}) {
		t.Fatalf("expected zero Config, got %+v", cfg)
	}
}

func TestSaveConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	want := Config{TelegramToken: "tok123", TelegramChatID: "42"}
	if err := SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	got, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if got != want {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", got, want)
	}

	path := filepath.Join(dir, ".radarx", "config.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected config file to be 0600 (owner-only, it may hold a bot token), got %o", perm)
	}
}

func TestCredentialsPrefersEnvOverConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SaveConfig(Config{TelegramToken: "file-token", TelegramChatID: "file-chat"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RADARX_TG_TOKEN", "env-token")
	t.Setenv("RADARX_TG_CHAT_ID", "env-chat")

	token, chatID, ok := Credentials()
	if !ok {
		t.Fatal("expected credentials to be found")
	}
	if token != "env-token" || chatID != "env-chat" {
		t.Fatalf("expected env vars to win, got token=%q chatID=%q", token, chatID)
	}
}

func TestCredentialsFallsBackToConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("RADARX_TG_TOKEN", "")
	t.Setenv("RADARX_TG_CHAT_ID", "")

	if err := SaveConfig(Config{TelegramToken: "file-token", TelegramChatID: "file-chat"}); err != nil {
		t.Fatal(err)
	}

	token, chatID, ok := Credentials()
	if !ok {
		t.Fatal("expected credentials to be found from the config file")
	}
	if token != "file-token" || chatID != "file-chat" {
		t.Fatalf("got token=%q chatID=%q", token, chatID)
	}
}

func TestCredentialsMissingEverywhere(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("RADARX_TG_TOKEN", "")
	t.Setenv("RADARX_TG_CHAT_ID", "")

	if _, _, ok := Credentials(); ok {
		t.Fatal("expected no credentials when nothing is configured")
	}
}
