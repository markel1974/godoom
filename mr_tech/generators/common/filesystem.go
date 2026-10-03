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

// IFileSystem defines an interface for abstract file system operations, including file opening and directory reading.
type IFileSystem interface {
	Open(path string) (IReader, error)

	ReadDir(path string) ([]os.DirEntry, error)
}
