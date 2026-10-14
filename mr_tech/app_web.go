//go:build js && wasm

package main

import (
	"bytes"
	"embed"
	"io"
	"os"
	"path/filepath"
	"syscall/js"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/web"
)

// assets is an embedded file system containing application resources and assets for runtime access.
var assets embed.FS

// AssetReader is a wrapper around bytes.Reader that includes a no-op Close method for compatibility with io.ReadCloser.
type AssetReader struct {
	*bytes.Reader
}

// NewAssetReader creates and returns a new AssetReader initialized with the provided byte slice.
func NewAssetReader(data []byte) *AssetReader {
	return &AssetReader{
		Reader: bytes.NewReader(data),
	}
}

// Close releases resources associated with the AssetReader, ensuring proper cleanup. It satisfies the io.Closer interface.
func (r *AssetReader) Close() error {
	return nil
}

// Resources is a type that provides functionality to manage and access embedded or runtime-loaded assets in the system.
type Resources struct {
}

// NewResources creates and returns a new instance of the Resources struct.
func NewResources() *Resources {
	return &Resources{}
}

// Open retrieves a resource as an IReader, prioritizing JS-provided buffers, with a fallback to embedded assets.
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

// ReadDir reads the contents of the directory specified by the given path and returns a list of directory entries.
func (r *Resources) ReadDir(path string) ([]os.DirEntry, error) {
	return assets.ReadDir(path)
}

// App represents the main application structure containing resources, display dimensions, and frame rate configuration.
type App struct {
	res    *Resources
	width  int
	height int
	fps    int
}

// NewApp creates and initializes a new App instance with its resources.
func NewApp() *App {
	return &App{
		res: NewResources(),
	}
}

// Setup initializes the App with the specified width, height, and frames per second (fps). Returns an error if initialization fails.
func (a *App) Setup(width, height, fps int) error {
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
func (a *App) GetResources() *Resources {
	return a.res
}

// WaitExit prevents the Go program from exiting prematurely in Wasm by blocking indefinitely using a channel.
func (a *App) WaitExit() {
	// In Wasm requestAnimationFrame non è bloccante,
	// dobbiamo fermare l'uscita prematura del programma Go.
	<-make(chan struct{})
}
