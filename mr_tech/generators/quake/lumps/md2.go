package lumps

import (
	"encoding/binary"
	"io"
)

// MD2Header represents the header information of an MD1 file format used in 3D models for Quake II game engine.
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

// MD2Triangle represents a triangular face in an MD1 model, storing vertex and texture coordinate indices.
type MD2Triangle struct {
	VertexIndices [3]uint16
	STIndices     [3]uint16
}

// MD2ST represents a structure containing two 16-bit signed integer components S and T used typically for storing values.
type MD2ST struct {
	S int16
	T int16
}

// MD2Vertex represents a single vertex in the MD1 model format, including position and normal index.
type MD2Vertex struct {
	V           [3]uint8
	NormalIndex uint8
}

// NewMD2Triangles reads and constructs a slice of MD2Triangle pointers based on the offset and count from the given io.ReadSeeker.
func NewMD2Triangles(rs io.ReadSeeker, offset int32, count int32) ([]*MD2Triangle, error) {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	pTris := make([]MD2Triangle, count)
	if err := binary.Read(rs, binary.LittleEndian, pTris); err != nil {
		return nil, err
	}

	tris := make([]*MD2Triangle, count)
	for idx, t := range pTris {
		tris[idx] = &MD2Triangle{
			VertexIndices: t.VertexIndices,
			STIndices:     t.STIndices,
		}
	}
	return tris, nil
}

// NewMD2STs reads MD2ST structures from an io.ReadSeeker starting at a specified offset and returns them as a slice of pointers.
func NewMD2STs(rs io.ReadSeeker, offset int32, count int32) ([]*MD2ST, error) {
	if _, err := rs.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	pSTs := make([]MD2ST, count)
	if err := binary.Read(rs, binary.LittleEndian, pSTs); err != nil {
		return nil, err
	}

	sts := make([]*MD2ST, count)
	for idx, st := range pSTs {
		sts[idx] = &MD2ST{
			S: st.S,
			T: st.T,
		}
	}
	return sts, nil
}
