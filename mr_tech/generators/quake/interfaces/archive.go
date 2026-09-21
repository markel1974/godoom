package interfaces

import "io"

// IArchive defines an interface for managing reader.go files, supporting file access and directory operations.
// Setup initializes the reader.go with the specified path.
// Open retrieves a file as a readable and seekable stream from the reader.go based on its full path.
// ReadDir returns a slice of file names present in the specified directory path within the reader.go.
// ReadDirFilter returns file names matching a wildcard pattern in the specified directory path within the reader.go.
type IArchive interface {
	Setup(path string) error

	Open(fullPath string) (io.ReadSeeker, error)

	ReadDir(fullPath string) ([]string, error)

	ReadDirFilter(fullPath string, wildcard string) ([]string, error)
}
