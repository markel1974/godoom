package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// LumpQ2Entities represents the lump index for Quake 2 entity data.
// LumpQ2Planes represents the lump index for Quake 2 plane data.
// LumpQ2Vertexes represents the lump index for Quake 2 vertex data.
// LumpQ2Visibility represents the lump index for Quake 2 visibility data.
// LumpQ2Nodes represents the lump index for Quake 2 node data.
// LumpQ2TexInfo represents the lump index for Quake 2 texture information.
// LumpQ2Faces represents the lump index for Quake 2 face data.
// LumpQ2Lighting represents the lump index for Quake 2 lightmaps data.
// LumpQ2Leaves represents the lump index for Quake 2 leaf data.
// LumpQ2LeafFaces represents the lump index for Quake 2 leaf faces data.
// LumpQ2LeafBrushes represents the lump index for Quake 2 leaf brushes data.
// LumpQ2Edges represents the lump index for Quake 2 edge data.
// LumpQ2SurfEdges represents the lump index for Quake 2 surface edges data.
// LumpQ2Models represents the lump index for Quake 2 model data.
// LumpQ2Brushes represents the lump index for Quake 2 brush data.
// LumpQ2BrushSides represents the lump index for Quake 2 brush sides data.
// LumpQ2Pop represents the lump index for Quake 2 population visibility data.
// LumpQ2Areas represents the lump index for Quake 2 area definitions.
// LumpQ2AreaPortals represents the lump index for Quake 2 area portal data.
// NumQ2Lumps defines the total number of lump types available in Quake 2.
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

// HeaderQ2 represents the header structure of a Quake II BSP file, containing metadata and lump directory information.
type HeaderQ2 struct {
	Magic   [4]byte
	Version int32
	Lumps   [NumQ2Lumps]struct {
		Offset int32
		Length int32
	}
}

// q2Model represents a Quake 2 model with spatial and structural data stored in BSP format.
type q2Model struct {
	Mins      [3]float32
	Maxs      [3]float32
	Origin    [3]float32
	HeadNode  int32
	FirstFace int32
	NumFaces  int32
}

// q2Face defines the structure of a face in a Quake 2 BSP file.
// It includes information about the plane, edges, texture, lighting, and lightmap.
type q2Face struct {
	PlaneID    uint16
	Side       uint16
	FirstEdge  int32
	NumEdges   uint16
	TexInfo    uint16
	LightTypes [4]uint8
	Lightmap   int32
}

// q2TexInfo represents texture information in a Quake 2 BSP file, including transformation vectors, flags, and texture name.
type q2TexInfo struct {
	Vecs        [2][4]float32
	Flags       uint32
	Value       uint32
	TextureName [32]byte
	NextTexInfo int32
}

// q2Edge represents an edge in a Quake 2 BSP file, defined by two vertex indices V1 and V2.
type q2Edge struct {
	V1, V2 uint16
}

// q2Vertex represents a vertex in 3D space with X, Y, and Z coordinates stored as 32-bit floating-point values.
type q2Vertex struct {
	X, Y, Z float32
}

// Q2BSPReader reads and processes Quake 2 BSP files, handling headers, entities, models, and textures.
type Q2BSPReader struct {
	reader     IReader
	header     HeaderQ2
	rs         io.ReadSeeker
	rsPal      io.ReadSeeker
	palette    []byte
	texManager *Textures
}

// NewQ2BSPReader creates and returns a new Q2BSPReader to handle Quake 2 BSP files with optional palette support.
func NewQ2BSPReader(rs io.ReadSeeker, rsPal io.ReadSeeker) *Q2BSPReader {
	return &Q2BSPReader{
		rs:         rs,
		texManager: NewTextures(),
		rsPal:      rsPal,
	}
}

// Setup initializes the Q2BSPReader, reading the header and optionally loading the palette if available.
func (q2 *Q2BSPReader) Setup(r IReader) error {
	q2.reader = r
	var err error
	if _, err = q2.rs.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err = binary.Read(q2.rs, binary.LittleEndian, &q2.header); err != nil {
		return err
	}

	if string(q2.header.Magic[:]) != "IBSP" {
		return fmt.Errorf("magic number non valido per Quake 2: %s", string(q2.header.Magic[:]))
	}

	// In Quake 2 le texture WAL usano spesso palette dedicate o truecolor,
	// ma carichiamo comunque la palette se fornita dal builder.
	if q2.rsPal != nil {
		q2.palette, err = NewPalette(q2.rsPal)
		if err != nil {
			fmt.Printf("Warning: palette.lmp non caricata in Q2 (le texture WAL la includono): %v\n", err)
		}
	}
	return nil
}

