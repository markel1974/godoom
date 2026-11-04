package proc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// UVKey represents a key used to identify a specific vertex within a polygon in UV mapping.
type UVKey struct {
	Poly int
	Vert int
}

// LWOLayer represents a layer chunk (LAYR) in a LightWave Object.
type LWOLayer struct {
	Number int
	Flags  int
	Pivot  geometry.XYZ
	Name   string
	Parent int
}

// LWOSurface represents surface attributes defined in a SURF chunk.
type LWOSurface struct {
	Name        string
	Source      string
	Color       [3]float32
	Diffuse     float32
	Specular    float32
	Glossiness  float32
	Luminosity  float32
	ClipIndex   int
	TexturePath string
}

// LWO represents a LightWave Object file, including its tags, points, and polygonal structure with associated UV mapping data.
type LWO struct {
	Tags     []string
	Points   []geometry.XYZ
	Polygons [][]int
	PolyTags map[int]int
	VMadUVs  map[UVKey][2]float32
	VMapUVs  map[int][2]float32

	Layers   []LWOLayer
	Surfaces map[string]*LWOSurface
	Clips    map[int]string

	BBoxMin geometry.XYZ
	BBoxMax geometry.XYZ
	HasBBox bool
}

// NewLWO creates and returns a pointer to a new LWO struct with initialized maps for PolyTags, VMadUVs, VMapUVs, Surfaces, and Clips.
func NewLWO() *LWO {
	return &LWO{
		PolyTags: make(map[int]int),
		VMadUVs:  make(map[UVKey][2]float32),
		VMapUVs:  make(map[int][2]float32),
		Surfaces: make(map[string]*LWOSurface),
		Clips:    make(map[int]string),
	}
}

// cleanTexturePath strips drive letters, game root prefixes, and file extensions from a texture path.
func cleanTexturePath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimSpace(p)
	if idx := strings.Index(p, ":"); idx != -1 {
		p = p[idx+1:]
	}
	p = strings.TrimPrefix(p, "/")
	pLower := strings.ToLower(p)
	if strings.HasPrefix(pLower, "doom/base/") {
		p = p[10:]
	} else if strings.HasPrefix(pLower, "base/") {
		p = p[5:]
	}
	p = strings.TrimPrefix(p, "/")
	ext := filepath.Ext(p)
	if ext != "" {
		p = strings.TrimSuffix(p, ext)
	}
	return strings.TrimSpace(p)
}

