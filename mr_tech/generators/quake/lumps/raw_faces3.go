package lumps

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// NewRawFaces3 processes a 3D model's data to extract and create raw face representations from a Quake 3 BSP file.
func NewRawFaces3(rs io.ReadSeeker, headers Headers3, modelIdx int, noDraw map[string]bool) ([]*RawFace, error) {
	lModels := headers.Lumps[LumpModels3]
	if _, err := rs.Seek(int64(lModels.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to models lump: %w", err)
	}
	models := make([]Model3, int(lModels.Length)/40)
	if err := binary.Read(rs, binary.LittleEndian, &models); err != nil {
		return nil, fmt.Errorf("failed to read models lump: %w", err)
	}

	if modelIdx < 0 || modelIdx >= len(models) {
		return nil, fmt.Errorf("modelIdx out of range")
	}
	targetModel := models[modelIdx]

	//Geometrical lumps
	lFaces := headers.Lumps[LumpFaces3]
	if _, err := rs.Seek(int64(lFaces.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to faces lump: %w", err)
	}
	faces := make([]Face3, int(lFaces.Length)/104)
	if err := binary.Read(rs, binary.LittleEndian, &faces); err != nil {
		return nil, fmt.Errorf("failed to read faces lump: %w", err)
	}
	lVerts := headers.Lumps[LumpVertexes3]
	if _, err := rs.Seek(int64(lVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to vertexes lump: %w", err)
	}
	vertexes := make([]Vertex3, int(lVerts.Length)/44)
	if err := binary.Read(rs, binary.LittleEndian, &vertexes); err != nil {
		return nil, fmt.Errorf("failed to read vertexes lump: %w", err)
	}
	lMeshVerts := headers.Lumps[LumpMeshVerts3]
	if _, err := rs.Seek(int64(lMeshVerts.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to mesh vertices lump: %w", err)
	}
	meshVerts := make([]int32, int(lMeshVerts.Length)/4)
	if err := binary.Read(rs, binary.LittleEndian, &meshVerts); err != nil {
		return nil, fmt.Errorf("failed to read mesh vertices lump: %w", err)
	}
	lTextures := headers.Lumps[LumpTextures3]
	if _, err := rs.Seek(int64(lTextures.Offset), io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to textures lump: %w", err)
	}
	textures := make([]Texture3, int(lTextures.Length)/72)
	if err := binary.Read(rs, binary.LittleEndian, &textures); err != nil {
		return nil, fmt.Errorf("failed to read textures lump: %w", err)
	}

	var rawFaces []*RawFace

	// 3. Risoluzione Topologica
	for i := int32(0); i < targetModel.NumFaces; i++ {
		face := faces[targetModel.FirstFace+i]
		tex := textures[face.TextureID]
		if (tex.Flags & 0x80) != 0 {
			continue // SURF_NODRAW
		}
		texNameBytes := make([]byte, 0, len(tex.Name))
		for _, b := range tex.Name {
			if b == 0 || len(texNameBytes) >= len(tex.Name)-1 {
				break
			}
			texNameBytes = append(texNameBytes, b)
		}
		texName := strings.TrimSpace(strings.ToLower(string(texNameBytes)))
		if _, found := noDraw[texName]; found {
			continue
		}
		isSky := (tex.Flags & 0x4) != 0 // SURF_SKY
		switch face.Type {
		case 1, 3: // Poligono Convesso (1) o Mesh Complessa (3)
			// usiamo l'indicizzazione per formare direttamente triangoli
			for j := int32(0); j < face.NumMesh; j += 3 {
				var tri []geometry.XYZ
				var uvs [][2]float64
				for k := int32(0); k < 3; k++ {
					vIdx := face.VertexStart + meshVerts[face.MeshStart+j+k]
					v := vertexes[vIdx]
					tri = append(tri, CreateXYZ(float64(v.Position[0]), float64(v.Position[1]), float64(v.Position[2])))
					uvs = append(uvs, [2]float64{float64(v.TexCoord[0]), float64(v.TexCoord[1])})
				}
				rawFaces = append(rawFaces, &RawFace{
					Points:  tri, // non dovremo fare il Fan se riceve già 3 punti
					UVs:     uvs,
					TexName: texName,
					IsSky:   isSky,
				})
			}

		case 2: // PATCH DI BEZIER (Biquadratica)
			w := int(face.PatchSize[0])
			h := int(face.PatchSize[1])

			// Le patch in Q3 sono griglie 3x3 unite. Troviamo quante sub-patch ci sono.
			numPatchesX := (w - 1) / 2
			numPatchesY := (h - 1) / 2

			for y := 0; y < numPatchesY; y++ {
				for x := 0; x < numPatchesX; x++ {
					var cp [9]Vertex3
					for row := 0; row < 3; row++ {
						for col := 0; col < 3; col++ {
							cpIdx := face.VertexStart + int32((y*2+row)*w+(x*2+col))
							cp[row*3+col] = vertexes[cpIdx]
						}
					}

					// Livello di Tassellatura (LOD). 5 = Risoluzione standard.
					triangles, uvs := Tessellate(cp, 5)

					for t := 0; t < len(triangles); t += 3 {
						rawFaces = append(rawFaces, &RawFace{
							Points:  []geometry.XYZ{triangles[t], triangles[t+1], triangles[t+2]},
							UVs:     [][2]float64{uvs[t], uvs[t+1], uvs[t+2]},
							TexName: texName,
							IsSky:   isSky,
						})
					}
				}
			}
		}
	}

	return rawFaces, nil
}
