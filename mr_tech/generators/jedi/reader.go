package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// GobHeader represents the header structure of a GOB file, containing metadata about the file's layout and content.
// Magic is a 4-byte identifier used to verify the file format.
// MasterOfs specifies the offset where the master directory begins.
type GobHeader struct {
	Magic     [4]byte
	MasterOfs int32
}

// GobEntry represents an entry in a GOB file, containing its offset, size, and a 13-byte name.
type GobEntry struct {
	Offset int32
	Size   int32
	Name   [13]byte
}

// GOB represents a container for managing binary data with indexed entries within a file.
type GOB struct {
	file    *os.File
	entries map[string]GobEntry
}

// NewGOB opens and parses a GOB file, returning a GOB structure for interaction or an error if the process fails.
func NewGOB(filename string) (*GOB, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	var header GobHeader
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if string(header.Magic[:]) != "GOB\n" {
		return nil, fmt.Errorf("magic non valido")
	}

	if _, err := f.Seek(int64(header.MasterOfs), io.SeekStart); err != nil {
		return nil, err
	}

	var numEntries int32
	if err := binary.Read(f, binary.LittleEndian, &numEntries); err != nil {
		return nil, err
	}

	entries := make(map[string]GobEntry)
	for i := int32(0); i < numEntries; i++ {
		var entry GobEntry
		if err := binary.Read(f, binary.LittleEndian, &entry); err != nil {
			return nil, err
		}

		nameLen := bytes.IndexByte(entry.Name[:], 0)
		if nameLen == -1 {
			nameLen = len(entry.Name)
		}
		cleanName := strings.ToUpper(string(entry.Name[:nameLen]))
		entries[cleanName] = entry
	}

	return &GOB{file: f, entries: entries}, nil
}

// GetPayload retrieves the payload data for a given name from the GOB entries, returning it as a byte slice.
func (g *GOB) GetPayload(name string) ([]byte, error) {
	entry, ok := g.entries[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("lump %s non trovato", name)
	}
	data := make([]byte, entry.Size)
	if _, err := g.file.Seek(int64(entry.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(g.file, data); err != nil {
		return nil, err
	}
	return data, nil
}

// Close releases the underlying file resource associated with the GOB instance and returns any error encountered.
func (g *GOB) Close() error {
	if g.file != nil {
		return g.file.Close()
	}
	return nil
}
