package lumps

import (
	"encoding/binary"
	"io"
	"unsafe"
)

// Face represents a sector in a game level, defining its geometry, textures, lighting, and special properties.
type Face struct {
	PlaneID    uint16
	Side       uint16
	FirstEdge  int32 // Offset base nell'array SurfEdges
	NumEdges   uint16
	TexInfo    uint16 // Indice per l'UV mapping
	LightTypes [4]uint8
	Lightmap   int32
}

// NewFace reads face data from the provided file and converts it to a slice of Face structs based on lump metadata.
// Returns a slice of Face structs or an error if reading or parsing fails.
func NewFace(rs io.ReadSeeker, lumpInfo *LumpInfo) ([]*Face, error) {
	type privateFace struct {
		PlaneID    uint16
		Side       uint16
		FirstEdge  int32
		NumEdges   uint16
		TexInfo    uint16
		LightTypes [4]uint8
		Lightmap   int32
	}
	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}
	var pFace privateFace
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pFace))
	pSectors := make([]privateFace, count)
	if err := binary.Read(rs, binary.LittleEndian, pSectors); err != nil {
		return nil, err
	}
	sectors := make([]*Face, count)
	for idx, p := range pSectors {
		sectors[idx] = &Face{
			PlaneID:    p.PlaneID,
			Side:       p.Side,
			FirstEdge:  p.FirstEdge,
			NumEdges:   p.NumEdges,
			TexInfo:    p.TexInfo,
			LightTypes: p.LightTypes,
			Lightmap:   p.Lightmap,
		}
	}
	return sectors, nil
}
