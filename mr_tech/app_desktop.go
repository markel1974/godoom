//go:build !js

package main

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop"
)

// App represents the core application structure holding resources and providing methods for application control.
type App struct {
	res    common.IResources
	width  int
	height int
	fps    int
	zFile  string
	path   string
}

// NewApp creates and returns a new instance of App with initialized resources.
func NewApp() *App {
	return &App{
		res: nil,
	}
}

// Setup initializes the application with specified width, height, and FPS, and configures its resources.
func (a *App) Setup(useZip bool, path string, width, height, fps int) error {
	if useZip {
		res := common.NewResourcesZip()
		res.SetPath(path)
		a.path = ""
		a.res = res
	} else {
		a.path = path
		a.res = common.NewResourcesOs()
	}
	if err := a.res.Setup(); err != nil {
		return err
	}
	a.width, a.height, a.fps = width, height, fps
	return nil
}

// GetContext creates and returns a new graphics context with the specified width, height, and frames per second (fps).
func (a *App) GetContext(player *config.Player) api.IContext {
	return desktop.NewContext(a.width, a.height, a.fps, player)
}

// WaitExit waits for the application to exit. On desktop platforms, no action is required as render.Start blocks execution.
func (a *App) WaitExit() {
	// Su desktop render.Start() blocca già l'esecuzione tramite glfw (w.th.Start),
	// quindi qui non dobbiamo fare nulla.
}

// GetPath retrieves the current file path associated with the application instance.
func (a *App) GetPath() string {
	return a.path
}

// GetResources retrieves the Resources instance associated with the App.
func (a *App) GetResources() common.IResources {
	return a.res
}
