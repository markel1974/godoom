//go:build js && wasm

package main

import (
	"path/filepath"
	"syscall/js"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/web"
)

// App represents the main application structure containing resources, display dimensions, and frame rate configuration.
type App struct {
	res    common.IFileSystem
	width  int
	height int
	fps    int
}

// NewApp creates and initializes a new App instance with its resources.
func NewApp() *App {
	return &App{
		res: nil,
	}
}

// Setup initializes the App with the specified width, height, and frames per second (fps). Returns an error if initialization fails.
func (a *App) Setup(width, height, fps int) error {
	var data []byte
	jsBuf := js.Global().Get("window").Get("gameAssetBuffer")
	if !jsBuf.IsUndefined() && !jsBuf.IsNull() {
		length := jsBuf.Get("length").Int()
		data = make([]byte, length)
		js.CopyBytesToGo(data, jsBuf)
		// Free the JS reference so browser can GC it
		js.Global().Get("window").Set("gameAssetBuffer", js.Null())
		a.res = NewResourcesMem(data)
	} else {
		a.res = NewResourcesAsset(data)
	}
	if err := a.res.Setup(); err != nil {
		return err
	}
	a.width = width
	a.height = height
	a.fps = fps
	return nil
}

// UpdatePath converts the given path to a standardized asset path by keeping only the file name and prefixing it with "html/asset/".
func (a *App) UpdatePath(path string) string {
	_, file := filepath.Split(path)
	return "html/asset/" + file
}

// GetContext returns a new instance of api.IContext configured with the app's width, height, and frames per second (fps).
func (a *App) GetContext(player *config.Player) api.IContext {
	return web.NewContext(a.width, a.height, a.fps, player)
}

// GetResources returns the Resources instance associated with the App.
func (a *App) GetResources() common.IFileSystem {
	return a.res
}

// WaitExit prevents the Go program from exiting prematurely in Wasm by blocking indefinitely using a channel.
func (a *App) WaitExit() {
	// In Wasm requestAnimationFrame non è bloccante,
	// dobbiamo fermare l'uscita prematura del programma Go.
	<-make(chan struct{})
}
