package main

import (
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

// ResourcesOs provides methods for accessing and managing resource files and directories.
type ResourcesOs struct {
}

// NewResourcesOs creates and initializes a new instance of the Resources struct and returns a pointer to it.
func NewResourcesOs() *ResourcesOs {
	return &ResourcesOs{}
}

// Setup initializes the ResourcesOs instance and prepares it for use, returning an error if the initialization fails.
func (r *ResourcesOs) Setup() error {
	return nil
}

// Open opens the specified resource and returns an IReader interface for reading, seeking, closing, and offset reading.
func (r *ResourcesOs) Open(in string) (common.IReader, error) {
	return os.Open(in)

}

// ReadDir reads the contents of the specified directory and returns a list of directory entries or an error.
func (r *ResourcesOs) ReadDir(fPath string) ([]os.DirEntry, error) {
	return os.ReadDir(fPath)
}
