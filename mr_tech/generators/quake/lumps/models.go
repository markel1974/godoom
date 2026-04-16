package lumps

import (
	"encoding/binary"
	"io"
	"unsafe"
)

// Model represents a BSP sub-model (BModel).
// Model 0 is the static world geometry, while subsequent models are moving brushes (doors, platforms, etc).
type Model struct {
	Mins      [3]float32
	Maxs      [3]float32
	Origin    [3]float32
	HeadNode  [4]int32
	VisLeafs  int32
	FirstFace int32
	NumFaces  int32
}

// NewModels reads model data from the provided file and converts it to a slice of Model structs based on lump metadata.
// Returns a slice of Model structs or an error if reading or parsing fails.
func NewModels(rs io.ReadSeeker, lumpInfo *LumpInfo) ([]*Model, error) {
	type privateModel struct {
		Mins      [3]float32
		Maxs      [3]float32
		Origin    [3]float32
		HeadNode  [4]int32
		VisLeafs  int32
		FirstFace int32
		NumFaces  int32
	}

	if err := Seek(rs, lumpInfo.Filepos); err != nil {
		return nil, err
	}

	var pModel privateModel
	count := int(lumpInfo.Size) / int(unsafe.Sizeof(pModel))
	pModels := make([]privateModel, count)

	if err := binary.Read(rs, binary.LittleEndian, pModels); err != nil {
		return nil, err
	}

	models := make([]*Model, count)
	for idx, p := range pModels {
		models[idx] = &Model{
			Mins:      p.Mins,
			Maxs:      p.Maxs,
			Origin:    p.Origin,
			HeadNode:  p.HeadNode,
			VisLeafs:  p.VisLeafs,
			FirstFace: p.FirstFace,
			NumFaces:  p.NumFaces,
		}
	}

	return models, nil
}
