package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// UVKey represents a key used to identify a specific vertex within a polygon in UV mapping.
type UVKey struct {
	Poly int
	Vert int
}

// LWO represents a LightWave Object file, including its tags, points, and polygonal structure with associated UV mapping data.
type LWO struct {
	Tags     []string
	Points   []geometry.XYZ
	Polygons [][]int
	PolyTags map[int]int
	VMadUVs  map[UVKey][2]float32
	VMapUVs  map[int][2]float32
}

// NewLWO creates and returns a pointer to a new LWO struct with initialized maps for PolyTags, VMadUVs, and VMapUVs.
func NewLWO() *LWO {
	return &LWO{
		PolyTags: make(map[int]int),
		VMadUVs:  make(map[UVKey][2]float32),
		VMapUVs:  make(map[int][2]float32),
	}
}

// Parse reads an LWO2 file from the provided io.Reader and populates the LWO structure with parsed data.
func (l *LWO) Parse(r io.Reader) error {
	const blockSize = 12
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading LWO: %w", err)
	}
	if len(data) < blockSize {
		return fmt.Errorf("LWO data too short")
	}
	if string(data[:4]) != "FORM" || string(data[8:blockSize]) != "LWO2" {
		return fmt.Errorf("not an LWO2 file")
	}

	pos := blockSize

	for pos+8 <= len(data) {
		chunkType := string(data[pos : pos+4])
		chunkSize := int(binary.BigEndian.Uint32(data[pos+4 : pos+8]))
		chunkStart := pos + 8
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(data) {
			chunkEnd = len(data)
		}

		cData := data[chunkStart:chunkEnd]
		pos = chunkEnd
		if chunkSize%2 == 1 {
			pos++ // IFF 2-byte alignment padding
		}

		switch chunkType {
		case "TAGS":
			// Null-terminated strings
			parts := bytes.Split(cData, []byte{0})
			for _, part := range parts {
				if len(part) > 0 {
					l.Tags = append(l.Tags, string(part))
				}
			}
		case "PNTS":
			numPoints := len(cData) / blockSize
			l.Points = make([]geometry.XYZ, numPoints)
			for i := 0; i < numPoints; i++ {
				off := i * blockSize
				bitsX := binary.BigEndian.Uint32(cData[off : off+4])
				bitsY := binary.BigEndian.Uint32(cData[off+4 : off+8])
				bitsZ := binary.BigEndian.Uint32(cData[off+8 : off+blockSize])
				// LightWave coordinate system (left-handed, Y-up) to Doom 3 (right-handed, Z-up):
				l.Points[i] = geometry.XYZ{X: float64(math.Float32frombits(bitsX)), Y: float64(math.Float32frombits(bitsZ)), Z: float64(math.Float32frombits(bitsY))}
			}
		case "POLS":
			if len(cData) > 4 && string(cData[:4]) == "FACE" {
				cPos := 4
				for cPos+2 <= len(cData) {
					numVerts := int(binary.BigEndian.Uint16(cData[cPos : cPos+2]))
					cPos += 2
					numVerts = numVerts & 0x03FF // mask flags
					var poly []int
					for v := 0; v < numVerts; v++ {
						vi, nextPos := l.vxIndex(cData, cPos)
						cPos = nextPos
						poly = append(poly, vi)
					}
					l.Polygons = append(l.Polygons, poly)
				}
			}
		case "PTAG":
			if len(cData) > 4 && string(cData[:4]) == "SURF" {
				cPos := 4
				for cPos+2 <= len(cData) {
					polyIdx, nextPos := l.vxIndex(cData, cPos)
					cPos = nextPos
					if cPos+2 > len(cData) {
						break
					}
					tagIdx := int(binary.BigEndian.Uint16(cData[cPos : cPos+2]))
					cPos += 2
					l.PolyTags[polyIdx] = tagIdx
				}
			}
		case "VMAP":
			if len(cData) > 6 && string(cData[:4]) == "TXUV" {
				dim := int(binary.BigEndian.Uint16(cData[4:6]))
				if dim == 2 {
					nullIdx := bytes.IndexByte(cData[6:], 0)
					if nullIdx >= 0 {
						cPos := 6 + nullIdx + 1
						if cPos%2 != 0 {
							cPos++
						}
						for cPos+8 <= len(cData) {
							vi, nextPos := l.vxIndex(cData, cPos)
							cPos = nextPos
							if cPos+8 > len(cData) {
								break
							}
							u := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos : cPos+4]))
							v := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos+4 : cPos+8]))
							cPos += 8
							l.VMapUVs[vi] = [2]float32{u, v}
						}
					}
				}
			}
		case "VMAD":
			if len(cData) > 6 && string(cData[:4]) == "TXUV" {
				dim := int(binary.BigEndian.Uint16(cData[4:6]))
				if dim == 2 {
					nullIdx := bytes.IndexByte(cData[6:], 0)
					if nullIdx >= 0 {
						cPos := 6 + nullIdx + 1
						if cPos%2 != 0 {
							cPos++
						}
						for cPos+8 <= len(cData) {
							vi, nextPos := l.vxIndex(cData, cPos)
							cPos = nextPos
							pi, nextPos2 := l.vxIndex(cData, cPos)
							cPos = nextPos2
							if cPos+8 > len(cData) {
								break
							}
							u := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos : cPos+4]))
							v := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos+4 : cPos+8]))
							cPos += 8
							l.VMadUVs[UVKey{Poly: pi, Vert: vi}] = [2]float32{u, v}
						}
					}
				}
			}
		case "LAYR":
			fmt.Println("Unsupported chunk type:", chunkType) //TODO
		case "BBOX":
			fmt.Println("Unsupported chunk type:", chunkType) //TODO
		case "CLIP":
			fmt.Println("Unsupported chunk type:", chunkType) //TODO
		case "SURF":
			fmt.Println("Unsupported chunk type:", chunkType) //TODO
		default:
			fmt.Println("Unsupported chunk type:", chunkType) //TODO
		}
	}

	return nil
}

// vxIndex decodes a variable-length index from a byte slice and returns the index and the updated offset position.
func (l *LWO) vxIndex(data []byte, offset int) (int, int) {
	if offset >= len(data) {
		return 0, offset
	}
	b0 := data[offset]
	if b0 < 0xFF {
		if offset+2 > len(data) {
			return 0, len(data)
		}
		val := (int(b0) << 8) | int(data[offset+1])
		return val, offset + 2
	}
	if offset+4 > len(data) {
		return 0, len(data)
	}
	val := (int(b0&0xFF) << 24) | (int(data[offset+1]) << 16) | (int(data[offset+2]) << 8) | int(data[offset+3])
	return val & 0xFFFFFF, offset + 4
}
