package q2

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

// Volumes is a structure for managing 3D spatial data partitioning, identified by an index and a chunk size.
type Volumes struct {
	vIdx      string
	chunkSize float64
}

// NewVolumes creates and initializes a new Volumes instance with the provided index and chunk size.
func NewVolumes(mIdx int, chunkSize float64) *Volumes {
	return &Volumes{
		vIdx:      strconv.Itoa(mIdx),
		chunkSize: chunkSize,
	}
}

// Create processes a list of RawFace instances, organizes them into spatial chunks, and converts them into Volumes.
func (vs *Volumes) Create(faces []*lumps.RawFace2) []*config.Volume {
	chunks := make(map[string]*config.Volume)
	var res []*config.Volume

	for _, v := range faces {
		animKind := config.MaterialKindLoop
		if v.IsSky() {
			animKind = config.MaterialKindSky
		} else if v.IsWarp() {
			animKind = config.MaterialKindLiquid
		}
		material := config.NewConfigMaterial([]string{v.TexName}, animKind, 1.0, 1.0, 0, 0)
		triangles := lumps.TriangulateConvex3d(v.Points)

		for _, tri := range triangles {
			var triUvs [][2]float64
			if len(v.UVs) > 0 {
				triUvs = make([][2]float64, 3)
				for k := 0; k < 3; k++ {
					pos := tri[k]
					for idx, pt := range v.Points {
						if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
							if len(v.UVs) > idx {
								triUvs[k] = v.UVs[idx]
							}
							break
						}
					}
				}
			}

			// 2. Find the triangle centroid
			cx := (tri[0].X + tri[1].X + tri[2].X) / 3.0
			cy := (tri[0].Y + tri[1].Y + tri[2].Y) / 3.0
			cz := (tri[0].Z + tri[1].Z + tri[2].Z) / 3.0

			// 3. Calculate the spatial hashing key (grid coordinates)
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
			volume.Faces = append(volume.Faces, config.NewConfigFace(tri, triUvs, material, v.TexName))
		}
	}
	return res
}
