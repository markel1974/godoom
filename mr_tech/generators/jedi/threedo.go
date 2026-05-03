package jedi

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

// ThreedoQuad represents a single face (which seems to default to Quads in this format).
type ThreedoQuad struct {
	VertexIndices []int // Indices into the object's vertex list
	Color         int
	Fill          string // e.g., "PLANE", "TEXTURE"
}

// ThreedoTexQuad maps texture coordinate indices to a quad.
type ThreedoTexQuad struct {
	TexVertIndices []int // Indices into the object's texture vertex list
}

// ThreedoObject represents a distinct sub-mesh within the 3DO file.
type ThreedoObject struct {
	Name        string
	TextureIdx  int // Index into the global Textures array
	Vertices    []geometry.XYZ
	Quads       []ThreedoQuad
	TexVertices [][2]float64 // [U, V]
	TexQuads    []ThreedoTexQuad
}

// Threedo represents the entire parsed 3D model.
type Threedo struct {
	Name     string
	Palette  string
	Textures []string
	Objects  []*ThreedoObject
}

func NewThreedo() *Threedo {
	return &Threedo{}
}

// Parse legge un file 3DO in formato testo
func (t *Threedo) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)

	type ParseMode int
	const (
		ModeGlobal ParseMode = iota
		ModeGlobalTextures
		ModeObjectHeader
		ModeObjectVertices
		ModeObjectQuads
		ModeObjectTexVertices
		ModeObjectTexQuads
	)

	currentMode := ModeGlobal
	var currentObject *ThreedoObject

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip comments and empty lines
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		// Look for section headers that change the parsing mode
		if strings.HasPrefix(line, "TEXTURES") && len(parts) == 2 {
			currentMode = ModeGlobalTextures
			continue
		} else if strings.HasPrefix(line, "OBJECT ") {
			// Start a new object
			currentObject = &ThreedoObject{}
			// Extract name, e.g., OBJECT "PLANES" -> PLANES
			currentObject.Name = strings.Trim(strings.Join(parts[1:], " "), "\"")
			t.Objects = append(t.Objects, currentObject)
			currentMode = ModeObjectHeader
			continue
		} else if strings.HasPrefix(line, "TEXTURE VERTICES") {
			currentMode = ModeObjectTexVertices
			continue
		} else if strings.HasPrefix(line, "TEXTURE QUADS") {
			currentMode = ModeObjectTexQuads
			continue
		} else if strings.HasPrefix(line, "VERTICES") {
			currentMode = ModeObjectVertices
			continue
		} else if strings.HasPrefix(line, "QUADS") {
			currentMode = ModeObjectQuads
			continue
		}

		// Parse data based on the current mode
		switch currentMode {
		case ModeGlobal:
			if parts[0] == "3DONAME" && len(parts) > 1 {
				t.Name = parts[1]
			} else if parts[0] == "PALETTE" && len(parts) > 1 {
				t.Palette = parts[1]
			}

		case ModeGlobalTextures:
			if parts[0] == "TEXTURE:" && len(parts) >= 2 {
				// E.g., TEXTURE: ZPGREY2.BM #0 -> keep ZPGREY2.BM
				t.Textures = append(t.Textures, parts[1])
			}

		case ModeObjectHeader:
			if parts[0] == "TEXTURE" && len(parts) == 2 && currentObject != nil {
				idx, _ := strconv.Atoi(parts[1])
				currentObject.TextureIdx = idx
			}

		case ModeObjectVertices:
			// Expected format: num: x y z
			if len(parts) >= 4 && strings.HasSuffix(parts[0], ":") && currentObject != nil {
				x, _ := strconv.ParseFloat(parts[1], 64)
				y, _ := strconv.ParseFloat(parts[2], 64)
				z, _ := strconv.ParseFloat(parts[3], 64)
				currentObject.Vertices = append(currentObject.Vertices, geometry.XYZ{X: x, Y: y, Z: z})
			}

		case ModeObjectQuads:
			// Expected format: num: a b c d color fill
			if len(parts) >= 7 && strings.HasSuffix(parts[0], ":") && currentObject != nil {
				quad := ThreedoQuad{}
				for i := 1; i <= 4; i++ {
					vIdx, _ := strconv.Atoi(parts[i])
					quad.VertexIndices = append(quad.VertexIndices, vIdx)
				}
				quad.Color, _ = strconv.Atoi(parts[5])
				quad.Fill = parts[6]
				currentObject.Quads = append(currentObject.Quads, quad)
			}

		case ModeObjectTexVertices:
			// Expected format: num: u v
			if len(parts) >= 3 && strings.HasSuffix(parts[0], ":") && currentObject != nil {
				u, _ := strconv.ParseFloat(parts[1], 64)
				v, _ := strconv.ParseFloat(parts[2], 64)
				currentObject.TexVertices = append(currentObject.TexVertices, [2]float64{u, v})
			}

		case ModeObjectTexQuads:
			// Expected format: num: a b c d
			if len(parts) >= 5 && strings.HasSuffix(parts[0], ":") && currentObject != nil {
				tQuad := ThreedoTexQuad{}
				for i := 1; i <= 4; i++ {
					tvIdx, _ := strconv.Atoi(parts[i])
					tQuad.TexVertIndices = append(tQuad.TexVertIndices, tvIdx)
				}
				currentObject.TexQuads = append(currentObject.TexQuads, tQuad)
			}
		}
	}
	return scanner.Err()
}

