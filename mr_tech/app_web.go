//go:build js && wasm

package main

import (
	"bytes"
	"embed"
	"io"
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/web"
)

// assets is an embedded filesystem containing files from the "assets" directory.
//
//go:embed html/asset/*
var assets embed.FS

// getContext creates and returns a new graphics context with the specified width and height.
func getContext(width, height int) api.IContext {
	return web.NewContext(width, height)
}

// waitExit prevents the premature exit of a Go program in a Wasm environment where requestAnimationFrame is non-blocking.
func waitExit() {
	// In Wasm requestAnimationFrame non è bloccante,
	// dobbiamo fermare l'uscita prematura del programma Go.
	<-make(chan struct{})
}

// AssetReader is a wrapper around bytes.Reader that includes an additional Close method to satisfy certain interfaces.
type AssetReader struct {
	*bytes.Reader
}

// NewAssetReader creates a new AssetReader instance from a byte slice.
func NewAssetReader(data []byte) *AssetReader {
	return &AssetReader{
		Reader: bytes.NewReader(data),
	}
}

// Close releases any resources associated with the AssetReader. Returns an error if the operation fails.
func (r *AssetReader) Close() error {
	return nil
}

// Resources is a struct that provides methods for accessing and interacting with external file-based resources.
type Resources struct {
}

// NewResources creates and returns a new instance of the Resources type for managing external file-based resources.
func NewResources() *Resources {
	return &Resources{}
}

// Open retrieves the specified resource as a reader interface or returns an error if the resource is unavailable.
func (r *Resources) Open(in string) (common.IReader, error) {
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

// ReadDir reads the contents of the specified directory and returns a slice of directory entries or an error.
func (r *Resources) ReadDir(path string) ([]os.DirEntry, error) {
	return assets.ReadDir(path)
}
