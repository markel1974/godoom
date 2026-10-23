package common

import "bytes"

// ReaderMem is a wrapper around bytes.Reader that implements the IReader interface for in-memory byte slice operations.
type ReaderMem struct {
	*bytes.Reader
}

// NewReaderMem creates a new ReaderMem wrapping the provided byte slice for in-memory reading operations.
func NewReaderMem(data []byte) ReaderMem {
	return ReaderMem{bytes.NewReader(data)}
}

// Close releases any resources associated with the ReaderMem instance and marks it as closed.
func (ReaderMem) Close() error { return nil }
