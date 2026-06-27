package main

import (
	"embed"
	_ "embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/liup215/gline/internal/gui"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var iconBytes []byte

func runGUI() {
	// Initialize configuration and logging
	if err := InitConfig(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	if err := gui.InitBackend(); err != nil {
		log.Fatalf("Failed to initialise backend: %v", err)
	}

	chatService := &gui.ChatService{Backend: gui.BackendInstance}
	chatService.InitSlashRegistry()

	app := application.New(application.Options{
		Name:        "gline",
		Description: "AI Programming Assistant",
		Services: []application.Service{
			application.NewService(chatService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Icon: iconBytes,
	})

	chatService.SetApp(app)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "gline",
		Width:  1400,
		Height: 900,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Intercept close button (X) to hide to system tray instead of quitting
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	// --- System Tray Setup ---
	systemTray := app.SystemTray.New()
	systemTray.SetIcon(iconBytes)
	systemTray.AttachWindow(window)

	trayMenu := app.NewMenu()
	toggleItem := trayMenu.Add("Hide gline")
	toggleItem.OnClick(func(ctx *application.Context) {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show().Focus()
		}
	})

	// Dynamically update menu label when window is hidden/shown
	window.RegisterHook(events.Common.WindowHide, func(e *application.WindowEvent) {
		if toggleItem != nil {
			toggleItem.SetLabel("Show gline")
		}
	})
	window.RegisterHook(events.Common.WindowShow, func(e *application.WindowEvent) {
		if toggleItem != nil {
			toggleItem.SetLabel("Hide gline")
		}
	})

	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	systemTray.SetMenu(trayMenu)
	// --- End System Tray Setup ---

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
