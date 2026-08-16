// Package notify delivers diff results to the operator through one or more
// channels. Notifier is the seam: the scheduler depends only on the interface,
// so console, desktop, Telegram, or any future channel plug in without changes.
//
// Secrets (like a Telegram bot token) are NEVER hardcoded — they come from the
// environment or a config file passed in by the caller.
package notify

import (
	"fmt"
	"os"
	"strings"

	"github.com/americooo/radarx/internal/model"
	"github.com/americooo/radarx/internal/store"
)

// Default assembles the standard set of notification channels: console and
// desktop are always on, Telegram is added only if a token+chatID are given
// (resolve them from env vars or the SQLite settings table before calling
// this — see ResolveCredentials). Both the CLI and the GUI use this so their
// notification behaviour never drifts apart.
func Default(telegramToken, telegramChatID string) Notifier {
	channels := []Notifier{Console{}, Desktop{}}
	if tg, ok := NewTelegram(telegramToken, telegramChatID); ok {
		channels = append(channels, tg)
	}
	return Multi{Notifiers: channels}
}

// ResolveCredentials resolves the Telegram token/chat id to use: environment
// variables (RADARX_TG_TOKEN / RADARX_TG_CHAT_ID) take priority — useful for
// CI, scripting, or power users — falling back to the SQLite settings table
// (telegram_token / telegram_chat_id, saved from the GUI's Settings screen).
// ok is false if either half is missing from both sources.
func ResolveCredentials(st *store.SQLiteStore) (token, chatID string, ok bool) {
	token, chatID = os.Getenv("RADARX_TG_TOKEN"), os.Getenv("RADARX_TG_CHAT_ID")
	if token != "" && chatID != "" {
		return token, chatID, true
	}
	if st == nil {
		return token, chatID, false
	}
	if token == "" {
		if v, has, err := st.GetSetting("telegram_token"); err == nil && has {
			token = v
		}
	}
	if chatID == "" {
		if v, has, err := st.GetSetting("telegram_chat_id"); err == nil && has {
			chatID = v
		}
	}
	return token, chatID, token != "" && chatID != ""
}

// Notifier delivers a diff result somewhere the operator will see it.
type Notifier interface {
	// Notify sends the diff. Implementations should only surface results that
	// carry signal (callers typically skip empty diffs before calling).
	Notify(d model.DiffResult) error
	// Name identifies the channel for logging.
	Name() string
}

// FormatText renders a diff into a compact, channel-agnostic message. Used by
// console and Telegram notifiers so the wording stays consistent.
func FormatText(d model.DiffResult) string {
	var b strings.Builder
	label := d.Root
	fmt.Fprintf(&b, "🛰 RadarX — %s\n", label)
	fmt.Fprintf(&b, "%d new, %d total changes\n", d.NewCount(), len(d.Changes))

	shown := 0
	for _, c := range d.Changes {
		if shown >= 15 { // don't flood a mobile notification
			fmt.Fprintf(&b, "…and %d more\n", len(d.Changes)-shown)
			break
		}
		switch c.Type {
		case model.ChangeNew:
			fmt.Fprintf(&b, "🟢 NEW  %s  %s\n", c.Kind, describe(c.After))
		case model.ChangeModified:
			fmt.Fprintf(&b, "🟡 CHG  %s  %s (%s)\n", c.Kind, c.Key, strings.Join(c.Fields, ","))
		case model.ChangeRemoved:
			fmt.Fprintf(&b, "🔴 DEL  %s  %s\n", c.Kind, c.Key)
		}
		shown++
	}
	return b.String()
}

func describe(a *model.Asset) string {
	if a == nil {
		return ""
	}
	switch a.Kind {
	case model.KindEndpoint:
		s := a.Key
		if a.StatusCode != 0 {
			s += fmt.Sprintf(" (%d)", a.StatusCode)
		}
		if a.Title != "" {
			s += " — " + a.Title
		}
		return s
	case model.KindCert:
		return fmt.Sprintf("%s CN=%s", a.Host, a.CertCN)
	default:
		return a.Key
	}
}

// Multi fans out a diff to several notifiers, collecting any errors so one
// failing channel (e.g. no network for Telegram) never blocks the others.
type Multi struct {
	Notifiers []Notifier
}

func (m Multi) Name() string { return "multi" }

func (m Multi) Notify(d model.DiffResult) error {
	var errs []string
	for _, n := range m.Notifiers {
		if err := n.Notify(d); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", n.Name(), err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("notify errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// Console prints the diff to stdout. Always available, no config needed.
type Console struct{}

func (Console) Name() string { return "console" }

func (Console) Notify(d model.DiffResult) error {
	fmt.Print(FormatText(d))
	return nil
}