// GetEntities reads and parses the entities lump from the BSP file, returning a slice of Entity pointers or an error.
func (q2 *Q2BSPReader) GetEntities() ([]*Entity, error) {
	lump := q2.header.Lumps[LumpQ2Entities]
	if _, err := q2.rs.Seek(int64(lump.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	data := make([]byte, lump.Length)
	if _, err := q2.rs.Read(data); err != nil {
		return nil, err
	}
	text := FromNullTerminatingString(data)
	return parseEntityText(text)
}

// GetModels retrieves all BSP sub-models (bmodels) from the Quake 2 BSP file or returns an error if not implemented.
func (q2 *Q2BSPReader) GetModels() ([]*Model, error) {
	// Q2 bmodels sono strutturati diversamente, il builder.go li legge internamente per ora.
	return nil, fmt.Errorf("implement me per i sub-models")
}

// RegisterPixels registers texture pixel data with the texture manager, applying palette and optional transformations.
func (q2 *Q2BSPReader) RegisterPixels(name string, width, height int, indices []byte, isTransparent bool, transIndex byte, invertY bool) error {
	return q2.texManager.RegisterPixels(name, width, height, indices, q2.palette, isTransparent, transIndex, invertY)
}

// GetRawFaces extracts raw face data for a specific model index from the Quake 2 BSP file and returns the corresponding faces.
func (q2 *Q2BSPReader) GetRawFaces(modelIdx int) ([]*RawFace, error) {
	// 1. Lettura dei Modelli per trovare l'offset delle facce
	lumpModels := q2.header.Lumps[LumpQ2Models]
	if _, err := q2.rs.Seek(int64(lumpModels.Offset), io.SeekStart); err != nil {
		return nil, err
	}
	numModels := int(lumpModels.Length) / 48
	models := make([]q2Model, numModels)
	if err := binary.Read(q2.rs, binary.LittleEndian, &models); err != nil {
		return nil, err
	}

	if modelIdx < 0 || modelIdx >= numModels {
		return nil, fmt.Errorf("modelIdx %d fuori dai limiti (0-%d)", modelIdx, numModels-1)
	}
	targetModel := models[modelIdx]

	// 2. Lettura massiva dei lumps topologici
	lumpFaces := q2.header.Lumps[LumpQ2Faces]
	q2.rs.Seek(int64(lumpFaces.Offset), io.SeekStart)
	faces := make([]q2Face, int(lumpFaces.Length)/20)
	binary.Read(q2.rs, binary.LittleEndian, &faces)

	lumpTexInfos := q2.header.Lumps[LumpQ2TexInfo]
	q2.rs.Seek(int64(lumpTexInfos.Offset), io.SeekStart)
	texInfos := make([]q2TexInfo, int(lumpTexInfos.Length)/76)
	binary.Read(q2.rs, binary.LittleEndian, &texInfos)

	lumpSurfEdges := q2.header.Lumps[LumpQ2SurfEdges]
	q2.rs.Seek(int64(lumpSurfEdges.Offset), io.SeekStart)
	surfEdges := make([]int32, int(lumpSurfEdges.Length)/4)
	binary.Read(q2.rs, binary.LittleEndian, &surfEdges)

	lumpEdges := q2.header.Lumps[LumpQ2Edges]
	q2.rs.Seek(int64(lumpEdges.Offset), io.SeekStart)
	edges := make([]q2Edge, int(lumpEdges.Length)/4)
	binary.Read(q2.rs, binary.LittleEndian, &edges)

	lumpVerts := q2.header.Lumps[LumpQ2Vertexes]
	q2.rs.Seek(int64(lumpVerts.Offset), io.SeekStart)
	vertexes := make([]q2Vertex, int(lumpVerts.Length)/12)
	binary.Read(q2.rs, binary.LittleEndian, &vertexes)

	// 3. Risoluzione dell'indirezione e generazione dei RawFace
	var rawFaces []*RawFace
	for i := int32(0); i < targetModel.NumFaces; i++ {
		faceIdx := targetModel.FirstFace + i
		face := faces[faceIdx]
		texInfo := texInfos[face.TexInfo]

		// Decodifica il nome della texture (array fisso di 32 byte null-terminated in C)
		texNameBytes := make([]byte, 0, 32)
		for _, b := range texInfo.TextureName {
			if b == 0 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.ToLower(string(texNameBytes))

		// In Quake 2 i flag sono inclusi nel TexInfo. SURF_SKY è il bitmask 0x4
		isSky := (texInfo.Flags & 0x4) != 0

		// Risoluzione SurfEdge -> Edge -> Vertex
		var points []geometry.XYZ
		for j := uint16(0); j < face.NumEdges; j++ {
			surfEdgeIdx := surfEdges[face.FirstEdge+int32(j)]

			var v q2Vertex
			if surfEdgeIdx >= 0 {
				v = vertexes[edges[surfEdgeIdx].V1] // Direzione positiva (Senso antiorario)
			} else {
				v = vertexes[edges[-surfEdgeIdx].V2] // Direzione invertita (Senso orario)
			}

			// Applichiamo la trasformazione degli assi standard z-up usando CreateXYZ
			points = append(points, CreateXYZ(float64(v.X), float64(v.Y), float64(v.Z)))
		}

		rawFaces = append(rawFaces, &RawFace{
			Points:  points,
			TexName: texName,
			IsSky:   isSky,
		})
	}

	q2.compileTextures(rawFaces)
	return rawFaces, nil
}

// GetTextures retrieves the texture manager associated with the Q2BSPReader.
func (q2 *Q2BSPReader) GetTextures() *Textures {
	return q2.texManager
}

// PrepareTexture loads and registers unique textures from a PAK file based on the provided faces, excluding sky textures.
func (q2 *Q2BSPReader) compileTextures(faces []*RawFace) {
	uniqueTextures := make(map[string]bool)
	for _, f := range faces {
		uniqueTextures[f.TexName] = true
	}
	for texName := range uniqueTextures {
		if texName == "sky" || len(texName) == 0 {
			continue
		}
		walPath := "textures" + PakSeparator + texName + ".wal"
		walFile, walErr := q2.reader.Open(walPath)
		if walErr != nil {
			fmt.Printf("Warning: missing asset %s: %s\n \n", walPath, walErr.Error())
			continue
		}

		walTex, walErr := ParseWal(walFile)
		if walErr != nil {
			fmt.Printf("Warning: can't open asset %s: %s\n", walPath, walErr.Error())
			continue
		}

		err := q2.RegisterPixels(texName, int(walTex.Header.Width), int(walTex.Header.Height), walTex.Pixels, false, 255, false)
		if err != nil {
			fmt.Printf("Warning: can't register asset %s: %s\n", walPath, err.Error())
			continue
		}
	}
}
