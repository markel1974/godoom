//go:build !js

package main

import (
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop"
)

// getContext creates and returns an OpenGL-like graphics context with the specified width and height.
func getContext(width, height, fps int) api.IContext {
	return desktop.NewContext(width, height, fps)
}

// waitExit blocks execution until the rendering process is explicitly terminated, requiring no additional actions here.
func waitExit() {
	// Su desktop render.Start() blocca già l'esecuzione tramite glfw (w.th.Start),
	// quindi qui non dobbiamo fare nulla.
}

func updateMode(mode int) int {
	return mode
}

func updatePath(mode int, path string) string {
	return path
}

// Resources represents a type that provides methods for accessing and interacting with external file-based resources.
type Resources struct {
}

// NewResources creates and returns a new instance of the Resources type.
func NewResources() *Resources {
	return &Resources{}
}

// Open opens the specified file and returns a ReadSeekCloser interface for reading its content or an error.
func (r *Resources) Open(in string) (common.IReader, error) {
	return os.Open(in)
}

// ReadDir reads the contents of the specified directory and returns a slice of directory entries or an error.
func (r *Resources) ReadDir(path string) ([]os.DirEntry, error) {
	return os.ReadDir(path)
}
