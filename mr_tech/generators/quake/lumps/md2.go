package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
)

const MD2Magic = 844121161 // "IDP2"
const MD2Version = 8

// MD2Header represents the header structure of an MD2 model file, providing metadata and offsets for the file format.
type MD2Header struct {
	Magic        int32
	Version      int32
	SkinWidth    int32
	SkinHeight   int32
	FrameSize    int32
	NumSkins     int32
	NumVertices  int32
	NumST        int32
	NumTris      int32
	NumGLCmds    int32
	NumFrames    int32
	OffsetSkins  int32
	OffsetST     int32
	OffsetTris   int32
	OffsetFrames int32
	OffsetGLCmds int32
	OffsetEnd    int32
}

// MD2Triangle represents a single triangle in an MD2 model, containing vertex and texture coordinate indices.
type MD2Triangle struct {
	VertexIndices [3]uint16
	STIndices     [3]uint16
}

// MD2ST represents a structure with two int16 fields: S and T, typically used for storing 2D texture coordinates.
type MD2ST struct {
	S int16
	T int16
}

// MD2Vertex represents a single vertex in the MD2 model, storing its position and a normal vector index.
type MD2Vertex struct {
	V           [3]uint8
	NormalIndex uint8
}

// MD2Triangles represents a collection of MD2Triangle elements.
type MD2Triangles struct {
	Triangles []*MD2Triangle
}

// NewMD2Triangles creates and returns a new instance of MD2Triangles.
func NewMD2Triangles() *MD2Triangles {
	return &MD2Triangles{}
}

// Parse reads MD2 triangle data from the provided io.ReadSeeker.
func (md2 *MD2Triangles) Parse(rs io.ReadSeeker, offset int32, count int32) error {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	triArray := make([]MD2Triangle, count)
	if err := binary.Read(rs, binary.LittleEndian, triArray); err != nil {
		return err
	}
	md2.Triangles = make([]*MD2Triangle, count)
	for i := range triArray {
		md2.Triangles[i] = &triArray[i]
	}
	return nil
}

// MD2STS represents a collection of pointers to MD2ST objects.
type MD2STS struct {
	STS []*MD2ST
}

// NewMD2STS creates and returns a pointer to an initialized MD2STS instance.
func NewMD2STS() *MD2STS {
	return &MD2STS{}
}

// Parse reads and populates texture coordinate data for MD2 model.
func (md2 *MD2STS) Parse(rs io.ReadSeeker, offset int32, count int32) error {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	stArray := make([]MD2ST, count)
	if err := binary.Read(rs, binary.LittleEndian, stArray); err != nil {
		return err
	}
	md2.STS = make([]*MD2ST, count)
	for i := range stArray {
		md2.STS[i] = &stArray[i]
	}
	return nil
}

// MD2FrameHeader represents the metadata prefixing each frame's vertex data in an MD2 file.
type MD2FrameHeader struct {
	Scale     [3]float32
	Translate [3]float32
	Name      [16]byte
}

// MD2Frames represents a collection of decompressed animation frames for an MD2 model.
type MD2Frames struct {
	Frames     [][][3]float64
	FrameNames []string
}

// NewMD2Frames creates and returns an initialized MD2Frames instance.
func NewMD2Frames() *MD2Frames {
	return &MD2Frames{}
}

// Parse reads frame metadata and decompresses the 8-bit vertex coordinates into 64-bit float coordinates using per-frame scale and translation.
func (md2 *MD2Frames) Parse(rs io.ReadSeeker, offset int32, numFrames int32, numVertices int32) error {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	md2.Frames = make([][][3]float64, numFrames)
	md2.FrameNames = make([]string, numFrames)
	// In MD2, Scale e Translate cambiano ad ogni singolo frame per massimizzare la precisione degli uint8.
	for i := int32(0); i < numFrames; i++ {
		var fHeader MD2FrameHeader
		if err := binary.Read(rs, binary.LittleEndian, &fHeader); err != nil {
			return err
		}
		pVerts := make([]MD2Vertex, numVertices)
		if err := binary.Read(rs, binary.LittleEndian, pVerts); err != nil {
			return err
		}
		frameVerts := make([][3]float64, numVertices)
		for vIdx, v := range pVerts {
			x := (float64(v.V[0]) * float64(fHeader.Scale[0])) + float64(fHeader.Translate[0])
			y := (float64(v.V[1]) * float64(fHeader.Scale[1])) + float64(fHeader.Translate[1])
			z := (float64(v.V[2]) * float64(fHeader.Scale[2])) + float64(fHeader.Translate[2])
			frameVerts[vIdx] = [3]float64{x, y, z}
		}
		md2.Frames[i] = frameVerts
		md2.FrameNames[i] = FromNullTerminatingString(fHeader.Name[:])
	}
	return nil
}

// MD2Skins represents a collection of texture paths associated with the MD2 model.
type MD2Skins struct {
	Names []string
}

// NewMD2Skins creates and returns an initialized MD2Skins instance.
func NewMD2Skins() *MD2Skins {
	return &MD2Skins{}
}

// Parse extracts the 64-byte null-terminated strings representing the texture/skin paths.
func (s *MD2Skins) Parse(rs io.ReadSeeker, offset int32, numSkins int32) error {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return err
	}
	names := make([][64]byte, numSkins)
	if err := binary.Read(rs, binary.LittleEndian, names); err != nil {
		return err
	}
	s.Names = make([]string, numSkins)
	for i, n := range names {
		s.Names[i] = FromNullTerminatingString(n[:])
	}
	return nil
}

// MD2Resource is the main orchestrator for the Quake 2 MD2 model format, bundling all sub-components.
type MD2Resource struct {
	Header    *MD2Header
	Skins     *MD2Skins
	TexCoords *MD2STS
	Triangles *MD2Triangles
	Frames    *MD2Frames
}

// NewMD2Resource creates an empty orchestrator for an MD2 model.
func NewMD2Resource() *MD2Resource {
	return &MD2Resource{
		Skins:     NewMD2Skins(),
		TexCoords: NewMD2STS(),
		Triangles: NewMD2Triangles(),
		Frames:    NewMD2Frames(),
	}
}

// Parse reads the file header and delegates the parsing to the respective sub-components via offsets.
func (md2 *MD2Resource) Parse(rs io.ReadSeeker) error {
	var header MD2Header
	if err := binary.Read(rs, binary.LittleEndian, &header); err != nil {
		return err
	}
	if header.Magic != MD2Magic || header.Version != MD2Version {
		return fmt.Errorf("formato MD2 non valido: magic %d, version %d", header.Magic, header.Version)
	}
	md2.Header = &header
	if err := md2.Skins.Parse(rs, header.OffsetSkins, header.NumSkins); err != nil {
		return fmt.Errorf("failed to parse MD2 skins: %w", err)
	}
	if err := md2.TexCoords.Parse(rs, header.OffsetST, header.NumST); err != nil {
		return fmt.Errorf("failed to parse MD2 tex coords: %w", err)
	}
	if err := md2.Triangles.Parse(rs, header.OffsetTris, header.NumTris); err != nil {
		return fmt.Errorf("failed to parse MD2 triangles: %w", err)
	}
	if err := md2.Frames.Parse(rs, header.OffsetFrames, header.NumFrames, header.NumVertices); err != nil {
		return fmt.Errorf("failed to parse MD2 frames: %w", err)
	}
	return nil
}
