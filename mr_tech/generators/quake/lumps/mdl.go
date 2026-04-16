package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
)

const MDLMagic = 1330660425 // MDLMagic is a constant representing the "magic number" identifier for MDL file headers in Quake formats.
// MDLVersion represents the version number of the MDL file format used for compatibility checks.
const MDLVersion = 6

// MDLHeader represents the header structure of a Quake MDL (3D model) file, containing metadata and configuration fields.
type MDLHeader struct {
	Magic          int32
	Version        int32
	Scale          [3]float32
	Translate      [3]float32
	BoundingRadius float32
	EyePosition    [3]float32
	NumSkins       int32
	SkinWidth      int32
	SkinHeight     int32
	NumVerts       int32
	NumTris        int32
	NumFrames      int32
	Synctype       int32
	Flags          int32
	Size           float32
}

// MDLSkin represents a skin used in the 3D model, identifying the group type and the palette indices stored in Data.
type MDLSkin struct {
	Group int32
	Data  []byte // Indici della palette
}

// MDLTexCoord represents a texture coordinate in the MDL model format.
// It contains S and T values, and specifies if it is located on a seam.
type MDLTexCoord struct {
	OnSeam int32
	S      int32
	T      int32
}

// MDLTriangle represents a triangular face in a 3D model with vertex indices and a front-facing flag.
type MDLTriangle struct {
	FacesFront int32
	Vertices   [3]int32
}

// MDLVertex represents a single vertex in an MDL model, including its position and associated normal vector index.
type MDLVertex struct {
	V           [3]uint8
	NormalIndex uint8
}

// Model3DResource represents a 3D model resource containing header, skins, texture coordinates, triangles, and decompressed frames.
type Model3DResource struct {
	Header    *MDLHeader
	Skins     []*MDLSkin
	TexCoords []*MDLTexCoord
	Triangles []*MDLTriangle
	// I frame conterranno i vertici già decompressi e convertiti in float64
	Frames [][][3]float64
}

// NewMDLResource reads a Quake MDL 3D model from the given io.ReadSeeker and returns a fully populated Model3DResource.
func NewMDLResource(rs io.ReadSeeker) (*Model3DResource, error) {
	var header MDLHeader
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Magic != MDLMagic || header.Version != MDLVersion {
		return nil, fmt.Errorf("formato MDL non valido")
	}

	resource := &Model3DResource{
		Header:    &header,
		Skins:     make([]*MDLSkin, header.NumSkins),
		TexCoords: make([]*MDLTexCoord, header.NumVerts),
		Triangles: make([]*MDLTriangle, header.NumTris),
		Frames:    make([][][3]float64, header.NumFrames),
	}

	// 1. Lettura Skin (assumendo single skin per semplicità, Quake supporta group skins)
	for i := int32(0); i < header.NumSkins; i++ {
		var group int32
		binary.Read(rs, binary.LittleEndian, &group)
		// Le skin di Quake sono array di indici palette larghi (SkinWidth * SkinHeight)
		data := make([]byte, header.SkinWidth*header.SkinHeight)
		binary.Read(rs, binary.LittleEndian, data)
		resource.Skins[i] = &MDLSkin{Group: group, Data: data}
	}

	// 2. Lettura TexCoords (ST)
	for i := int32(0); i < header.NumVerts; i++ {
		var st MDLTexCoord
		binary.Read(rs, binary.LittleEndian, &st)
		resource.TexCoords[i] = &st
	}

	// 3. Lettura Triangoli (Topologia)
	for i := int32(0); i < header.NumTris; i++ {
		var tri MDLTriangle
		binary.Read(rs, binary.LittleEndian, &tri)
		resource.Triangles[i] = &tri
	}

	// 4. Lettura Frames (Vertex Morphing)
	for i := int32(0); i < header.NumFrames; i++ {
		var group int32 // Simple frame (0) o Group frame (non 0)
		binary.Read(rs, binary.LittleEndian, &group)

		// Bounding Box del frame
		var bboxMin, bboxMax MDLVertex
		var name [16]byte
		binary.Read(rs, binary.LittleEndian, &bboxMin)
		binary.Read(rs, binary.LittleEndian, &bboxMax)
		binary.Read(rs, binary.LittleEndian, &name)

		pVerts := make([]MDLVertex, header.NumVerts)
		binary.Read(rs, binary.LittleEndian, pVerts)

		// Decompressione spaziale immediata
		frameVerts := make([][3]float64, header.NumVerts)
		for vIdx, v := range pVerts {
			// X = (v.x * scale.x) + translate.x
			x := (float64(v.V[0]) * float64(header.Scale[0])) + float64(header.Translate[0])
			y := (float64(v.V[1]) * float64(header.Scale[1])) + float64(header.Translate[1])
			z := (float64(v.V[2]) * float64(header.Scale[2])) + float64(header.Translate[2])

			// Applichiamo la conversione Z-up (Quake Z-up -> Engine Z-up)
			// Attenzione: Quake 1 MDL usa un sistema di assi spesso strano,
			// generalmente richiede uno swap Y/Z o negazioni.
			frameVerts[vIdx] = [3]float64{x, y, z}
		}
		resource.Frames[i] = frameVerts
	}

	return resource, nil
}
