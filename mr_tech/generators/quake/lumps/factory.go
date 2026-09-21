package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// BSPVersion represents the version of a BSP (Binary Space Partitioning) file used in different game engines.
type BSPVersion int

// BSPVersionQ1 represents the BSP version for Quake 1 files.
// BSPVersionQ2 represents the BSP version for Quake 2 files.
// BSPVersionQ3 represents the BSP version for Quake 3 files.
const (
	BSPVersionQ1 BSPVersion = 29
	BSPVersionQ2 BSPVersion = 38
	BSPVersionQ3 BSPVersion = 46
)

// IArchive defines an interface for managing archive files, supporting file access and directory operations.
// Setup initializes the archive with the specified path.
// Open retrieves a file as a readable and seekable stream from the archive based on its full path.
// ReadDir returns a slice of file names present in the specified directory path within the archive.
// ReadDirFilter returns file names matching a wildcard pattern in the specified directory path within the archive.
type IArchive interface {
	Setup(path string) error

	Open(fullPath string) (io.ReadSeeker, error)

	ReadDir(fullPath string) ([]string, error)

	ReadDirFilter(fullPath string, wildcard string) ([]string, error)
}

// IReader represents an interface for reading and seeking through a resource identified by a file path.
type IReader interface {
	Open(path string) (io.ReadSeeker, error)
}

// IBSPReader defines an interface for reading and processing BSP (Binary Space Partitioning) structure data.
type IBSPReader interface {
	Setup() error

	GetArchive() IArchive

	Build(root *config.Root) error

	GetPlayerInfo() (float64, geometry.XYZ)

	GetEntities() ([]*Entity, error)

	GetModels() ([]*Model, error)

	GetTextures() *Textures

	RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error

	RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error

	GetRawFaces(modelIdx int) ([]*RawFace, error)

	GetExternalBModelFileName(classname string) string

	GetModelFileName(classname string) string
}

// NewBSPReader creates and initializes an IBSPReader for the specified BSP file using the provided IArchive instance.
func NewBSPReader(arc IArchive, bspPath string) (IBSPReader, error) {
	rs, rErr := arc.Open(bspPath)
	if rErr != nil {
		return nil, fmt.Errorf("can't open %s: %s", bspPath, rErr.Error())
	}
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
			return NewQ3BSPReader(arc, rs), nil
		case BSPVersionQ2:
			palette, _ := arc.Open("gfx" + PakSeparator + "palette.lmp")
			return NewQ2BSPReader(arc, rs, palette), nil
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
	palette, _ := arc.Open("gfx" + PakSeparator + "palette.lmp")
	return NewQ1BSPReader(arc, rs, palette), nil
}

// NewArchive creates a new archive instance based on the file extension, supporting ".pk3" and other formats.
func NewArchive(pakPath string) (IArchive, error) {
	if strings.HasSuffix(strings.ToLower(pakPath), ".pk3") {
		return NewPk3(), nil
	}
	return NewPak(), nil
}
