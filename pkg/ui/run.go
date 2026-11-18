package ui

import (
	"io/fs"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

// Run starts the Wails desktop application without embedded assets (used with dev server).
func Run() {
	RunWithAssets(nil)
}

// RunWithAssets starts the Wails desktop application with the provided asset filesystem.
func RunWithAssets(assets fs.FS) {
	app := NewApp()

	appOptions := &options.App{
		Title:     "File Transfer Utility",
		Width:     980,
		Height:    680,
		MinWidth:  860,
		MinHeight: 580,
		BackgroundColour: &options.RGBA{
			R: 15,
			G: 23,
			B: 42,
			A: 255,
		},
		OnStartup:  app.Startup,
		OnShutdown: app.Shutdown,
		Bind: []interface{}{
			app,
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		Linux: &linux.Options{
			WindowIsTranslucent: false,
		},
	}

	if assets != nil {
		appOptions.AssetServer = &assetserver.Options{
			Assets: assets,
		}
	}

	err := wails.Run(appOptions)
	if err != nil {
		log.Fatalf("Error running Wails application: %v", err)
	}
}
