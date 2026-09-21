package quake

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/generators/quake/interfaces"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/generators/quake/q1"
	"github.com/markel1974/godoom/mr_tech/generators/quake/q2"
	"github.com/markel1974/godoom/mr_tech/generators/quake/q3"
)

// BSPVersion represents the version of a BSP (Binary Space Partitioning) file used in different game engines.
type BSPVersion int

// BSPVersionQ1 represents the BSP version for Quake 1 files.
// BSPVersionQ2 represents the BSP version for Quake 2 files.
// BSPVersionQ3 represents the BSP version for Quake 3 files.
const (
// BSPVersionQ1 BSPVersion = 29
// BSPVersionQ2 BSPVersion = 38
// BSPVersionQ3 BSPVersion = 46
)

// NewBSPReader creates and initializes an IBSPReader for the specified BSP file using the provided IArchive instance.
func NewBSPReader(arc interfaces.IArchive, bspPath string) (interfaces.IBSPReader, error) {
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
		switch int(version) {
		case q3.BSPVersionQ3:
			return q3.NewQ3BSPReader(arc, rs), nil
		case q2.BSPVersionQ2:
			return q2.NewQ2BSPReader(arc, rs), nil
		default:
			return nil, fmt.Errorf("unsupported IBSP version: %d", version)
		}
	}

	// Quake 1 (version 29)
	return q1.NewQ1BSPReader(arc, rs), nil
}

// NewArchive creates a new reader.go instance based on the file extension, supporting ".pk3" and other formats.
func NewArchive(pakPath string) (interfaces.IArchive, error) {
	if strings.HasSuffix(strings.ToLower(pakPath), ".pk3") {
		return lumps.NewPk3(), nil
	}
	return lumps.NewPak(), nil
}
