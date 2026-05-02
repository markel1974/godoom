package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
)

// BSPVersion represents the versioning enumeration for BSP (Binary Space Partitioning) structures in a system.
type BSPVersion int

// BSPVersionQ1 represents the BSP version used in Quake 1.
// BSPVersionQ2 represents the BSP version used in Quake 2.
const (
	BSPVersionQ1 BSPVersion = 29
	BSPVersionQ2 BSPVersion = 38
)

// IBSPReader defines an interface for reading BSP files, enabling access to entities and other BSP structures.
type IBSPReader interface {
	Setup() error

	GetEntities() ([]*Entity, error)

	GetModels() ([]*Model, error)

	GetTextures() *Textures

	RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error

	GetRawFaces(modelIdx int) ([]*RawFace, error)
}

// Factory detects the BSP file version from the provided io.ReadSeeker and returns an appropriate IBSPReader implementation.
func Factory(rs io.ReadSeeker, palette io.ReadSeeker) (IBSPReader, error) {
	var magic [4]byte
	if err := binary.Read(rs, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic bytes: %w", err)
	}

	// Quake 2 (IBSP)
	if string(magic[:]) == "IBSP" {
		var version int32
		if err := binary.Read(rs, binary.LittleEndian, &version); err != nil {
			return nil, err
		}
		if version != int32(BSPVersionQ2) {
			return nil, fmt.Errorf("unsupported IBSP version: %d", version)
		}
		// Riavvolgiamo lo stream all'inizio prima di passare al parser specifico
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("failed to rewind stream: %w", err)
		}
		return NewQ2BSPReader(rs, palette), nil
	}

	// Quake 1 (Nessun magic "IBSP", inizia direttamente con la versione 29)
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
