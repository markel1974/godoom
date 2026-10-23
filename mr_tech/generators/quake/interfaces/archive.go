package interfaces

import "io"

// IArchive defines an interface for managing resources_os.go files, supporting file access and directory operations.
// Setup initializes the resources_os.go with the specified path.
// Open retrieves a file as a readable and seekable stream from the resources_os.go based on its full path.
// ReadDir returns a slice of file names present in the specified directory path within the resources_os.go.
// ReadDirFilter returns file names matching a wildcard pattern in the specified directory path within the resources_os.go.
type IArchive interface {
	Setup(path string) error

	Open(fullPath string) (io.ReadSeeker, error)

	ReadDir(fullPath string) ([]string, error)

	ReadDirFilter(fullPath string, wildcard string) ([]string, error)
}
