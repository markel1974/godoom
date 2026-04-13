package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// LumpEntities represents the lump index for entities.
// LumpPlanes represents the lump index for planes.
// LumpTextures represents the lump index for textures.
// LumpVertexes represents the lump index for vertex data.
// LumpVisibility represents the lump index for visibility information.
// LumpNodes represents the lump index for nodes.
// LumpTexInfo represents the lump index for texture information.
// LumpFaces represents the lump index for faces.
// LumpLighting represents the lump index for lighting information.
// LumpClipNodes represents the lump index for clip nodes.
// LumpLeaves represents the lump index for leaves.
// LumpMarkSurfaces represents the lump index for marked surfaces.
// LumpEdges represents the lump index for edges.
// LumpSurfEdges represents the lump index for surface edges.
// LumpModels represents the lump index for models.
// NumLumps represents the total number of lump types.
const (
	LumpEntities = iota
	LumpPlanes
	LumpTextures
	LumpVertexes
	LumpVisibility
	LumpNodes
	LumpTexInfo
	LumpFaces
	LumpLighting
	LumpClipNodes
	LumpLeaves
	LumpMarkSurfaces
	LumpEdges
	LumpSurfEdges
	LumpModels
	NumLumps
)

// Header represents the structure of a BSP file header containing version and lump directory information.
type Header struct {
	Version int32
	Lumps   [NumLumps]struct {
		Offset int32
		Length int32
	}
}

// LumpInfo represents metadata for a lump, including its file position, size, and name.
type LumpInfo struct {
	Filepos int64
	Size    int32
	Name    string
}

// NewLumpInfo creates and returns a pointer to a LumpInfo instance with the specified position, size, and name initialized.
func NewLumpInfo(pos int64, size int32, name string) *LumpInfo {
	return &LumpInfo{
		Filepos: pos,
		Size:    size,
		Name:    name,
	}
}

// getLumpName returns the name of the lump corresponding to the provided index or "UNKNOWN" if the index is out of bounds.
func getLumpName(index int) string {
	names := []string{
		"ENTITIES", "PLANES", "TEXTURES", "VERTEXES", "VISIBILITY",
		"NODES", "TEXINFO", "FACES", "LIGHTING", "CLIPNODES",
		"LEAVES", "MARKSURFACES", "EDGES", "SURFEDGES", "MODELS",
	}
	if index >= 0 && index < len(names) {
		return names[index]
	}
	return "UNKNOWN"
}

// NewLumpInfos reads and parses header information from the given file, returning a slice of LumpInfo structs.
func NewLumpInfos(rs io.ReadSeeker) ([]*LumpInfo, error) {
	if _, err := rs.Seek(0, os.SEEK_SET); err != nil {
		return nil, err
	}
	header := &Header{}
	if err := binary.Read(rs, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	if header.Version != 29 {
		return nil, fmt.Errorf("bad BSP version: %d (expected 29)", header.Version)
	}
	lumpInfos := make([]*LumpInfo, NumLumps)
	for i := 0; i < NumLumps; i++ {
		lump := header.Lumps[i]
		lumpInfos[i] = NewLumpInfo(int64(lump.Offset), lump.Length, getLumpName(i))
	}
	return lumpInfos, nil
}
