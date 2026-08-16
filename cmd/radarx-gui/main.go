package main

import (
	"context"
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "radarx-gui",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown: func(ctx context.Context) {
			// Best-effort: stop any background monitoring loop so it doesn't
			// keep scanning after the app has quit. The process is about to
			// exit anyway, so an error here is safe to ignore.
			_ = app.StopMonitoring()
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			// Wails v2.14.0 has no stable systray API (see
			// pkg/buildassets/onhold/tray — explicitly "on hold", and pulling
			// in a third-party systray library would add another CGo
			// dependency to a build pipeline we just finished stabilizing
			// across Windows/macOS/Linux). Instead: if monitoring is active,
			// treat the window's close button as "hide to background" rather
			// than "quit", so continuous monitoring keeps running. The user
			// exits by stopping monitoring first (Settings) or killing the
			// process/window from the OS.
			if app.IsMonitoring() {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
