package jedi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// LfdEntry represents a single entry in an LFD file, containing metadata about the entry's type, name, size, and offset.
type LfdEntry struct {
	Type   string
	Name   string
	Size   uint32
	Offset int64
}

// LFD represents a container for managing LFD file data, including a reader and a map of its entries.
type LFD struct {
	reader  io.ReadSeeker // Cambiato da *os.File a io.ReadSeeker
	entries map[string]LfdEntry
}

// NewLFD creates and returns a new instance of LFD with initialized fields.
func NewLFD() *LFD {
	return &LFD{}
}

// Parse reads and parses the LFD file structure from the provided io.ReadSeeker and populates the entries map.
func (l *LFD) Parse(r io.ReadSeeker) error {
	l.reader = r
	l.entries = make(map[string]LfdEntry)

	var rMapType [4]byte
	var rMapName [8]byte
	var rMapSize uint32

	if err := binary.Read(r, binary.LittleEndian, &rMapType); err != nil {
		return err
	}

	if string(rMapType[:]) != "RMAP" {
		return fmt.Errorf("invalid LFD signature, found: %s", string(rMapType[:]))
	}

	_ = binary.Read(r, binary.LittleEndian, &rMapName)
	_ = binary.Read(r, binary.LittleEndian, &rMapSize)

	numEntries := rMapSize / 16
	currentOffset := int64(16) + int64(rMapSize)

	for i := uint32(0); i < numEntries; i++ {
		var eType [4]byte
		var eName [8]byte
		var eSize uint32

		_ = binary.Read(r, binary.LittleEndian, &eType)
		_ = binary.Read(r, binary.LittleEndian, &eName)
		_ = binary.Read(r, binary.LittleEndian, &eSize)

		cleanType := strings.TrimSpace(string(eType[:]))
		nameLen := bytes.IndexByte(eName[:], 0)
		if nameLen == -1 {
			nameLen = 8
		}
		cleanName := strings.TrimSpace(string(eName[:nameLen]))

		entry := LfdEntry{
			Type:   cleanType,
			Name:   cleanName,
			Size:   eSize,
			Offset: currentOffset + 16,
		}

		key := fmt.Sprintf("%s_%s", cleanType, cleanName)
		l.entries[key] = entry

		currentOffset += 16 + int64(eSize)
	}

	return nil
}

// GetPayload retrieves a payload by its resource type and name, returning the data as a byte slice or an error if not found.
func (l *LFD) GetPayload(resType, name string) ([]byte, error) {
	key := fmt.Sprintf("%s_%s", resType, name)
	entry, ok := l.entries[key]
	if !ok {
		return nil, fmt.Errorf("resource %s not found", key)
	}

	data := make([]byte, entry.Size)
	if _, err := l.reader.Seek(entry.Offset, io.SeekStart); err != nil {
		return nil, err
	}
	if _, err := io.ReadFull(l.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}
