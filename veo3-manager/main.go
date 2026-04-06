package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"veo3-manager/internal/config"
	"veo3-manager/internal/fileserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg := config.DefaultConfig()
	app := NewApp(cfg)

	err := wails.Run(&options.App{
		Title:            "Thanh Nhàn VEO 3",
		Width:            1280,
		Height:           800,
		MinWidth:         960,
		MinHeight:        600,
		Frameless:        true,
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: fileserver.NewHandler(cfg.DownloadDir),
		},
		BackgroundColour: &options.RGBA{R: 15, G: 15, B: 20, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		OnDomReady:       app.domReady,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
