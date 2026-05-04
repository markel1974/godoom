// lumps_q1_wrapper.go
package lumps

import (
	"fmt"
	"io"

	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// Q1BSPReader is a reader for Quake 1 BSP files, providing access to lump data and associated metadata.
type Q1BSPReader struct {
	rs          io.ReadSeeker
	rsPal       io.ReadSeeker
	infos       []*LumpInfo
	palette     []byte
	mipTextures []*MipTexture
	faces       []*Face
	surfEdges   []int32
	edges       []*Edge
	vertexes    []*Vertex
	texInfos    []*TexInfo
	texManager  *Textures
}

// NewQ1BSPReader initializes a Q1BSPReader for reading Quake 1 BSP files and a palette from the provided io.ReadSeekers.
func NewQ1BSPReader(rs io.ReadSeeker, rsPal io.ReadSeeker) *Q1BSPReader {
	return &Q1BSPReader{
		rs:         rs,
		rsPal:      rsPal,
		texManager: NewTextures(),
	}
}

// Setup initializes the Q1BSPReader by processing lump information and palette data from the provided file streams.
func (q1 *Q1BSPReader) Setup() error {
	var err error
	if q1.infos, err = NewLumpInfos(q1.rs); err != nil {
		return err
	}
	if q1.palette, err = NewPalette(q1.rsPal); err != nil {
		return err
	}
	if q1.faces, err = q1.getFaces(); err != nil {
		return err
	}
	if q1.surfEdges, err = q1.getSurfEdges(); err != nil {
		return err
	}
	if q1.edges, err = q1.getEdges(); err != nil {
		return err
	}
	if q1.vertexes, err = q1.getVertexes(); err != nil {
		return err
	}
	if q1.texInfos, err = q1.getTexInfos(); err != nil {
		return err
	}
	if q1.mipTextures, err = q1.getMipTextures(); err != nil {
		return err
	}
	for _, mt := range q1.mipTextures {
		if mt != nil && mt.Name != "" {
			if err = q1.RegisterPixels(mt.Name, int(mt.Width), int(mt.Height), mt.Pixels[0], false, 255, false); err != nil {
				fmt.Printf("Warning: texture %s error: %s\n", mt.Name, err.Error())
			}
		}
	}
	return nil
}

// GetEntities retrieves a list of entities by parsing the lump data for entities in the BSP file. Returns entities or an error.
func (q1 *Q1BSPReader) GetEntities() ([]*Entity, error) {
	return NewEntities(q1.rs, q1.infos[LumpEntities])
}

// GetModels reads and returns all BSP sub-models as a slice of Model structs or an error if loading fails.
func (q1 *Q1BSPReader) GetModels() ([]*Model, error) {
	return NewModels(q1.rs, q1.infos[LumpModels])
}

// GetTextures returns a pointer to the Textures manager associated with the Q1BSPReader instance.
func (q1 *Q1BSPReader) GetTextures() *Textures {
	return q1.texManager
}

// RegisterPixels registers a texture's pixel data with the specified attributes in the Q1BSPReader's texture manager.
func (q1 *Q1BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q1.texManager.RegisterPixels(name, width, height, indices, q1.palette, isTransparent, transIndex, invertY)
}

func (q1 *Q1BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	// 1. Carica i dati base
	models, _ := q1.GetModels()
	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("invalid model index")
	}
	model := models[modelIdx]

	var rawFaces []*RawFace

	// 2. Itera solo sulle facce di questo modello (0 = World, 1+ = BModels)
	for i := int32(0); i < model.NumFaces; i++ {
		faceIdx := model.FirstFace + i
		bspFace := q1.faces[faceIdx]
		texInfo := q1.texInfos[bspFace.TexInfo]
		texName := "default"
		isSky := (texInfo.Flags & 4) != 0
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
			texName = q1.mipTextures[texInfo.MipTex].Name
		}
		var points []geometry.XYZ
		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := q1.surfEdges[bspFace.FirstEdge+int32(j)]
			var v *Vertex
			if surfEdgeIdx >= 0 {
				v = q1.vertexes[q1.edges[surfEdgeIdx].Vertex0]
			} else {
				v = q1.vertexes[q1.edges[-surfEdgeIdx].Vertex1]
			}
			points = append(points, CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z)))
		}

		rawFaces = append(rawFaces, &RawFace{
			Points:  points,
			TexName: texName,
			IsSky:   isSky,
		})
	}

	return rawFaces, nil
}

// GetVertexes retrieves a slice of Vertex pointers from the BSP file and returns an error if the operation fails.
func (q1 *Q1BSPReader) getVertexes() ([]*Vertex, error) {
	return NewVertexes(q1.rs, q1.infos[LumpVertexes])
}

// GetEdges reads and returns the list of edges from the BSP file, represented as directed line segments.
func (q1 *Q1BSPReader) getEdges() ([]*Edge, error) {
	return NewEdges(q1.rs, q1.infos[LumpEdges])
}

// GetSurfEdges retrieves an array of int32 representing surface edge indices from the BSP file.
// A negative index indicates reversed edge vertex order. Returns an error if reading fails.
func (q1 *Q1BSPReader) getSurfEdges() ([]int32, error) {
	return NewSurfEdges(q1.rs, q1.infos[LumpSurfEdges])
}

// GetFaces retrieves all face structures from the BSP file using metadata from the LumpFaces lump.
// Returns a slice of Face pointers or an error if the operation fails.
func (q1 *Q1BSPReader) getFaces() ([]*Face, error) {
	return NewFace(q1.rs, q1.infos[LumpFaces])
}

// GetTexInfos retrieves texture mapping information from the loaded BSP file and returns a slice of TexInfo pointers.
func (q1 *Q1BSPReader) getTexInfos() ([]*TexInfo, error) {
	return NewTexInfos(q1.rs, q1.infos[LumpTexInfos])
}

// GetMipTextures retrieves all *MipTexture objects from the TEXTURES lump in the BSP file. Returns an error on failure.
func (q1 *Q1BSPReader) getMipTextures() ([]*MipTexture, error) {
	return NewMipTextures(q1.rs, q1.infos[LumpTextures])
}
