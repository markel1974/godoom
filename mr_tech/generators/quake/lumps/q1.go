// lumps_q1_wrapper.go
package lumps

import (
	"fmt"
	"io"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Q1BSPReader reads and processes Quake 1 BSP files by managing lumps, textures, and geometry data.
type Q1BSPReader struct {
	reader      IReader
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

// NewQ1BSPReader initializes and returns a pointer to a new Q1BSPReader instance using the provided io.ReadSeeker streams.
func NewQ1BSPReader(rs io.ReadSeeker, rsPal io.ReadSeeker) *Q1BSPReader {
	return &Q1BSPReader{
		rs:         rs,
		rsPal:      rsPal,
		texManager: NewTextures(),
	}
}

// Setup initializes the Q1BSPReader by loading BSP data, textures, palettes, and related metadata from the provided reader.
func (q1 *Q1BSPReader) Setup(reader IReader) error {
	q1.reader = reader
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

// GetEntities retrieves all entities from the BSP file and returns them as a slice of Entity pointers or an error.
func (q1 *Q1BSPReader) GetEntities() ([]*Entity, error) {
	return NewEntities(q1.rs, q1.infos[LumpEntities])
}

// GetModels retrieves the BSP models from the lump data and returns a slice of models or an error if reading fails.
func (q1 *Q1BSPReader) GetModels() ([]*Model, error) {
	return NewModels(q1.rs, q1.infos[LumpModels])
}

// GetTextures returns the texture manager instance containing textures defined in the BSP file.
func (q1 *Q1BSPReader) GetTextures() *Textures {
	return q1.texManager
}

// RegisterPixels registers a texture by name with specified dimensions, pixel data, palette, transparency, and alignment.
func (q1 *Q1BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q1.texManager.RegisterPixels(name, width, height, indices, q1.palette, isTransparent, transIndex, invertY)
}

// RegisterPixelsRGBA registers a texture using raw RGBA pixel data with optional Y-axis inversion.
func (q1 *Q1BSPReader) RegisterPixelsRGBA(name string, width, height int, pixels []byte, invertY bool) error {
	return q1.texManager.RegisterPixelsRGBA(name, width, height, pixels, invertY)
}

// GetRawFaces extracts raw face data for a specified model index, including geometry, texture names, and UV coordinates.
func (q1 *Q1BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	models, _ := q1.GetModels()
	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("invalid model index")
	}
	model := models[modelIdx]

	var rawFaces []*RawFace

	// Itera solo sulle facce di questo modello (0 = World, 1+ = BModels)
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
		var uvs [][2]float64

		// Prepare texture width/height for normalization
		texW, texH := float64(256), float64(256)
		if texInfo.MipTex < uint32(len(q1.mipTextures)) && q1.mipTextures[texInfo.MipTex] != nil {
			texW = float64(q1.mipTextures[texInfo.MipTex].Width)
			texH = float64(q1.mipTextures[texInfo.MipTex].Height)
			if texW == 0 {
				texW = 256
			}
			if texH == 0 {
				texH = 256
			}
		}

		for j := uint16(0); j < bspFace.NumEdges; j++ {
			surfEdgeIdx := q1.surfEdges[bspFace.FirstEdge+int32(j)]
			var v *Vertex
			if surfEdgeIdx >= 0 {
				v = q1.vertexes[q1.edges[surfEdgeIdx].Vertex0]
			} else {
				v = q1.vertexes[q1.edges[-surfEdgeIdx].Vertex1]
			}
			pos := CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z))
			points = append(points, pos)

			// Compute UVs using Quake 1 vector projection
			// Note: Q1 raw vertex coords are used for projection
			u := (float64(v.X) * float64(texInfo.Vecs[0][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[0][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[0][2])) +
				float64(texInfo.Vecs[0][3])
			vt := (float64(v.X) * float64(texInfo.Vecs[1][0])) +
				(float64(v.Y) * float64(texInfo.Vecs[1][1])) +
				(float64(v.Z) * float64(texInfo.Vecs[1][2])) +
				float64(texInfo.Vecs[1][3])

			uvs = append(uvs, [2]float64{u / texW, vt / texH})
		}

		rawFaces = append(rawFaces, &RawFace{
			Points:  points,
			UVs:     uvs,
			TexName: texName,
			IsSky:   isSky,
		})
	}

	return rawFaces, nil
}

// GetExternalBModelFileName returns the file name of the external BSP model associated with the given classname.
func (q1 *Q1BSPReader) GetExternalBModelFileName(classname string) string {
	return _q1DictBModel[classname]
}

// GetModelFileName returns the file name of a model corresponding to the given classname from the predefined model dictionary.
func (q1 *Q1BSPReader) GetModelFileName(classname string) string {
	return _q1DictModelFilename[classname]
}

// getVertexes retrieves the vertex data from the BSP file using the lump information and returns a slice of vertex pointers.
func (q1 *Q1BSPReader) getVertexes() ([]*Vertex, error) {
	return NewVertexes(q1.rs, q1.infos[LumpVertexes])
}

// getEdges loads and returns the list of edges from the edges lump data or an error if the operation fails.
func (q1 *Q1BSPReader) getEdges() ([]*Edge, error) {
	return NewEdges(q1.rs, q1.infos[LumpEdges])
}

// getSurfEdges retrieves an array of surface edges, representing directed edge indices used for face definitions.
// Returns a slice of int32 values and an error if reading fails.
func (q1 *Q1BSPReader) getSurfEdges() ([]int32, error) {
	return NewSurfEdges(q1.rs, q1.infos[LumpSurfEdges])
}

// getFaces retrieves all Face structures from the BSP file using lump metadata and returns them or an error if encountered.
func (q1 *Q1BSPReader) getFaces() ([]*Face, error) {
	return NewFace(q1.rs, q1.infos[LumpFaces])
}

// getTexInfos retrieves texture mapping information from the level data and returns a slice of TexInfo and an error.
func (q1 *Q1BSPReader) getTexInfos() ([]*TexInfo, error) {
	return NewTexInfos(q1.rs, q1.infos[LumpTexInfos])
}

// getMipTextures reads and decodes all mipmap textures from the lump data in the BSP file. Returns an error on failure.
func (q1 *Q1BSPReader) getMipTextures() ([]*MipTexture, error) {
	return NewMipTextures(q1.rs, q1.infos[LumpTextures])
}
