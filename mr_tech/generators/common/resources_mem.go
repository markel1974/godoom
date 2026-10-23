package common

import (
	"os"
)

// ResourcesMem is a structure for managing in-memory resources, designed to simulate file-like operations.
type ResourcesMem struct {
	s []byte
}

// NewResourcesMem creates a new ResourcesAsset instance initialized with the provided byte slice.
func NewResourcesMem(s []byte) *ResourcesMem {
	return &ResourcesMem{
		s: s,
	}
}

// Setup initializes the ResourcesMem instance and prepares it for use.
func (r *ResourcesMem) Setup() error {
	return nil
}

// ReadDir reads the directory named by the given path and returns a list of directory entries or an error.
func (r *ResourcesMem) ReadDir(path string) ([]os.DirEntry, error) {
	return nil, nil
}

// Open opens an in-memory resource using the provided string and returns an IReader interface or an error if it fails.
func (r *ResourcesMem) Open(in string) (IReader, error) {
	return NewReaderMem(r.s), nil
}

// Close releases any resources associated with the ResourcesMem instance. Returns an error if the operation fails.
func (r *ResourcesMem) Close() error {
	return nil
}
