// lumps_q1_wrapper.go
package lumps

import (
	"io"
)

// Q1BSPReader is a reader for Quake 1 BSP files, providing access to lump data and associated metadata.
type Q1BSPReader struct {
	rs      io.ReadSeeker
	rsPal   io.ReadSeeker
	infos   []*LumpInfo
	palette []byte
}

// NewQ1BSPReader initializes a Q1BSPReader for reading Quake 1 BSP files and a palette from the provided io.ReadSeekers.
func NewQ1BSPReader(rs io.ReadSeeker, rsPal io.ReadSeeker) *Q1BSPReader {
	return &Q1BSPReader{rs: rs, rsPal: rsPal}
}

// Setup initializes the Q1BSPReader by processing lump information and palette data from the provided file streams.
func (q1 *Q1BSPReader) Setup() error {
	var err error
	q1.infos, err = NewLumpInfos(q1.rs)
	if err != nil {
		return err
	}
	q1.palette, err = NewPalette(q1.rsPal)
	if err != nil {
		return err
	}
	return nil
}

// GetEntities retrieves a list of entities by parsing the lump data for entities in the BSP file. Returns entities or an error.
func (q1 *Q1BSPReader) GetEntities() ([]*Entity, error) {
	return NewEntities(q1.rs, q1.infos[LumpEntities])
}

// GetVertexes retrieves a slice of Vertex pointers from the BSP file and returns an error if the operation fails.
func (q1 *Q1BSPReader) GetVertexes() ([]*Vertex, error) {
	return NewVertexes(q1.rs, q1.infos[LumpVertexes])
}

// GetEdges reads and returns the list of edges from the BSP file, represented as directed line segments.
func (q1 *Q1BSPReader) GetEdges() ([]*Edge, error) {
	return NewEdges(q1.rs, q1.infos[LumpEdges])
}

// GetSurfEdges retrieves an array of int32 representing surface edge indices from the BSP file.
// A negative index indicates reversed edge vertex order. Returns an error if reading fails.
func (q1 *Q1BSPReader) GetSurfEdges() ([]int32, error) {
	return NewSurfEdges(q1.rs, q1.infos[LumpSurfEdges])
}

// GetFaces retrieves all face structures from the BSP file using metadata from the LumpFaces lump.
// Returns a slice of Face pointers or an error if the operation fails.
func (q1 *Q1BSPReader) GetFaces() ([]*Face, error) {
	return NewFace(q1.rs, q1.infos[LumpFaces])
}

// GetTexInfos retrieves texture mapping information from the loaded BSP file and returns a slice of TexInfo pointers.
func (q1 *Q1BSPReader) GetTexInfos() ([]*TexInfo, error) {
	return NewTexInfos(q1.rs, q1.infos[LumpTexInfo])
}

// GetMipTextures retrieves all *MipTexture objects from the TEXTURES lump in the BSP file. Returns an error on failure.
func (q1 *Q1BSPReader) GetMipTextures() ([]*MipTexture, error) {
	return NewMipTextures(q1.rs, q1.infos[LumpTextures])
}

// GetModels reads and returns all BSP sub-models as a slice of Model structs or an error if loading fails.
func (q1 *Q1BSPReader) GetModels() ([]*Model, error) {
	return NewModels(q1.rs, q1.infos[LumpModels])
}

// GetPalette retrieves the palette data from the BSP file. It returns the palette as a byte slice or an error if unavailable.
func (q1 *Q1BSPReader) GetPalette() ([]byte, error) {
	return q1.palette, nil
}
