package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

// GrpMagic is the signature string used to validate the header of a GRP archive file.
const GrpMagic = "KenSilverman"

// GrpEntry represents an entry in a GRP archive, including its name, offset within the archive, and size in bytes.
type GrpEntry struct {
	Name   string
	Offset int64
	Size   int64
}

func NewGrpEntry(name string, offset, size int64) *GrpEntry {
	return &GrpEntry{
		Name:   name,
		Offset: offset,
		Size:   size,
	}
}

// ArchiveGrp represents a container for managing and accessing files within a GRP archive format.
type ArchiveGrp struct {
	reader  io.ReaderAt
	closer  io.Closer
	entries map[string]*GrpEntry
}

// NewArchiveGrp initializes and returns a new instance of ArchiveGrp with default values.
func NewArchiveGrp() *ArchiveGrp {
	return &ArchiveGrp{}
}

// Parse initializes the ArchiveGrp by reading and parsing the GRP file structure from the provided reader and closer.
func (a *ArchiveGrp) Parse(reader io.ReaderAt, closer io.Closer) error {
	var magic [12]byte
	if _, err := reader.ReadAt(magic[:], 0); err != nil {
		return fmt.Errorf("reading grp magic: %w", err)
	}
	if string(magic[:]) != GrpMagic {
		return fmt.Errorf("invalid grp signature %q (expected %q)", string(magic[:]), GrpMagic)
	}

	var numFiles int32
	var numFilesBuf [4]byte
	if _, err := reader.ReadAt(numFilesBuf[:], 12); err != nil {
		return fmt.Errorf("reading grp numfiles: %w", err)
	}
	numFiles = int32(binary.LittleEndian.Uint32(numFilesBuf[:]))
	if numFiles < 0 {
		return fmt.Errorf("invalid grp numfiles %d", numFiles)
	}

	a.reader = reader
	a.closer = closer
	a.entries = make(map[string]*GrpEntry, numFiles)

	// Read directory entries
	// Each entry is 12 bytes filename + 4 bytes filesize = 16 bytes
	dirSize := int64(numFiles) * 16
	dirBuf := make([]byte, dirSize)
	if _, err := reader.ReadAt(dirBuf, 16); err != nil {
		return fmt.Errorf("reading grp directory: %w", err)
	}

	curDataOffset := int64(16) + dirSize

	for i := 0; i < int(numFiles); i++ {
		entryOffset := i * 16
		rawName := dirBuf[entryOffset : entryOffset+12]
		fileSize := int64(binary.LittleEndian.Uint32(dirBuf[entryOffset+12 : entryOffset+16]))

		// Trim null characters from filename
		nullIdx := bytes.IndexByte(rawName, 0)
		var name string
		if nullIdx >= 0 {
			name = string(rawName[:nullIdx])
		} else {
			name = string(rawName)
		}
		nameUpper := strings.ToUpper(strings.TrimSpace(name))
		entry := NewGrpEntry(nameUpper, curDataOffset, fileSize)
		a.entries[nameUpper] = entry
		curDataOffset += fileSize
	}

	return nil
}

// Match searches the archive for file names matching the provided regex pattern and optionally sorts the results.
func (a *ArchiveGrp) Match(p string, useSort bool) ([]string, error) {
	r, err := regexp.Compile(p)
	if err != nil {
		return nil, err
	}
	var matches []string
	for name := range a.entries {
		if r.MatchString(name) {
			matches = append(matches, name)
		}
	}
	if useSort {
		sort.Strings(matches)
	}
	return matches, nil
}

// HasFile checks if a file with the specified name exists in the ArchiveGrp. The check is case-insensitive.
func (a *ArchiveGrp) HasFile(name string) bool {
	_, ok := a.entries[strings.ToUpper(name)]
	return ok
}

// GetPayload retrieves the raw data for the specified file name from the archive.
// It returns the file's contents as a byte slice or an error if the file is not found or cannot be read.
func (a *ArchiveGrp) GetPayload(name string) ([]byte, error) {
	entry, ok := a.entries[strings.ToUpper(name)]
	if !ok {
		return nil, fmt.Errorf("file %s not found in grp", name)
	}
	data := make([]byte, entry.Size)
	if _, err := a.reader.ReadAt(data, entry.Offset); err != nil && err != io.EOF {
		return nil, fmt.Errorf("reading %s: %w", name, err)
	}
	return data, nil
}

// GetReader retrieves a file from the archive by name and returns it as an io.ReadCloser or an error if not found.
func (a *ArchiveGrp) GetReader(name string) (io.ReadCloser, error) {
	data, err := a.GetPayload(name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Close releases resources associated with the ArchiveGrp by invoking the underlying io.Closer if present.
func (a *ArchiveGrp) Close() error {
	if a.closer != nil {
		return a.closer.Close()
	}
	return nil
}
