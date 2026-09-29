package lumps

import (
	"encoding/binary"
	"io"
)

// LumpQ3Entities represents the lump index for Quake 3 entities data.
// LumpQ3Textures represents the lump index for Quake 3 texture data.
// LumpQ3Planes represents the lump index for Quake 3 plane data.
// LumpQ3Nodes represents the lump index for Quake 3 node data.
// LumpQ3Leafs represents the lump index for Quake 3 leaf node data.
// LumpQ3LeafFaces represents the lump index for Quake 3 leaf face data.
// LumpQ3LeafBrushes represents the lump index for Quake 3 leaf brush data.
// LumpQ3Models represents the lump index for Quake 3 model data.
// LumpQ3Brushes represents the lump index for Quake 3 brush data.
// LumpQ3BrushSides represents the lump index for Quake 3 brush side data.
// LumpQ3Vertexes represents the lump index for Quake 3 vertex data.
// LumpQ3MeshVerts represents the lump index for Quake 3 mesh vertex data.
// LumpQ3Effects represents the lump index for Quake 3 effect data.
// LumpQ3Faces represents the lump index for Quake 3 face data.
// LumpQ3Lightmaps represents the lump index for Quake 3 lightmap data.
// LumpQ3LightVols represents the lump index for Quake 3 light volume data.
// LumpQ3VisData represents the lump index for Quake 3 visibility data.
// NumQ3Lumps represents the total number of lumps in a Quake 3 BSP file.
const (
	LumpEntities3    = 0
	LumpTextures3    = 1
	LumpPlanes3      = 2
	LumpNodes3       = 3
	LumpLeafs3       = 4
	LumpLeafFaces3   = 5
	LumpLeafBrushes3 = 6
	LumpModels3      = 7
	LumpBrushes3     = 8
	LumpBrushSides3  = 9
	LumpVertexes3    = 10
	LumpMeshVerts3   = 11
	LumpEffects3     = 12
	LumpFaces3       = 13
	LumpLightmaps3   = 14
	LumpLightVols3   = 15
	LumpVisData3     = 16
	NumLumps3        = 17
)

// Headers3 represents the header structure of a Quake 3 BSP file, including magic, version, and lump directory entries.
type Headers3 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumLumps3]struct {
		Offset int32
		Length int32
	}
}

// NewHeader3 reads binary data from the provided io.Reader and parses it into a Headers3 structure or returns an error.
func NewHeader3(rs io.Reader) (Headers3, error) {
	var header Headers3
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return Headers3{}, err
	}
	return header, nil
}
