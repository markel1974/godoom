package lumps

import (
	"encoding/binary"
	"os"
	"unsafe"
)

// Leaf represents a node in a BSP tree, containing spatial data and references for rendering and sound.
type Leaf struct {
	Contents   int32
	VisOffset  int32
	Mins       [3]int16 // AABB per clipping rapido
	Maxs       [3]int16
	FirstFace  uint16 // Indice nel lump di indirezione MarkSurfaces
	NumFaces   uint16
	AmbientSnd [4]uint8
}

// NewLeaves reads and parses lump data from a file into a slice of Leaf pointers based on the provided LumpInfo meta information.
func NewLeaves(f *os.File, lumpInfo *LumpInfo) ([]*Leaf, error) {
	var pLeaf Leaf
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pLeaf))
	pLeaves := make([]Leaf, count)
	if err := binary.Read(f, binary.LittleEndian, pLeaves); err != nil {
		return nil, err
	}
	leaves := make([]*Leaf, count)
	for idx, s := range pLeaves {
		leaves[idx] = &Leaf{
			Contents:   s.Contents,
			VisOffset:  s.VisOffset,
			Mins:       s.Mins,
			Maxs:       s.Maxs,
			FirstFace:  s.FirstFace,
			NumFaces:   s.NumFaces,
			AmbientSnd: s.AmbientSnd,
		}
	}
	return leaves, nil
}
