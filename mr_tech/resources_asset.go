package main

import (
	"embed"
	"io"
	"os"

	"github.com/markel1974/godoom/mr_tech/generators/common"
)

// assets is an embedded file system used to store and access static files or resources bundled with the application.
var assets embed.FS

// ResourcesAsset provides methods to interact with embedded asset files and directories.
type ResourcesAsset struct {
}

// NewResourcesAsset creates and returns a new instance of AssetReader, a utility for managing and accessing assets.
func NewResourcesAsset(s []byte) *ResourcesAsset {
	return &ResourcesAsset{}
}

// Setup initializes the AssetReader instance and prepares it for use, returning an error if the setup process fails.
func (r *ResourcesAsset) Setup() error {
	return nil
}

// ReadDir reads the directory named by the given path and returns a list of directory entries or an error if any occurs.
func (r *ResourcesAsset) ReadDir(path string) ([]os.DirEntry, error) {
	return assets.ReadDir(path)
}

// Open retrieves an asset by its path, reads its contents, and returns an IReader for further processing.
func (r *ResourcesAsset) Open(in string) (common.IReader, error) {
	f, err := assets.Open(in)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return NewReaderMem(data), nil
}

// Close releases any resources held by AssetReader and performs necessary cleanup operations.
func (r *ResourcesAsset) Close() error {
	return nil
}