// readS0 reads a null-terminated, 2-byte aligned string (IFF S0 format) from data starting at offset.
func (l *LWO) readS0(data []byte, offset int) (string, int) {
	if offset >= len(data) {
		return "", offset
	}
	nullIdx := bytes.IndexByte(data[offset:], 0)
	if nullIdx < 0 {
		return string(data[offset:]), len(data)
	}
	str := string(data[offset : offset+nullIdx])
	next := offset + nullIdx + 1
	if (nullIdx+1)%2 != 0 {
		next++ // IFF S0 2-byte alignment padding
	}
	if next > len(data) {
		next = len(data)
	}
	return str, next
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
	currentPointOffset := 0
	currentPolyOffset := 0

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

		case "LAYR":
			currentPointOffset = len(l.Points)
			currentPolyOffset = len(l.Polygons)

			layer := LWOLayer{Parent: -1}
			if len(cData) >= 16 {
				layer.Number = int(binary.BigEndian.Uint16(cData[:2]))
				layer.Flags = int(binary.BigEndian.Uint16(cData[2:4]))
				px := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[4:8])))
				py := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[8:12])))
				pz := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[12:16])))
				// Convert LW (left-handed, Y-up) to Doom 3 (right-handed, Z-up):
				layer.Pivot = geometry.XYZ{X: px, Y: pz, Z: py}

				name, nPos := l.readS0(cData, 16)
				layer.Name = name
				if nPos+2 <= len(cData) {
					layer.Parent = int(binary.BigEndian.Uint16(cData[nPos : nPos+2]))
				}
			}
			l.Layers = append(l.Layers, layer)

		case "PNTS":
			numPoints := len(cData) / blockSize
			for i := 0; i < numPoints; i++ {
				off := i * blockSize
				bitsX := binary.BigEndian.Uint32(cData[off : off+4])
				bitsY := binary.BigEndian.Uint32(cData[off+4 : off+8])
				bitsZ := binary.BigEndian.Uint32(cData[off+8 : off+blockSize])
				// LightWave coordinate system (left-handed, Y-up) to Doom 3 (right-handed, Z-up):
				pt := geometry.XYZ{
					X: float64(math.Float32frombits(bitsX)),
					Y: float64(math.Float32frombits(bitsZ)),
					Z: float64(math.Float32frombits(bitsY)),
				}
				l.Points = append(l.Points, pt)
			}

		case "BBOX":
			if len(cData) >= 24 {
				minX := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[0:4])))
				minY := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[4:8])))
				minZ := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[8:12])))
				maxX := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[12:16])))
				maxY := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[16:20])))
				maxZ := float64(math.Float32frombits(binary.BigEndian.Uint32(cData[20:24])))

				// Convert LW (left-handed, Y-up) to Doom 3 (right-handed, Z-up):
				bboxMin := geometry.XYZ{X: minX, Y: minZ, Z: minY}
				bboxMax := geometry.XYZ{X: maxX, Y: maxZ, Z: maxY}

				if !l.HasBBox {
					l.BBoxMin = bboxMin
					l.BBoxMax = bboxMax
					l.HasBBox = true
				} else {
					if bboxMin.X < l.BBoxMin.X {
						l.BBoxMin.X = bboxMin.X
					}
					if bboxMin.Y < l.BBoxMin.Y {
						l.BBoxMin.Y = bboxMin.Y
					}
					if bboxMin.Z < l.BBoxMin.Z {
						l.BBoxMin.Z = bboxMin.Z
					}
					if bboxMax.X > l.BBoxMax.X {
						l.BBoxMax.X = bboxMax.X
					}
					if bboxMax.Y > l.BBoxMax.Y {
						l.BBoxMax.Y = bboxMax.Y
					}
					if bboxMax.Z > l.BBoxMax.Z {
						l.BBoxMax.Z = bboxMax.Z
					}
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
						vi, nextPos := l.vxIndex(cData, cPos)
						cPos = nextPos
						poly = append(poly, vi+currentPointOffset)
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
					l.PolyTags[polyIdx+currentPolyOffset] = tagIdx
				}
			}

		case "VMAP":
			if len(cData) > 6 && string(cData[:4]) == "TXUV" {
				dim := int(binary.BigEndian.Uint16(cData[4:6]))
				if dim == 2 {
					_, cPos := l.readS0(cData, 6)
					for cPos+8 <= len(cData) {
						vi, nextPos := l.vxIndex(cData, cPos)
						cPos = nextPos
						if cPos+8 > len(cData) {
							break
						}
						u := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos : cPos+4]))
						v := math.Float32frombits(binary.BigEndian.Uint32(cData[cPos+4 : cPos+8]))
						cPos += 8
						// LightWave V coordinates are bottom-to-top (0=bottom, 1=top), while Doom 3 is top-to-bottom (0=top, 1=bottom)
						l.VMapUVs[vi+currentPointOffset] = [2]float32{u, 1.0 - v}
					}
				}
			}

		case "VMAD":
			if len(cData) > 6 && string(cData[:4]) == "TXUV" {
				dim := int(binary.BigEndian.Uint16(cData[4:6]))
				if dim == 2 {
					_, cPos := l.readS0(cData, 6)
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
						// LightWave V coordinates are bottom-to-top (0=bottom, 1=top), while Doom 3 is top-to-bottom (0=top, 1=bottom)
						l.VMadUVs[UVKey{Poly: pi + currentPolyOffset, Vert: vi + currentPointOffset}] = [2]float32{u, 1.0 - v}
					}
				}
			}

		case "CLIP":
			if len(cData) >= 4 {
				clipIndex := int(binary.BigEndian.Uint32(cData[:4]))
				subPos := 4
				for subPos+6 <= len(cData) {
					subType := string(cData[subPos : subPos+4])
					subSize := int(binary.BigEndian.Uint16(cData[subPos+4 : subPos+6]))
					subStart := subPos + 6
					subEnd := subStart + subSize
					if subEnd > len(cData) {
						subEnd = len(cData)
					}
					subData := cData[subStart:subEnd]
					subPos = subEnd
					if subSize%2 == 1 {
						subPos++ // 2-byte alignment padding
					}

					if subType == "STIL" {
						path, _ := l.readS0(subData, 0)
						clean := cleanTexturePath(path)
						if clean != "" {
							l.Clips[clipIndex] = clean
						}
					}
				}
			}

		case "SURF":
			name, sPos := l.readS0(cData, 0)
			source, sPos := l.readS0(cData, sPos)
			surf := &LWOSurface{
				Name:       name,
				Source:     source,
				Color:      [3]float32{1.0, 1.0, 1.0},
				Diffuse:    1.0,
				Specular:   0.0,
				Glossiness: 0.0,
			}

			subPos := sPos
			for subPos+6 <= len(cData) {
				subType := string(cData[subPos : subPos+4])
				subSize := int(binary.BigEndian.Uint16(cData[subPos+4 : subPos+6]))
				subStart := subPos + 6
				subEnd := subStart + subSize
				if subEnd > len(cData) {
					subEnd = len(cData)
				}
				subData := cData[subStart:subEnd]
				subPos = subEnd
				if subSize%2 == 1 {
					subPos++ // 2-byte alignment padding
				}

				switch subType {
				case "COLR":
					if len(subData) >= 12 {
						surf.Color[0] = math.Float32frombits(binary.BigEndian.Uint32(subData[0:4]))
						surf.Color[1] = math.Float32frombits(binary.BigEndian.Uint32(subData[4:8]))
						surf.Color[2] = math.Float32frombits(binary.BigEndian.Uint32(subData[8:12]))
					}
				case "DIFF":
					if len(subData) >= 4 {
						surf.Diffuse = math.Float32frombits(binary.BigEndian.Uint32(subData[:4]))
					}
				case "LUMI":
					if len(subData) >= 4 {
						surf.Luminosity = math.Float32frombits(binary.BigEndian.Uint32(subData[:4]))
					}
				case "SPEC":
					if len(subData) >= 4 {
						surf.Specular = math.Float32frombits(binary.BigEndian.Uint32(subData[:4]))
					}
				case "GLOS":
					if len(subData) >= 4 {
						surf.Glossiness = math.Float32frombits(binary.BigEndian.Uint32(subData[:4]))
					}
				case "BLOK":
					bPos := 0
					for bPos+6 <= len(subData) {
						bType := string(subData[bPos : bPos+4])
						bSize := int(binary.BigEndian.Uint16(subData[bPos+4 : bPos+6]))
						bStart := bPos + 6
						bEnd := bStart + bSize
						if bEnd > len(subData) {
							bEnd = len(subData)
						}
						bData := subData[bStart:bEnd]
						bPos = bEnd
						if bSize%2 == 1 {
							bPos++
						}

						if bType == "IMAG" && len(bData) >= 2 {
							surf.ClipIndex = int(binary.BigEndian.Uint16(bData[:2]))
						}
					}
				}
			}

			if surf.ClipIndex > 0 {
				if clipPath, ok := l.Clips[surf.ClipIndex]; ok {
					surf.TexturePath = clipPath
				}
			}
			l.Surfaces[name] = surf

		default:
			// Safely skip any other chunks (e.g. ENVL, DESC, TEXT, etc.)
		}
	}

	// Link any surfaces whose CLIP chunk was defined after the SURF chunk
	for _, surf := range l.Surfaces {
		if surf.TexturePath == "" && surf.ClipIndex > 0 {
			if clipPath, ok := l.Clips[surf.ClipIndex]; ok {
				surf.TexturePath = clipPath
			}
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
