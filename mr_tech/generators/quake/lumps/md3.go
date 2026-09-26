package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
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

type MD3Surface struct {
	Header    MD3SurfaceHeader
	Vertices  []MD3Vertex
	Triangles []MD3Triangle
	Shaders   []MD3Shader
	TexCoords []MD3TexCoord
}

type MD3Resource struct {
	Header     *MD3Header
	Frames     []MD3Frame
	FrameNames []string
	Tags       []MD3Tag
	Surfaces   []MD3Surface
}

func NewMD3Resource() *MD3Resource {
	return &MD3Resource{}
}

func (m *MD3Resource) Parse(rs io.ReadSeeker) (*MD3Resource, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var header3 MD3Header
	if err := binary.Read(rs, binary.LittleEndian, &header3); err != nil {
		return nil, err
	}

	if header3.Magic != MD3Magic || header3.Version != MD3Version {
		return nil, fmt.Errorf("formato MD3 non valido: magic %d, version %d", header3.Magic, header3.Version)
	}

	// Leggi i frame names per l'animazione
	if _, err := rs.Seek(int64(header3.OfsFrames), io.SeekStart); err != nil {
		return nil, err
	}

	res := NewMD3Resource()
	res.Header = &header3
	res.Frames = make([]MD3Frame, header3.NumFrames)
	if err := binary.Read(rs, binary.LittleEndian, &res.Frames); err != nil {
		return nil, err
	}
	res.FrameNames = make([]string, header3.NumFrames)
	for i, f := range res.Frames {
		res.FrameNames[i] = strings.TrimRight(string(f.Name[:]), "\x00")
	}

	if res.Header.NumTags > 0 && res.Header.OfsTags > 0 {
		if _, err := rs.Seek(int64(res.Header.OfsTags), io.SeekStart); err != nil {
			return nil, err
		}
		numTotalTags := int(res.Header.NumFrames * res.Header.NumTags)
		res.Tags = make([]MD3Tag, numTotalTags)
		if err := binary.Read(rs, binary.LittleEndian, &res.Tags); err != nil {
			return nil, err
		}
	}

	offsetSurf := int64(res.Header.OfsSurfaces)

	for s := 0; s < int(res.Header.NumSurfaces); s++ {
		var surf MD3Surface
		if _, err := rs.Seek(offsetSurf, io.SeekStart); err != nil {
			return nil, err
		}
		if err := binary.Read(rs, binary.LittleEndian, &surf.Header); err != nil {
			return nil, err
		}
		// Shaders (Materiali)
		if _, err := rs.Seek(offsetSurf+int64(surf.Header.OfsShaders), io.SeekStart); err != nil {
			return nil, err
		}
		surf.Shaders = make([]MD3Shader, surf.Header.NumShaders)
		if err := binary.Read(rs, binary.LittleEndian, &surf.Shaders); err != nil {
			return nil, err
		}
		// Triangoli
		if _, err := rs.Seek(offsetSurf+int64(surf.Header.OfsTriangles), io.SeekStart); err != nil {
			return nil, err
		}
		surf.Triangles = make([]MD3Triangle, surf.Header.NumTriangles)
		if err := binary.Read(rs, binary.LittleEndian, &surf.Triangles); err != nil {
			return nil, err
		}
		// TexCoords
		if _, err := rs.Seek(offsetSurf+int64(surf.Header.OfsSt), io.SeekStart); err != nil {
			return nil, err
		}
		surf.TexCoords = make([]MD3TexCoord, surf.Header.NumVerts)
		if err := binary.Read(rs, binary.LittleEndian, &surf.TexCoords); err != nil {
			return nil, err
		}
		// Vertici XYZ
		if _, err := rs.Seek(offsetSurf+int64(surf.Header.OfsXYZNormal), io.SeekStart); err != nil {
			return nil, err
		}
		numTotalVerts := int(surf.Header.NumFrames * surf.Header.NumVerts)
		surf.Vertices = make([]MD3Vertex, numTotalVerts)
		if err := binary.Read(rs, binary.LittleEndian, &surf.Vertices); err != nil {
			return nil, err
		}
		// Avanza alla superficie successiva
		offsetSurf += int64(surf.Header.OfsEnd)
		res.Surfaces = append(res.Surfaces, surf)
	}

	return res, nil
}
