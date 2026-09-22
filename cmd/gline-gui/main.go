//go:build gui

// gline-gui: standalone desktop GUI entry for gline.
//
// Build:   go build -tags gui -ldflags "-s -w -H=windowsgui" -o gline-gui.exe ./cmd/gline-gui/
// Install: gline-gui.exe → C:\Users\<you>\bin\gline-gui.exe
//
// Launching this binary opens the GUI directly — no terminal window, no
// Cobra, no TUI code compiled in.  System-tray behaviour is identical to
// "gline --gui".
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/liup215/gline/internal/config"
	"github.com/liup215/gline/internal/gui"
	glog "github.com/liup215/gline/internal/log"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var iconBytes []byte

func main() {
	initConfig()
	runGUI()
}

func initConfig() {
	cm := config.NewManager()
	if err := cm.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	cfg := cm.Get()
	if err := glog.Init(glog.Config{
		Level:   cfg.Log.Level,
		File:    cfg.Log.File,
		Console: false, // GUI binary — never write to stdout/stderr
		Color:   true,
	}); err != nil {
		glog.Fatalf("Failed to initialise logger: %v", err)
	}
}

func runGUI() {
	if err := gui.InitBackend(); err != nil {
		glog.Fatalf("Failed to initialise backend: %v", err)
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

	// --- System tray ---
	// NOTE: deliberately NOT calling systemTray.AttachWindow(window).
	// Wails v3.0.0-alpha.63 force-registers a WindowLostFocus → Hide()
	// listener on attached windows (popover-style behaviour for tray
	// popups), which makes the main window vanish whenever it loses
	// focus (e.g. Alt-Tab, clicking another app). alpha2.x replaced
	// this with an opt-in HideOnFocusLost option. Until we upgrade,
	// wire the tray click toggle manually instead — same UX, no
	// auto-hide on blur.
	systemTray := app.SystemTray.New()
	systemTray.SetIcon(iconBytes)
	systemTray.OnClick(func() {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show().Focus()
		}
	})

	trayMenu := app.NewMenu()
	toggleItem := trayMenu.Add("Hide gline")
	toggleItem.OnClick(func(ctx *application.Context) {
		if window.IsVisible() {
			window.Hide()
		} else {
			window.Show().Focus()
		}
	})
	window.RegisterHook(events.Common.WindowHide, func(e *application.WindowEvent) {
		toggleItem.SetLabel("Show gline")
	})
	window.RegisterHook(events.Common.WindowShow, func(e *application.WindowEvent) {
		toggleItem.SetLabel("Hide gline")
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})
	systemTray.SetMenu(trayMenu)
	// --- End system tray ---

	if err := app.Run(); err != nil {
		glog.Fatal(err.Error())
	}
}
