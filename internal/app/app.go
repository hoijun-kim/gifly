// Package app is the Wails binding layer: the App struct's exported methods are
// callable from the Svelte front end, and this is the only place the GUI reaches
// the engine (internal/gifjob, internal/probe, internal/ffmpeg).
package app

import "context"

// App is the Wails-bound application object.
type App struct {
	ctx context.Context
}

// NewApp constructs the App.
func NewApp() *App { return &App{} }

// Startup captures the Wails runtime context (bound in main.go's OnStartup). It
// is exported because Wails' OnStartup requires an accessible method value.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

// Greeting is a trivial bound method proving the JS<->Go bridge; Task 2 replaces
// it with the real bindings.
func (a *App) Greeting() string { return "gifly" }
