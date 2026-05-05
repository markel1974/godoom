package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
)

const MD1Magic = 1330660425 // MDLMagic is a constant representing the "magic number" identifier for MDL file headers in Quake formats.
// MD1Version represents the version number of the MDL file format used for compatibility checks.
const MD1Version = 6

// MD1Header represents the header structure of a Quake MDL (3D model) file, containing metadata and configuration fields.
type MD1Header struct {
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

// MD1Skin represents a skin used in the 3D model, identifying the group type and the palette indices stored in Data.
type MD1Skin struct {
	Group int32
	Data  []byte // Indici della palette
}

// MD1TexCoord represents a texture coordinate in the MDL model format.
// It contains S and T values, and specifies if it is located on a seam.
type MD1TexCoord struct {
	OnSeam int32
	S      int32
	T      int32
}

// MD1Triangle represents a triangular face in a 3D model with vertex indices and a front-facing flag.
type MD1Triangle struct {
	FacesFront int32
	Vertices   [3]int32
}

// MD1Vertex represents a single vertex in an MDL model, including its position and associated normal vector index.
type MD1Vertex struct {
	V           [3]uint8
	NormalIndex uint8
}

// MD1Resource represents a 3D model resource containing header, skins, texture coordinates, triangles, and decompressed frames.
type MD1Resource struct {
	Header     *MD1Header
	Skins      []*MD1Skin
	TexCoords  []*MD1TexCoord
	Triangles  []*MD1Triangle
	Frames     [][][3]float64
	FrameNames []string
}

func NewMD1Resource() *MD1Resource {
	return &MD1Resource{}
}

// Parse parse a Quake MD1 3D model from the given io.ReadSeeker and returns a fully populated MD1Resource.
func (md1 *MD1Resource) Parse(rs io.ReadSeeker) error {
	var header MD1Header
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	if header.Magic != MD1Magic || header.Version != MD1Version {
		return fmt.Errorf("formato MDL non valido: magic %d, version %d", header.Magic, header.Version)
	}

	md1.Header = &header
	md1.Skins = make([]*MD1Skin, header.NumSkins)
	md1.TexCoords = make([]*MD1TexCoord, header.NumVerts)
	md1.Triangles = make([]*MD1Triangle, header.NumTris)
	// Pre-allochiamo a 0 per i frame, perché i Group Frame espanderanno l'array oltre NumFrames
	md1.Frames = make([][][3]float64, 0, header.NumFrames)
	md1.FrameNames = make([]string, 0, header.NumFrames)

	// 1. Lettura Skin (Supporto per Group Skins)
	for i := int32(0); i < header.NumSkins; i++ {
		var group int32
		if err := binary.Read(rs, binary.LittleEndian, &group); err != nil {
			return fmt.Errorf("failed to read skin group flag: %w", err)
		}

		if group == 0 {
			// Single Skin
			size := header.SkinWidth * header.SkinHeight
			data := make([]byte, size)
			if err := binary.Read(rs, binary.LittleEndian, data); err != nil {
				return fmt.Errorf("failed to read skin data: %w", err)
			}
			md1.Skins[i] = &MD1Skin{Group: group, Data: data}
		} else {
			// Group Skin (usate tipicamente per animazioni di liquidi o torce)
			var numGroupSkins int32
			if err := binary.Read(rs, binary.LittleEndian, &numGroupSkins); err != nil {
				return fmt.Errorf("failed to read group skins count: %w", err)
			}
			// Lettura e scarto dei timestamp di animazione del gruppo
			times := make([]float32, numGroupSkins)
			if err := binary.Read(rs, binary.LittleEndian, times); err != nil {
				return fmt.Errorf("failed to read group skin times: %w", err)
			}
			// Salviamo solo il primo frame della skin animata per il nostro Builder,
			// ma dobbiamo scorrere i byte di tutte le skin per non disallineare lo stream.
			var firstSkinData []byte
			size := header.SkinWidth * header.SkinHeight
			for j := int32(0); j < numGroupSkins; j++ {
				data := make([]byte, size)
				if err := binary.Read(rs, binary.LittleEndian, data); err != nil {
					return fmt.Errorf("failed to read group skin data %d: %w", j, err)
				}
				if j == 0 {
					firstSkinData = data
				}
			}
			md1.Skins[i] = &MD1Skin{Group: group, Data: firstSkinData}
		}
	}

	// 2. Lettura TexCoords (ST)
	stArray := make([]MD1TexCoord, header.NumVerts)
	if err := binary.Read(rs, binary.LittleEndian, stArray); err != nil {
		return fmt.Errorf("failed to read tex coords: %w", err)
	}
	for i := range stArray {
		md1.TexCoords[i] = &stArray[i]
	}

	// 3. Lettura Triangoli (Topologia)
	triArray := make([]MD1Triangle, header.NumTris)
	if err := binary.Read(rs, binary.LittleEndian, triArray); err != nil {
		return fmt.Errorf("failed to read triangles: %w", err)
	}
	for i := range triArray {
		md1.Triangles[i] = &triArray[i]
	}

	// 4. Lettura Frames (Supporto per Group Frames)
	for i := int32(0); i < header.NumFrames; i++ {
		var group int32
		if err := binary.Read(rs, binary.LittleEndian, &group); err != nil {
			return fmt.Errorf("failed to read frame group flag: %w", err)
		}

		if group == 0 {
			// Single Frame
			var bboxMin, bboxMax MD1Vertex
			var name [16]byte
			if err := binary.Read(rs, binary.LittleEndian, &bboxMin); err != nil {
				return err
			}
			if err := binary.Read(rs, binary.LittleEndian, &bboxMax); err != nil {
				return err
			}
			if err := binary.Read(rs, binary.LittleEndian, &name); err != nil {
				return err
			}

			pVerts := make([]MD1Vertex, header.NumVerts)
			if err := binary.Read(rs, binary.LittleEndian, pVerts); err != nil {
				return err
			}

			frameVerts := md1.processVertices(pVerts, header)
			md1.Frames = append(md1.Frames, frameVerts)
			md1.FrameNames = append(md1.FrameNames, FromNullTerminatingString(name[:]))
		} else {
			// Group Frame (Animazione raggruppata)
			var numGroupFrames int32
			if err := binary.Read(rs, binary.LittleEndian, &numGroupFrames); err != nil {
				return err
			}

			// Bounding box generale dell'intero gruppo (ignorato ai fini geometrici)
			var groupBboxMin, groupBboxMax MD1Vertex
			if err := binary.Read(rs, binary.LittleEndian, &groupBboxMin); err != nil {
				return err
			}
			if err := binary.Read(rs, binary.LittleEndian, &groupBboxMax); err != nil {
				return err
			}

			// Timestamp del gruppo
			times := make([]float32, numGroupFrames)
			if err := binary.Read(rs, binary.LittleEndian, times); err != nil {
				return err
			}

			// "Spalmiamo" tutti i sub-frame del gruppo nell'array principale dei frame
			for j := int32(0); j < numGroupFrames; j++ {
				var bboxMin, bboxMax MD1Vertex
				var name [16]byte
				if err := binary.Read(rs, binary.LittleEndian, &bboxMin); err != nil {
					return err
				}
				if err := binary.Read(rs, binary.LittleEndian, &bboxMax); err != nil {
					return err
				}
				if err := binary.Read(rs, binary.LittleEndian, &name); err != nil {
					return err
				}

				pVerts := make([]MD1Vertex, header.NumVerts)
				if err := binary.Read(rs, binary.LittleEndian, pVerts); err != nil {
					return err
				}

				frameVerts := md1.processVertices(pVerts, header)
				md1.Frames = append(md1.Frames, frameVerts)
				md1.FrameNames = append(md1.FrameNames, FromNullTerminatingString(name[:]))
			}
		}
	}

	return nil
}

// processVertices transforms MD1Vertex array positions into scaled and translated [3]float64 coordinates using header data.
func (md1 *MD1Resource) processVertices(pVerts []MD1Vertex, header MD1Header) [][3]float64 {
	frameVerts := make([][3]float64, header.NumVerts)
	for vIdx, v := range pVerts {
		x := (float64(v.V[0]) * float64(header.Scale[0])) + float64(header.Translate[0])
		y := (float64(v.V[1]) * float64(header.Scale[1])) + float64(header.Translate[1])
		z := (float64(v.V[2]) * float64(header.Scale[2])) + float64(header.Translate[2])
		frameVerts[vIdx] = [3]float64{x, y, z}
	}
	return frameVerts
}
