package common

import (
	"io"
	"os"
)

// IReader defines a composite interface for reading, seeking, closing, and reading at specific offsets in a resource.
type IReader interface {
	io.Reader
	io.Seeker
	io.Closer
	io.ReaderAt
}

// IResources defines an interface for abstract file system operations, including file opening and directory reading.
type IResources interface {
	Setup() error

	Open(path string) (IReader, error)

	ReadDir(path string) ([]os.DirEntry, error)
}
