package lumps

import (
	"encoding/binary"
	"io"
)

// Model3 represents a 3D model in a Quake 3 BSP file, including bounds and references to associated geometry data.
type Model3 struct {
	Mins       [3]float32
	Maxs       [3]float32
	FirstFace  int32
	NumFaces   int32
	FirstBrush int32
	NumBrushes int32
}

// NewModels3 reads and parses a list of models from a Quake 3 BSP file using the provided header and read-seeker.
func NewModels3(rs io.ReadSeeker, header Headers3) ([]*Model, error) {
	lModels := header.Lumps[LumpModels3]
	if _, err := rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lModels.Length) / 40
	models := make([]Model3, numModels)
	if err := binary.Read(rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}
	out := make([]*Model, numModels)
	for i, m := range models {
		out[i] = &Model{
			Mins:      m.Mins,
			Maxs:      m.Maxs,
			FirstFace: m.FirstFace,
			NumFaces:  m.NumFaces,
		}
	}
	return out, nil
}