func (t *Threedo) ToMD2() (*config.MD2, []string) {
	var allTriangles [][3]config.MD2Vertex
	var usedTextures []string

	// Create a quick lookup map to avoid duplicate texture names in our return list
	texMap := make(map[string]bool)

	// Iterate over all sub-objects
	for _, obj := range t.Objects {
		// Track which textures are actually used
		if obj.TextureIdx >= 0 && obj.TextureIdx < len(t.Textures) {
			texName := t.Textures[obj.TextureIdx]
			if !texMap[texName] {
				usedTextures = append(usedTextures, texName)
				texMap[texName] = true
			}
		}

		// Iterate over the quads (or N-gons) in this object
		for qIdx, quad := range obj.Quads {
			pLen := len(quad.VertexIndices)
			if pLen < 3 {
				continue
			}

			// Ensure we have matching texture coordinates if the fill type uses them
			hasUVs := quad.Fill == "TEXTURE" && qIdx < len(obj.TexQuads) && len(obj.TexQuads[qIdx].TexVertIndices) == pLen

			// Triangle Fan triangulation (anchored at vertex 0)
			for i := 1; i < pLen-1; i++ {

				// 1. Get physical positions
				v0 := obj.Vertices[quad.VertexIndices[0]]
				v1 := obj.Vertices[quad.VertexIndices[i]]
				v2 := obj.Vertices[quad.VertexIndices[i+1]]

				// 2. Get UV coordinates (default to 0.0 if not a textured face)
				var uv0, uv1, uv2 [2]float64
				if hasUVs {
					uv0 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[0]]
					uv1 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[i]]
					uv2 = obj.TexVertices[obj.TexQuads[qIdx].TexVertIndices[i+1]]
				}

				// Build the triangle for your engine
				tri := [3]config.MD2Vertex{
					{Pos: v0, U: float32(uv0[0]), V: float32(uv0[1])},
					{Pos: v1, U: float32(uv1[0]), V: float32(uv1[1])},
					{Pos: v2, U: float32(uv2[0]), V: float32(uv2[1])},
				}
				allTriangles = append(allTriangles, tri)
			}
		}
	}

	// Create a single-frame MD2
	cModel := config.NewMD2(1, []string{"stand"})
	cModel.Frames[0].Triangles = allTriangles
	return cModel, usedTextures
}
