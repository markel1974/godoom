package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

// GobHeader represents the header of a GOB file, containing magic bytes and master offset.
type GobHeader struct {
	Magic     [4]byte
	MasterOfs int32
}

// GobEntry represents an entry in a GOB file, containing its offset, size, and name.
type GobEntry struct {
	Offset int32
	Size   int32
	Name   [13]byte
}

// Gob represents a binary resource file mapping structure providing access to embedded entities.
// It wraps an io.ReaderAt for thread-safe reading at specified offsets using a GobEntity descriptor.
type Gob struct {
	file  io.ReaderAt
	entry *GobEntry
}

// NewGob initializes a new Gob structure using the provided io.ReaderAt and GobEntity parameters.
func NewGob(file io.ReaderAt, entry *GobEntry) *Gob {
	return &Gob{file: file, entry: entry}
}

// Read retrieves the data associated with the current Gob entry and returns it as a byte slice. An error is returned if the read fails.
func (g *Gob) Read() ([]byte, error) {
	data := make([]byte, g.entry.Size)
	// ReadAt esegue una lettura thread-safe all'offset specificato
	if _, err := g.file.ReadAt(data, int64(g.entry.Offset)); err != nil {
		// ReadAt può ritornare io.EOF se legge esattamente fino alla fine, gestiscilo se necessario
		if err != io.EOF {
			return nil, err
		}
	}
	return data, nil
}

// GobHandler manages a collection of Gob files and their entries, providing parsing, retrieval, and cleanup functionality.
type GobHandler struct {
	entries map[string]*Gob
	files   []*os.File
}

// NewGobHandler initializes and returns a new instance of GobHandler with an empty entry map.
func NewGobHandler() *GobHandler {
	return &GobHandler{entries: make(map[string]*Gob)}
}

// Parse reads the specified directory to identify and process .GOB files, adding them to the entries map in GobHandler.
// It skips non-GOB files and directories, returning any encountered error during the process.
func (g *GobHandler) Parse(dirPath string) error {
	entries, dErr := os.ReadDir(dirPath)
	if dErr != nil {
		return dErr
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		pName := strings.TrimSpace(strings.ToUpper(entry.Name()))
		if !strings.HasSuffix(pName, ".GOB") {
			continue
		}
		if err := g.add(dirPath + string(os.PathSeparator) + entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

// add reads a GOB file, extracts its entries, and registers them in the GobHandler's entries map.
func (g *GobHandler) add(filename string) error {
	f, fErr := os.Open(filename)
	if fErr != nil {
		return fErr
	}
	g.files = append(g.files, f)
	var header GobHeader
	if err := binary.Read(f, binary.LittleEndian, &header); err != nil {
		return err
	}
	if string(header.Magic[:]) != "GOB\n" {
		return fmt.Errorf("invalid magic")
	}
	if _, err := f.Seek(int64(header.MasterOfs), io.SeekStart); err != nil {
		return err
	}
	var numEntries int32
	if err := binary.Read(f, binary.LittleEndian, &numEntries); err != nil {
		return err
	}
	for i := int32(0); i < numEntries; i++ {
		entry := &GobEntry{}
		if err := binary.Read(f, binary.LittleEndian, entry); err != nil {
			return err
		}
		nameLen := bytes.IndexByte(entry.Name[:], 0)
		if nameLen == -1 {
			nameLen = len(entry.Name)
		}
		cleanName := strings.ToUpper(string(entry.Name[:nameLen]))
		g.entries[cleanName] = NewGob(f, entry)
	}
	return nil
}

// GetPayload retrieves the payload data associated with the specified name in uppercase from the GobHandler entries map.
func (g *GobHandler) GetPayload(name string) ([]byte, error) {
	gob, ok := g.entries[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("lump %s non trovato", name)
	}
	return gob.Read()
}

// Close releases all open file resources held by the GobHandler and clears the internal file slice.
func (g *GobHandler) Close() error {
	for _, f := range g.files {
		if f != nil {
			f.Close()
		}
	}
	g.files = nil
	return nil
}
