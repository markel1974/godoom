package lumps

import (
	"encoding/binary"
	"io"
)

// Model2 represents a 3D model structure containing bounding box data, spatial origin, and face indexing information.
type Model2 struct {
	Mins      [3]float32
	Maxs      [3]float32
	Origin    [3]float32
	HeadNode  int32
	FirstFace int32
	NumFaces  int32
}

// NewModels2 parses model data from the specified ReadSeeker based on file headers and returns a slice of Model2 pointers.
func NewModels2(rs io.ReadSeeker, headers Headers2) ([]*Model2, error) {
	lumpModels := headers.Lumps[LumpModels2]
	if _, err := rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]Model2, numModels)
	if err := binary.Read(rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	out := make([]*Model2, numModels)
	for i, m := range models {
		out[i] = &Model2{
			Mins:      m.Mins,
			Maxs:      m.Maxs,
			Origin:    m.Origin,
			HeadNode:  m.HeadNode,
			FirstFace: m.FirstFace,
			NumFaces:  m.NumFaces,
		}
	}
	return out, nil
}
