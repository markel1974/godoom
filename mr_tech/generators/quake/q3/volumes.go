package q3

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
)

// Volumes represents a structure that works with Shaders to facilitate geometry processing and face generation.
type Volumes struct {
	shaders *Shaders
}

// NewVolumes initializes and returns a new Faces instance with the provided Shaders dependency.
func NewVolumes(shaders *Shaders) *Volumes {
	return &Volumes{
		shaders: shaders,
	}
}

// Create generates and groups faces into 3D volumes based on spatial hashing, material properties, and shader configurations.
func (f *Volumes) Create(mIdx int, faces []*lumps.RawFace) ([]*config.Volume, error) {
	const chunkSize = float64(1024)

	var volumes []*config.Volume
	chunks := make(map[string]*config.Volume)
	vIdx := strconv.Itoa(mIdx)
	for _, v := range faces {
		animKind := config.MaterialKindLoop
		if v.IsSky {
			animKind = config.MaterialKindSky
		}
		texNameLC := strings.ToLower(v.TexName)

		var material *config.Material
		if animMap := f.shaders.GetAnimMap(texNameLC); len(animMap) > 0 {
			animMapLC := make([]string, len(animMap))
			for i, frame := range animMap {
				animMapLC[i] = strings.ToLower(frame)
			}
			material = config.NewConfigMaterial(animMapLC, animKind, 1.0, 1.0, 0, 0)
		} else if v.IsSky {
			if editorImg := f.shaders.GetEditorImage(texNameLC); editorImg != "" {
				material = config.NewConfigMaterial([]string{strings.ToLower(editorImg)}, animKind, 1.0, 1.0, 0, 0)
			} else {
				material = config.NewConfigMaterial([]string{texNameLC}, animKind, 1.0, 1.0, 0, 0)
			}
		} else {
			material = config.NewConfigMaterial([]string{texNameLC}, animKind, 1.0, 1.0, 0, 0)
		}
		material.Shader = v.TexName

		if f.shaders.IsAdditive(texNameLC) {
			material.BlendMode = config.BlendModeAdditive
		}
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
			gridX := int(math.Floor(cx / chunkSize))
			gridY := int(math.Floor(cy / chunkSize))
			gridZ := int(math.Floor(cz / chunkSize))

			chunkKey := fmt.Sprintf("%d_%d_%d", gridX, gridY, gridZ)
			volume, exists := chunks[chunkKey]
			if !exists {
				chunkId := fmt.Sprintf("quake_world_%s_chunk_%s", vIdx, chunkKey)
				volume = config.NewConfigVolume(chunkId, "quake_bsp_chunk")
				chunks[chunkKey] = volume
				volumes = append(volumes, volume)
			}
			volume.Faces = append(volume.Faces, config.NewConfigFace(tri, triUvs, material, v.TexName))
		}
	}
	return volumes, nil
}
