//go:build !js

package main

import (
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop"
)

// Resources provides methods for accessing and managing resource files and directories.
type Resources struct {
}

// NewResources creates and initializes a new instance of the Resources struct and returns a pointer to it.
func NewResources() *Resources {
	return &Resources{}
}

// Open opens the specified resource and returns an IReader interface for reading, seeking, closing, and offset reading.
func (r *Resources) Open(in string) (common.IReader, error) {
	return os.Open(in)
}

// ReadDir reads the contents of the specified directory and returns a list of directory entries or an error.
func (r *Resources) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}

// App represents the core application structure holding resources and providing methods for application control.
type App struct {
	res    *Resources
	width  int
	height int
	fps    int
}

// NewApp creates and returns a new instance of App with initialized resources.
func NewApp() *App {
	return &App{
		res: NewResources(),
	}
}

func (a *App) Setup(width, height, fps int) error {
	a.width, a.height, a.fps = width, height, fps
	return nil
}

// GetContext creates and returns a new graphics context with the specified width, height, and frames per second (fps).
func (a *App) GetContext() api.IContext {
	return desktop.NewContext(a.width, a.height, a.fps)
}

// WaitExit waits for the application to exit. On desktop platforms, no action is required as render.Start blocks execution.
func (a *App) WaitExit() {
	// Su desktop render.Start() blocca già l'esecuzione tramite glfw (w.th.Start),
	// quindi qui non dobbiamo fare nulla.
}

// UpdatePath modifies the provided path based on the given mode and returns the updated path as a string.
func (a *App) UpdatePath(path string) string {
	return path
}

// GetResources retrieves the Resources instance associated with the App.
func (a *App) GetResources() *Resources {
	return a.res
}
