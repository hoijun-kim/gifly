package main

import (
	"embed"
	"log"

	"github.com/hoijun-kim/gifly/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := app.NewApp()
	if err := wails.Run(&options.App{
		Title:       "gifly",
		Width:       720,
		Height:      620,
		MinWidth:    520,
		MinHeight:   480,
		Frameless:   true,
		AssetServer: &assetserver.Options{Assets: assets},
		OnStartup:   a.Startup,
		Bind:        []interface{}{a},
	}); err != nil {
		log.Fatal(err)
	}
}
