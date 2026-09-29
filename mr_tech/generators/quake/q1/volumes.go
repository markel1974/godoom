package q1

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes represents a container for chunk-based 3D volumes with unique identifiers and configurable chunk size.
type Volumes struct {
	chunkSize float64
	vIdx      string
}

// NewVolumes initializes a Volumes instance with specified chunk size and a unique identifier based on the given index.
func NewVolumes(mIdx int, chunkSize float64) *Volumes {
	return &Volumes{
		chunkSize: chunkSize,
		vIdx:      strconv.Itoa(mIdx),
	}
}

// Generate processes raw face data to create volumes and faces, assigning materials and spatially hashing into chunks.
func (vs *Volumes) Generate(faces []*lumps.RawFace) []*config.Volume {
	chunks := make(map[string]*config.Volume)
	var res []*config.Volume

	for _, v := range faces {
		animKind := config.MaterialKindLoop
		velX, velY := 0.0, 0.0
		if v.IsSky {
			animKind = config.MaterialKindSky
			velX, velY = 0.05, 0.05
		} else if len(v.TexName) > 0 && v.TexName[0] == '*' {
			animKind = config.MaterialKindLiquid
		}
		material := config.NewConfigMaterial([]string{v.TexName}, animKind, 1.0, 1.0, velX, velY)
		triangles := lumps.TriangulateConvex3d(v.Points)

		//isLiquid := len(v.TexName) > 0 && v.TexName[0] == '*'
		for _, rawTri := range triangles {
			var rawTriUvs [][2]float64
			if len(v.UVs) > 0 {
				rawTriUvs = make([][2]float64, 3)
				for k := 0; k < 3; k++ {
					pos := rawTri[k]
					for idx, pt := range v.Points {
						if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
							if len(v.UVs) > idx {
								rawTriUvs[k] = v.UVs[idx]
							}
							break
						}
					}
				}

				// Find the triangle centroid
				cx := (rawTri[0].X + rawTri[1].X + rawTri[2].X) / 3.0
				cy := (rawTri[0].Y + rawTri[1].Y + rawTri[2].Y) / 3.0
				cz := (rawTri[0].Z + rawTri[1].Z + rawTri[2].Z) / 3.0

				// Calculate the spatial hashing key (grid coordinates)
				gridX := int(math.Floor(cx / vs.chunkSize))
				gridY := int(math.Floor(cy / vs.chunkSize))
				gridZ := int(math.Floor(cz / vs.chunkSize))

				chunkKey := fmt.Sprintf("%d_%d_%d", gridX, gridY, gridZ)
				volume, exists := chunks[chunkKey]
				if !exists {
					chunkId := fmt.Sprintf("quake_world_%s_chunk_%s", vs.vIdx, chunkKey)
					volume = config.NewConfigVolume(chunkId, "quake_bsp_chunk")
					chunks[chunkKey] = volume
					res = append(res, volume)
				}

				volume.Faces = append(volume.Faces, config.NewConfigFace([]geometry.XYZ{rawTri[0], rawTri[1], rawTri[2]}, rawTriUvs, material, v.TexName))
			}
		}
	}

	return res
}
