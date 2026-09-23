package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const MD3Magic = 860898377 // "IDP3"
const MD3Version = 15

type MD3Header struct {
	Magic       int32
	Version     int32
	Name        [68]byte
	NumFrames   int32
	NumTags     int32
	NumSurfaces int32
	NumSkins    int32
	OfsFrames   int32
	OfsTags     int32
	OfsSurfaces int32
	OfsEOF      int32
}

type MD3Frame struct {
	Mins        [3]float32
	Maxs        [3]float32
	LocalOrigin [3]float32
	Radius      float32
	Name        [16]byte
}

type MD3Tag struct {
	Name   [64]byte
	Origin [3]float32
	Axis   [3][3]float32
}

type MD3SurfaceHeader struct {
	Magic        int32 // "IDP3"
	Name         [64]byte
	Flags        int32
	NumFrames    int32
	NumShaders   int32
	NumVerts     int32
	NumTriangles int32
	OfsTriangles int32
	OfsShaders   int32
	OfsSt        int32
	OfsXYZNormal int32
	OfsEnd       int32
}

type MD3Shader struct {
	Name  [64]byte
	Index int32
}

type MD3Triangle struct {
	Indexes [3]int32
}

type MD3TexCoord struct {
	St [2]float32
}

type MD3Vertex struct {
	Coord  [3]int16
	Normal [2]uint8
}

type MD3Resource struct{}

func NewMD3Resource() *MD3Resource {
	return &MD3Resource{}
}

func (m *MD3Resource) Parse(rs io.ReadSeeker, texManager *Textures, basePath string, skinMap map[string]string) (*config.MD1, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var header MD3Header
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	if header.Magic != MD3Magic || header.Version != MD3Version {
		return nil, fmt.Errorf("formato MD3 non valido: magic %d, version %d", header.Magic, header.Version)
	}

	// Leggi i frame names per l'animazione
	if _, err := rs.Seek(int64(header.OfsFrames), io.SeekStart); err != nil {
		return nil, err
	}
	frames := make([]MD3Frame, header.NumFrames)
	if err := binary.Read(rs, binary.LittleEndian, &frames); err != nil {
		return nil, err
	}

	frameNames := make([]string, header.NumFrames)
	for i, f := range frames {
		frameNames[i] = strings.TrimRight(string(f.Name[:]), "\x00")
	}

	// Inizializza il contenitore generico config.MD1 (usato dall'engine come frame array)
	cfg := config.NewMD1(int(header.NumFrames), frameNames)

	// Leggi i tags
	if header.NumTags > 0 && header.OfsTags > 0 {
		if _, err := rs.Seek(int64(header.OfsTags), io.SeekStart); err != nil {
			return nil, err
		}
		numTotalTags := int(header.NumFrames * header.NumTags)
		tags := make([]MD3Tag, numTotalTags)
		if err := binary.Read(rs, binary.LittleEndian, &tags); err != nil {
			return nil, err
		}
		for i := 0; i < int(header.NumFrames); i++ {
			for j := 0; j < int(header.NumTags); j++ {
				tag := tags[i*int(header.NumTags)+j]
				tagName := strings.TrimRight(string(tag.Name[:]), "\x00")
				// MD3 tags don't seem to be scaled by 1/64, but let's check later, wait, MD3 tags coordinates are float32, so no md3Scale needed!
				cfg.Frames[i].Tags[tagName] = geometry.XYZ{
					X: float64(tag.Origin[0]),
					Y: float64(tag.Origin[1]),
					Z: float64(tag.Origin[2]),
				}
			}
		}
	}

	// Per scalare i vertici MD3 (che sono short int) a float
	const md3Scale = 1.0 / 64.0

	// Spostati all'inizio delle superfici
	offsetSurf := int64(header.OfsSurfaces)

	for s := 0; s < int(header.NumSurfaces); s++ {
		if _, err := rs.Seek(offsetSurf, io.SeekStart); err != nil {
			return nil, err
		}

		var surfHeader MD3SurfaceHeader
		if err := binary.Read(rs, binary.LittleEndian, &surfHeader); err != nil {
			return nil, err
		}

		// Shaders (Materiali)
		rs.Seek(offsetSurf+int64(surfHeader.OfsShaders), io.SeekStart)
		shaders := make([]MD3Shader, surfHeader.NumShaders)
		binary.Read(rs, binary.LittleEndian, &shaders)

		// Trova il materiale (usiamo il primo shader come materiale base)
		var material *config.Material
		surfName := strings.TrimRight(string(surfHeader.Name[:]), "\x00")
		var shaderName string

		if skinMap != nil {
			if texPath, ok := skinMap[surfName]; ok {
				shaderName = texPath
			}
		}

		if len(shaderName) == 0 && len(shaders) > 0 {
			shaderName = strings.TrimRight(string(shaders[0].Name[:]), "\x00")
			shaderName = strings.ReplaceAll(shaderName, "\\", "/")
			if len(shaderName) > 0 && !strings.Contains(shaderName, "/") {
				shaderName = basePath + shaderName
			}
		}

		if len(shaderName) > 0 {
			material = config.NewConfigMaterial([]string{shaderName}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
		}

		// Triangoli
		rs.Seek(offsetSurf+int64(surfHeader.OfsTriangles), io.SeekStart)
		triangles := make([]MD3Triangle, surfHeader.NumTriangles)
		binary.Read(rs, binary.LittleEndian, &triangles)

		// TexCoords
		rs.Seek(offsetSurf+int64(surfHeader.OfsSt), io.SeekStart)
		texCoords := make([]MD3TexCoord, surfHeader.NumVerts)
		binary.Read(rs, binary.LittleEndian, &texCoords)

		// Vertici XYZ
		rs.Seek(offsetSurf+int64(surfHeader.OfsXYZNormal), io.SeekStart)
		numTotalVerts := int(surfHeader.NumFrames * surfHeader.NumVerts)
		vertices := make([]MD3Vertex, numTotalVerts)
		binary.Read(rs, binary.LittleEndian, &vertices)

		// Assembla i triangoli per ogni frame
		for i := 0; i < int(surfHeader.NumFrames); i++ {
			for t := 0; t < int(surfHeader.NumTriangles); t++ {
				var configTri config.MD1Triangle
				configTri.Material = material

				for k := 0; k < 3; k++ {
					vIndex := triangles[t].Indexes[k]

					// L'array 'vertices' contiene i vertici di tutti i frame concatenati
					globVIndex := (i * int(surfHeader.NumVerts)) + int(vIndex)
					v := vertices[globVIndex]
					uv := texCoords[vIndex]

					configTri.Vertices[k] = config.MD1Vertex{
						Pos: geometry.XYZ{
							X: float64(v.Coord[0]) * md3Scale,
							Y: float64(v.Coord[1]) * md3Scale,
							Z: float64(v.Coord[2]) * md3Scale,
						},
						U: uv.St[0],
						V: uv.St[1],
					}
				}
				cfg.Frames[i].Triangles = append(cfg.Frames[i].Triangles, configTri)
			}
		}

		// Avanza alla superficie successiva
		offsetSurf += int64(surfHeader.OfsEnd)
	}

	return cfg, nil
}
