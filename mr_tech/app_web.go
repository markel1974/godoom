//go:build js && wasm

package main

import (
	"bytes"
	"embed"
	"io"
	"os"
	"path/filepath"
	"syscall/js"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/web"
)

// assets is an embedded file system used to store and access files bundled into the application at build time.
var assets embed.FS

// AssetReader is a wrapper around bytes.Reader that implements the io.Closer interface.
type AssetReader struct {
	*bytes.Reader
}

// NewAssetReader creates a new AssetReader instance initialized with the provided byte slice for reading operations.
func NewAssetReader(data []byte) *AssetReader {
	return &AssetReader{
		Reader: bytes.NewReader(data),
	}
}

// Close releases any resources associated with the AssetReader, implementing the io.Closer interface.
func (r *AssetReader) Close() error {
	return nil
}

// Resources provides functionality to manage and access game assets, supporting operations like file opening and directory reading.
type Resources struct {
}

// NewResources creates and returns a pointer to a new instance of the Resources struct.
func NewResources() *Resources {
	return &Resources{}
}

// Open retrieves a resource by name, using a JS buffer if available, or falling back to embedded assets otherwise.
func (r *Resources) Open(in string) (common.IReader, error) {
	// First check if JS provided a buffer
	jsBuf := js.Global().Get("window").Get("gameAssetBuffer")
	if !jsBuf.IsUndefined() && !jsBuf.IsNull() {
		length := jsBuf.Get("length").Int()
		data := make([]byte, length)
		js.CopyBytesToGo(data, jsBuf)

		// Free the JS reference so browser can GC it
		js.Global().Get("window").Set("gameAssetBuffer", js.Null())

		return NewAssetReader(data), nil
	}

	// Fallback to embedded assets if no JS buffer was provided
	f, err := assets.Open(in)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return NewAssetReader(data), nil
}

// ReadDir reads the contents of the specified directory path and returns a slice of directory entries or an error.
func (r *Resources) ReadDir(path string) ([]os.DirEntry, error) {
	return assets.ReadDir(path)
}

// App represents the main application structure, managing resources and providing core functionalities like updates and context creation.
type App struct {
	res *Resources
}

// NewApp initializes and returns a new instance of App with its resources set up.
func NewApp() *App {
	return &App{
		res: NewResources(),
	}
}

// UpdateFPS updates the frames-per-second (FPS) value and returns the updated FPS.
func (a *App) UpdateFPS(fps int) int {
	//TODO FROM SWITCH
	return 60
}

// UpdateCalibration updates the calibration by disabling shadow usage and shadow lights in the provided Calibration object.
func (a *App) UpdateCalibration(cal *model.Calibration) {
	cal.EnableShadow = false
	cal.UseShadowLights = true
}

// UpdateMode sets the application's operating mode to the provided value and returns the updated mode.
func (a *App) UpdateMode(mode int) int {
	return mode
}

// UpdatePath processes the file path and returns a new web-embedded asset path based on the input filename.
func (a *App) UpdatePath(mode int, path string) string {
	// Estraiamo solo il nome del file (es. "DOOM.WAD" o "pak0.pak")
	_, file := filepath.Split(path)
	// E lo mappiamo nella cartella embedded del web!
	return "html/asset/" + file
}

// GetContext initializes and returns an IContext instance for rendering using the specified width, height, and fps settings.
func (a *App) GetContext(width, height, fps int) api.IContext {
	return web.NewContext(width, height, fps)
}

// GetResources retrieves the Resources instance associated with the App.
func (a *App) GetResources() *Resources {
	return a.res
}

// WaitExit prevents premature program termination in Go by blocking execution indefinitely in Wasm environments.
func (a *App) WaitExit() {
	// In Wasm requestAnimationFrame non è bloccante,
	// dobbiamo fermare l'uscita prematura del programma Go.
	<-make(chan struct{})
}
