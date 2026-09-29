package lumps

import (
	"encoding/binary"
	"io"
)

// LumpQ3Entities represents the lump index for storing entity data in a Quake 3 map.
// LumpQ3Textures represents the lump index for storing texture data in a Quake 3 map.
// LumpQ3Planes represents the lump index for storing plane data in a Quake 3 map.
// LumpQ3Nodes represents the lump index for storing node data in a Quake 3 map.
// LumpQ3Leafs represents the lump index for storing leaf data in a Quake 3 map.
// LumpQ3LeafFaces represents the lump index for storing leaf face data in a Quake 3 map.
// LumpQ3LeafBrushes represents the lump index for storing leaf brush data in a Quake 3 map.
// LumpQ3Models represents the lump index for storing model data in a Quake 3 map.
// LumpQ3Brushes represents the lump index for storing brush data in a Quake 3 map.
// LumpQ3BrushSides represents the lump index for storing brush side data in a Quake 3 map.
// LumpQ3Vertexes represents the lump index for storing vertex data in a Quake 3 map.
// LumpQ3MeshVerts represents the lump index for storing mesh vertex data in a Quake 3 map.
// LumpQ3Effects represents the lump index for storing special effect data in a Quake 3 map.
// LumpQ3Faces represents the lump index for storing face data in a Quake 3 map.
// LumpQ3Lightmaps represents the lump index for storing lightmap data in a Quake 3 map.
// LumpQ3LightVols represents the lump index for storing light volume data in a Quake 3 map.
// LumpQ3VisData represents the lump index for storing visibility data in a Quake 3 map.
// NumQ3Lumps represents the total number of lumps in a Quake 3 map.
const (
	LumpQ3Entities    = 0
	LumpQ3Textures    = 1
	LumpQ3Planes      = 2
	LumpQ3Nodes       = 3
	LumpQ3Leafs       = 4
	LumpQ3LeafFaces   = 5
	LumpQ3LeafBrushes = 6
	LumpQ3Models      = 7
	LumpQ3Brushes     = 8
	LumpQ3BrushSides  = 9
	LumpQ3Vertexes    = 10
	LumpQ3MeshVerts   = 11
	LumpQ3Effects     = 12
	LumpQ3Faces       = 13
	LumpQ3Lightmaps   = 14
	LumpQ3LightVols   = 15
	LumpQ3VisData     = 16
	NumQ3Lumps        = 17
)

// Header3 represents the header structure of a Quake 3 BSP file, containing metadata and lump information.
type Header3 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ3Lumps]struct {
		Offset int32
		Length int32
	}
}

func NewHeader3(rs io.Reader) (Header3, error) {
	var header Header3
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return Header3{}, err
	}
	return header, nil
}
