package notify

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/americooo/radarx/internal/model"
)

// Desktop shows a native OS toast/notification by shelling out to the platform's
// built-in notifier. This keeps RadarX dependency-free: no CGO, no libnotify
// bindings — just the tools already present on each OS.
//
//	Linux:   notify-send
//	Windows: PowerShell toast (BurntToast-free, uses WinRT via PowerShell)
//	macOS:   osascript
type Desktop struct{}

func (Desktop) Name() string { return "desktop" }

func (Desktop) Notify(d model.DiffResult) error {
	if !d.HasSignal() {
		return nil
	}
	title := fmt.Sprintf("RadarX: %s", d.Root)
	body := fmt.Sprintf("%d new, %d changes", d.NewCount(), len(d.Changes))
	return sendDesktop(title, body)
}

func (Desktop) NotifyFinding(f model.Finding) error {
	title := fmt.Sprintf("RadarX: %s — %s", f.Module, f.Severity)
	body := f.Title
	if f.Asset.Key != "" {
		body = fmt.Sprintf("%s (%s)", body, f.Asset.Key)
	}
	return sendDesktop(title, body)
}

func sendDesktop(title, body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	switch runtime.GOOS {
	case "linux":
		return exec.CommandContext(ctx, "notify-send", "-a", "RadarX", title, body).Run()
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title %q`, body, title)
		return exec.CommandContext(ctx, "osascript", "-e", script).Run()
	case "windows":
		// Uses PowerShell + WinRT toast API, available on Windows 10/11 by default.
		ps := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$texts = $template.GetElementsByTagName("text")
$texts.Item(0).AppendChild($template.CreateTextNode(%q)) > $null
$texts.Item(1).AppendChild($template.CreateTextNode(%q)) > $null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("RadarX").Show($toast)
`, title, body)
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
	default:
		return fmt.Errorf("desktop notifications unsupported on %s", runtime.GOOS)
	}
}
