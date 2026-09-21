package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// BSPVersion represents the versioning enumeration for BSP (Binary Space Partitioning) structures in a system.
type BSPVersion int

// BSPVersionQ1 represents the BSP version used in Quake 1.
// BSPVersionQ2 represents the BSP version used in Quake 2.
const (
	BSPVersionQ1 BSPVersion = 29
	BSPVersionQ2 BSPVersion = 38
	BSPVersionQ3 BSPVersion = 46
)

type IArchive interface {
	Setup(path string) error

	Open(fullPath string) (io.ReadSeeker, error)

	ReadDir(fullPath string) ([]string, error)

	ReadDirFilter(fullPath string, wildcard string) ([]string, error)
}

type IReader interface {
	Open(path string) (io.ReadSeeker, error)
}

// IBSPReader defines an interface for reading BSP files, enabling access to entities and other BSP structures.
type IBSPReader interface {
	Setup(r IReader) error

	GetEntities() ([]*Entity, error)

	GetModels() ([]*Model, error)

	GetTextures() *Textures

	RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error

	RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error

	GetRawFaces(modelIdx int) ([]*RawFace, error)

	GetExternalBModelFileName(classname string) string

	GetModelFileName(classname string) string
}

// NewBSPReader detects the BSP file version from the provided io.ReadSeeker and returns an appropriate IBSPReader implementation.
func NewBSPReader(rs io.ReadSeeker, palette io.ReadSeeker) (IBSPReader, error) {
	var magic [4]byte
	if err := binary.Read(rs, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic bytes: %w", err)
	}

	// Quake 2 & 3 (IBSP)
	if string(magic[:]) == "IBSP" {
		var version int32
		if err := binary.Read(rs, binary.LittleEndian, &version); err != nil {
			return nil, err
		}
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to rewind stream: %w", err)
		}
		switch BSPVersion(version) {
		case BSPVersionQ3:
			return NewQ3BSPReader(rs), nil
		case BSPVersionQ2:
			return NewQ2BSPReader(rs, palette), nil
		default:
			return nil, fmt.Errorf("unsupported IBSP version: %d", version)
		}
	}

	// Quake 1 (version 29)
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to rewind stream: %w", err)
	}
	var version int32
	if err := binary.Read(rs, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version != int32(BSPVersionQ1) {
		return nil, fmt.Errorf("unsupported Q1 BSP version: %d", version)
	}
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to rewind stream: %w", err)
	}
	return NewQ1BSPReader(rs, palette), nil
}

func NewArchive(pakPath string) (IArchive, error) {
	if strings.HasSuffix(strings.ToLower(pakPath), ".pk3") {
		return NewPk3(), nil
	}
	return NewPak(), nil
}
