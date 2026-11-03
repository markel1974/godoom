package doom3

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// lwoVxIndex reads a variable-length index (2 or 4 bytes) from data at offset.
func lwoVxIndex(data []byte, offset int) (int, int) {
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

// ParseLWO parses a LightWave Object (LWO2) binary model file and converts it into a Config Model3DEntry.
func ParseLWO(r io.Reader, materials *Materials) (*config.Model3DEntry, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading LWO: %w", err)
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("LWO data too short")
	}

	if string(data[:4]) != "FORM" || string(data[8:12]) != "LWO2" {
		return nil, fmt.Errorf("not an LWO2 file")
	}

	pos := 12
	var tags []string
	var points []geometry.XYZ
	var polygons [][]int
	polyTags := make(map[int]int)

	type uvKey struct {
		poly int
		vert int
	}
	vmadUVs := make(map[uvKey][2]float32)
	vmapUVs := make(map[int][2]float32)

	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
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

		switch chunkID {
		case "TAGS":
			// Null-terminated strings
			parts := bytes.Split(cData, []byte{0})
			for _, part := range parts {
				if len(part) > 0 {
					tags = append(tags, string(part))
				}
			}

		case "PNTS":
			numPoints := len(cData) / 12
			points = make([]geometry.XYZ, numPoints)
			for i := 0; i < numPoints; i++ {
				off := i * 12
				bitsX := binary.BigEndian.Uint32(cData[off : off+4])
				bitsY := binary.BigEndian.Uint32(cData[off+4 : off+8])
				bitsZ := binary.BigEndian.Uint32(cData[off+8 : off+12])
				points[i] = geometry.XYZ{
					X: float64(math.Float32frombits(bitsX)),
					Y: float64(math.Float32frombits(bitsY)),
					Z: float64(math.Float32frombits(bitsZ)),
				}
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
						vi, nextPos := lwoVxIndex(cData, cPos)
						cPos = nextPos
						poly = append(poly, vi)
					}
					polygons = append(polygons, poly)
				}
			}

		case "PTAG":
			if len(cData) > 4 && string(cData[:4]) == "SURF" {
				cPos := 4
				for cPos+2 <= len(cData) {
					polyIdx, nextPos := lwoVxIndex(cData, cPos)
					cPos = nextPos
					if cPos+2 > len(cData) {
						break
					}
					tagIdx := int(binary.BigEndian.Uint16(cData[cPos : cPos+2]))
					cPos += 2
					polyTags[polyIdx] = tagIdx
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
							vi, nextPos := lwoVxIndex(cData, cPos)
							cPos = nextPos
							if cPos+8 > len(cData) {
								break
							}
							u := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos : cPos+4]))
							v := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos+4 : cPos+8]))
							cPos += 8
							vmapUVs[vi] = [2]float32{u, v}
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
							vi, nextPos := lwoVxIndex(cData, cPos)
							cPos = nextPos
							pi, nextPos2 := lwoVxIndex(cData, cPos)
							cPos = nextPos2
							if cPos+8 > len(cData) {
								break
							}
							u := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos : cPos+4]))
							v := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos+4 : cPos+8]))
							cPos += 8
							vmadUVs[uvKey{poly: pi, vert: vi}] = [2]float32{u, v}
						}
					}
				}
			}
		}
	}

	if len(points) == 0 || len(polygons) == 0 {
		return nil, fmt.Errorf("no geometry found in LWO")
	}

	// Build materials for each tag
	tagMaterials := make([]*config.Material, len(tags))
	for i, tagName := range tags {
		if materials != nil {
			tagMaterials[i] = materials.CreateConfigMaterial(tagName)
		}
		if tagMaterials[i] == nil {
			tagMaterials[i] = config.NewConfigMaterial("lwo.part", []string{tagName}, config.MaterialKindSolid)
		}
	}

	var triangles []config.Model3DEntryTriangle

	for polyIdx, poly := range polygons {
		if len(poly) < 3 {
			continue
		}

		var mat *config.Material
		if tagIdx, ok := polyTags[polyIdx]; ok && tagIdx >= 0 && tagIdx < len(tagMaterials) {
			mat = tagMaterials[tagIdx]
		}
		if mat == nil && len(tagMaterials) > 0 {
			mat = tagMaterials[0]
		}

		getVertex := func(vi int) config.Model3DEntryVertex {
			var p geometry.XYZ
			if vi >= 0 && vi < len(points) {
				p = points[vi]
			}
			var u, v float32
			if uv, ok := vmadUVs[uvKey{poly: polyIdx, vert: vi}]; ok {
				u, v = uv[0], uv[1]
			} else if uv, ok := vmapUVs[vi]; ok {
				u, v = uv[0], uv[1]
			}
			return config.Model3DEntryVertex{
				Pos: p,
				U:   u,
				V:   v,
			}
		}

		// Triangulate convex polygons (triangle fan)
		v0 := getVertex(poly[0])
		for i := 1; i < len(poly)-1; i++ {
			v1 := getVertex(poly[i])
			v2 := getVertex(poly[i+1])

			tri := config.NewModel3DEntryTriangle(mat)
			tri.Vertices[0] = v0
			tri.Vertices[1] = v1
			tri.Vertices[2] = v2
			triangles = append(triangles, tri)
		}
	}

	if len(triangles) == 0 {
		return nil, fmt.Errorf("no triangles generated from LWO")
	}

	cModel := config.NewModel3DEntry(1, []string{"idle"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(triangles)
	cModel.ActionDefinitions = []string{"idle"}
	cModel.ActionIntervals = [][2]int{{0, 0}}

	cModel.DropToFloor()

	return cModel, nil
}
