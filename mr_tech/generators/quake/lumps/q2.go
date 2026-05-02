package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
)

// LumpQ2Entities represents the lump index for entities in a Quake 2 BSP file.
// LumpQ2Planes represents the lump index for planes in a Quake 2 BSP file.
// LumpQ2Vertexes represents the lump index for vertex data in a Quake 2 BSP file.
// LumpQ2Visibility represents the lump index for visibility data in a Quake 2 BSP file.
// LumpQ2Nodes represents the lump index for nodes in a Quake 2 BSP file.
// LumpQ2TexInfo represents the lump index for texture information in a Quake 2 BSP file.
// LumpQ2Faces represents the lump index for faces in a Quake 2 BSP file.
// LumpQ2Lighting represents the lump index for lighting data in a Quake 2 BSP file.
// LumpQ2Leaves represents the lump index for BSP tree leaves in a Quake 2 BSP file.
// LumpQ2LeafFaces represents the lump index for leaf-face references in a Quake 2 BSP file.
// LumpQ2LeafBrushes represents the lump index for leaf-brush references in a Quake 2 BSP file.
// LumpQ2Edges represents the lump index for edges in a Quake 2 BSP file.
// LumpQ2SurfEdges represents the lump index for surface edge references in a Quake 2 BSP file.
// LumpQ2Models represents the lump index for models in a Quake 2 BSP file.
// LumpQ2Brushes represents the lump index for brushes in a Quake 2 BSP file.
// LumpQ2BrushSides represents the lump index for brush sides in a Quake 2 BSP file.
// LumpQ2Pop represents the lump index for the Pop field, typically unused in a Quake 2 BSP file.
// LumpQ2Areas represents the lump index for area definitions in a Quake 2 BSP file.
// LumpQ2AreaPortals represents the lump index for area portals in a Quake 2 BSP file.
// NumQ2Lumps represents the total number of lumps in a Quake 2 BSP file.
const (
	LumpQ2Entities    = 0
	LumpQ2Planes      = 1
	LumpQ2Vertexes    = 2
	LumpQ2Visibility  = 3
	LumpQ2Nodes       = 4
	LumpQ2TexInfo     = 5
	LumpQ2Faces       = 6
	LumpQ2Lighting    = 7
	LumpQ2Leaves      = 8
	LumpQ2LeafFaces   = 9
	LumpQ2LeafBrushes = 10
	LumpQ2Edges       = 11
	LumpQ2SurfEdges   = 12
	LumpQ2Models      = 13
	LumpQ2Brushes     = 14
	LumpQ2BrushSides  = 15
	LumpQ2Pop         = 16
	LumpQ2Areas       = 17
	LumpQ2AreaPortals = 18
	NumQ2Lumps        = 19
)

// HeaderQ2 represents the structure of the header in a Quake 2 BSP file format.
// Magic contains the magic number identifying the file type.
// Version specifies the version of the BSP file.
// Lumps is an array holding lump metadata such as offset and length.
type HeaderQ2 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ2Lumps]struct {
		Offset int32
		Length int32
	}
}

// Q2BSPReader provides methods to read and parse Quake 2 BSP (Binary Space Partitioning) map files.
type Q2BSPReader struct {
	header     HeaderQ2
	rs         io.ReadSeeker
	rsPal      io.ReadSeeker
	palette    []byte
	texManager *Textures
}

// NewQ2BSPReader initializes a Q2BSPReader for reading Quake 2 BSP files and a palette from the provided io.ReadSeekers.
func NewQ2BSPReader(rs io.ReadSeeker, rsPal io.ReadSeeker) *Q2BSPReader {
	reader := &Q2BSPReader{
		rs:         rs,
		texManager: NewTextures(),
		rsPal:      rsPal,
	}
	return reader
}

// Setup initializes the Q2BSPReader, preparing it for reading data from the associated io.ReadSeeker.
func (q2 *Q2BSPReader) Setup() error {
	var err error
	if err = binary.Read(q2.rs, binary.LittleEndian, &q2.header); err != nil {
		return err
	}
	q2.palette, err = NewPalette(q2.rsPal)
	if err != nil {
		return err
	}
	return fmt.Errorf("implement me")
}

// GetEntities reads and parses the entity data from the lump and returns a slice of entity pointers or an error.
func (q2 *Q2BSPReader) GetEntities() ([]*Entity, error) {
	lump := q2.header.Lumps[LumpQ2Entities]
	_, _ = q2.rs.Seek(int64(lump.Offset), io.SeekStart)
	data := make([]byte, lump.Length)
	if _, err := q2.rs.Read(data); err != nil {
		return nil, err
	}
	text := FromNullTerminatingString(data)
	return parseEntityText(text) // Usiamo la tua funzione esistente!
}

// GetVertexes reads and returns the list of Vertex objects from the BSP file. It returns an error if reading fails.
func (q2 *Q2BSPReader) GetVertexes() ([]*Vertex, error) {
	return nil, fmt.Errorf("implement me")
}

// GetEdges reads and returns all edges from the BSP file along with any error encountered during the process.
func (q2 *Q2BSPReader) GetEdges() ([]*Edge, error) {
	return nil, fmt.Errorf("implement me")
}

// GetSurfEdges retrieves the list of surface edge indices from the BSP file and returns them as a slice of int32 values.
// It returns an error if the operation fails.
func (q2 *Q2BSPReader) GetSurfEdges() ([]int32, error) {
	return nil, fmt.Errorf("implement me")
}

// GetFaces retrieves all faces from the BSP file, which represent sector geometry and related properties.
func (q2 *Q2BSPReader) GetFaces() ([]*Face, error) {
	return nil, fmt.Errorf("implement me")
}

// GetTexInfos retrieves texture mapping information from the BSP file, including texture vectors, indices, and flags.
func (q2 *Q2BSPReader) GetTexInfos() ([]*TexInfo, error) {
	return nil, fmt.Errorf("implement me")
}

// GetMipTextures retrieves all mipmapped textures from the BSP file, returning them as a slice of MipTexture pointers.
func (q2 *Q2BSPReader) GetMipTextures() ([]*MipTexture, error) {
	return nil, fmt.Errorf("implement me")
}

// GetModels reads and returns a list of sub-models (BModels) from the BSP file or an error if the operation fails.
func (q2 *Q2BSPReader) GetModels() ([]*Model, error) {
	return nil, fmt.Errorf("implement me")
}

func (q2 *Q2BSPReader) GetTextures() *Textures {
	return q2.texManager
}

func (q2 *Q2BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q2.texManager.RegisterPixels(name, width, height, indices, q2.palette, isTransparent, transIndex, invertY)
}
