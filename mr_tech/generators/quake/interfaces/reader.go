package interfaces

import "io"

// IReader represents an interface for reading and seeking through a resource identified by a file path.
type IReader interface {
	Open(path string) (io.ReadSeeker, error)
}
