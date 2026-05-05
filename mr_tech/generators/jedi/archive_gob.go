package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	ExtLevel = ".LEV"
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

// ArchiveGob manages a collection of Gob files and their entries, providing parsing, retrieval, and cleanup functionality.
type ArchiveGob struct {
	entries map[string]*Gob
	files   []*os.File
	levels  []string
}

// NewArchiveGob initializes and returns a new instance of ArchiveGob with an empty entry map.
func NewArchiveGob() *ArchiveGob {
	return &ArchiveGob{entries: make(map[string]*Gob)}
}

// Parse reads the specified directory to identify and process .GOB files, adding them to the entries map in ArchiveGob.
// It skips non-GOB files and directories, returning any encountered error during the process.
func (g *ArchiveGob) Parse(dirPath string) error {
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

// add reads a GOB file, extracts its entries, and registers them in the ArchiveGob's entries map.
func (g *ArchiveGob) add(filename string) error {
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
		if pos := strings.Index(cleanName, ExtLevel); pos > 0 {
			g.levels = append(g.levels, cleanName[:pos])
		}
	}
	return nil
}

// GetLevels returns a slice of strings containing the names of all level entries identified in the GOB files.
func (g *ArchiveGob) GetLevels() []string {
	return g.levels
}

// GetPayload retrieves the payload data associated with the specified name in uppercase from the ArchiveGob entries map.
func (g *ArchiveGob) GetPayload(name string) ([]byte, error) {
	gob, ok := g.entries[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("%s not found", name)
	}
	return gob.Read()
}

// Close releases all open file resources held by the ArchiveGob and clears the internal file slice.
func (g *ArchiveGob) Close() error {
	for _, f := range g.files {
		if f != nil {
			f.Close()
		}
	}
	g.files = nil
	return nil
}
