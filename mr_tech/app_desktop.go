//go:build !js

package main

import (
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/model"
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
	res *Resources
}

// NewApp creates and returns a new instance of App with initialized resources.
func NewApp() *App {
	return &App{
		res: NewResources(),
	}
}

// GetContext creates and returns a new graphics context with the specified width, height, and frames per second (fps).
func (a *App) GetContext(width, height, fps int) api.IContext {
	return desktop.NewContext(width, height, fps)
}

// WaitExit waits for the application to exit. On desktop platforms, no action is required as render.Start blocks execution.
func (a *App) WaitExit() {
	// Su desktop render.Start() blocca già l'esecuzione tramite glfw (w.th.Start),
	// quindi qui non dobbiamo fare nulla.
}

// UpdateMode updates the operating mode of the application and returns the updated mode.
func (a *App) UpdateMode(mode int) int {
	return mode
}

// UpdateFPS updates the application's frames-per-second (FPS) value and returns the updated FPS.
func (a *App) UpdateFPS(fps int) int {
	return fps
}

// UpdatePath modifies the provided path based on the given mode and returns the updated path as a string.
func (a *App) UpdatePath(mode int, path string) string {
	return path
}

// UpdateCalibration updates the Calibration object to enable shadow rendering and the use of shadow lights.
func (a *App) UpdateCalibration(cal *model.Calibration) {
	cal.EnableShadow = true
	cal.UseShadowLights = true
}

// GetResources retrieves the Resources instance associated with the App.
func (a *App) GetResources() *Resources {
	return a.res
}
