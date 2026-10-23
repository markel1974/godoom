package lumps

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
)

const GrpMagic = "KenSilverman"

type GrpEntry struct {
	Name   string
	Offset int64
	Size   int64
}

type ArchiveGrp struct {
	reader  io.ReaderAt
	closer  io.Closer
	entries map[string]*GrpEntry
	maps    []string
}

func OpenGrp(reader io.ReaderAt, closer io.Closer) (*ArchiveGrp, error) {
	var magic [12]byte
	if _, err := reader.ReadAt(magic[:], 0); err != nil {
		return nil, fmt.Errorf("reading grp magic: %w", err)
	}
	if string(magic[:]) != GrpMagic {
		return nil, fmt.Errorf("invalid grp signature %q (expected %q)", string(magic[:]), GrpMagic)
	}

	var numFiles int32
	var numFilesBuf [4]byte
	if _, err := reader.ReadAt(numFilesBuf[:], 12); err != nil {
		return nil, fmt.Errorf("reading grp numfiles: %w", err)
	}
	numFiles = int32(binary.LittleEndian.Uint32(numFilesBuf[:]))
	if numFiles < 0 {
		return nil, fmt.Errorf("invalid grp numfiles %d", numFiles)
	}

	arc := &ArchiveGrp{
		reader:  reader,
		closer:  closer,
		entries: make(map[string]*GrpEntry, numFiles),
	}

	// Read directory entries
	// Each entry is 12 bytes filename + 4 bytes filesize = 16 bytes
	dirSize := int64(numFiles) * 16
	dirBuf := make([]byte, dirSize)
	if _, err := reader.ReadAt(dirBuf, 16); err != nil {
		return nil, fmt.Errorf("reading grp directory: %w", err)
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
		name = strings.TrimSpace(name)
		nameUpper := strings.ToUpper(name)

		entry := &GrpEntry{
			Name:   nameUpper,
			Offset: curDataOffset,
			Size:   fileSize,
		}
		arc.entries[nameUpper] = entry
		if strings.HasSuffix(nameUpper, ".MAP") {
			arc.maps = append(arc.maps, nameUpper)
		}

		curDataOffset += fileSize
	}

	sort.Strings(arc.maps)
	return arc, nil
}

func (a *ArchiveGrp) GetMaps() []string {
	return a.maps
}

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

func (a *ArchiveGrp) GetReader(name string) (io.ReadCloser, error) {
	data, err := a.GetPayload(name)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (a *ArchiveGrp) Close() error {
	if a.closer != nil {
		return a.closer.Close()
	}
	return nil
}
