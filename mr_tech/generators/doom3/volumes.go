package doom3

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes represents a structure for managing chunks of 3D geometry parsed from a Doom 3 .proc file.
type Volumes struct {
	pFile     *proc.File
	chunkSize float64
	materials *Materials
}

// NewVolumes initializes and returns a pointer to a Volumes instance with the provided proc.File, chunk size, and materials.
func NewVolumes(pFile *proc.File, chunkSize float64, materials *Materials) *Volumes {
	return &Volumes{
		pFile:     pFile,
		chunkSize: chunkSize,
		materials: materials,
	}
}

// Build processes the surfaces of models from the associated proc file, creating and returning 3D volume configurations.
func (vs *Volumes) Build() ([]*config.Volume, error) {
	chunks := make(map[string]*config.Volume)
	var volumes []*config.Volume

	for _, model := range vs.pFile.Models {
		// In Doom 3 .proc files, _area* models contain the static world geometry in world coordinates.
		// Other models (func_static, triggers, movers, fractures) are entity submodels in local coordinates.
		if !strings.HasPrefix(model.Name, "_area") {
			continue
		}
		for _, surface := range model.Surfaces {
			matName := surface.Material
			texNameLC := strings.ToLower(matName)

			// Skip decal overlay surfaces and fake baked shadow planes which require multipass/decal blending.
			if strings.HasPrefix(texNameLC, "textures/decals/") || strings.HasSuffix(texNameLC, "_shadow") {
				continue
			}
			if vs.materials != nil {
				if def := vs.materials.Resolve(matName); def != nil && def.IsDecal {
					continue
				}
			}

			var material *config.Material
			if vs.materials != nil {
				material = vs.materials.CreateConfigMaterial(matName)
			} else {
				kind := ToMaterial(texNameLC)
				material = config.NewConfigMaterial(matName, []string{texNameLC}, kind)
			}

			for i := 0; i+2 < len(surface.Indices); i += 3 {
				i0, i1, i2 := surface.Indices[i], surface.Indices[i+1], surface.Indices[i+2]
				if i0 >= len(surface.Vertices) || i1 >= len(surface.Vertices) || i2 >= len(surface.Vertices) {
					continue
				}

				v0, v1, v2 := surface.Vertices[i0], surface.Vertices[i1], surface.Vertices[i2]
				tri := []geometry.XYZ{v0.Pos, v1.Pos, v2.Pos}
				triUvs := [][2]float64{{v0.U, v0.V}, {v1.U, v1.V}, {v2.U, v2.V}}

				cx := (tri[0].X + tri[1].X + tri[2].X) / 3.0
				cy := (tri[0].Y + tri[1].Y + tri[2].Y) / 3.0
				cz := (tri[0].Z + tri[1].Z + tri[2].Z) / 3.0

				gridX := int(math.Floor(cx / vs.chunkSize))
				gridY := int(math.Floor(cy / vs.chunkSize))
				gridZ := int(math.Floor(cz / vs.chunkSize))

				chunkKey := fmt.Sprintf("%d_%d_%d", gridX, gridY, gridZ)
				volume, exists := chunks[chunkKey]
				if !exists {
					chunkId := fmt.Sprintf("doom3_chunk_%s", chunkKey)
					volume = config.NewConfigVolume(chunkId, "doom3_world_chunk")
					chunks[chunkKey] = volume
					volumes = append(volumes, volume)
				}
				volume.Faces = append(volume.Faces, config.NewConfigFace(tri, triUvs, material, matName))
			}
		}
	}

	return volumes, nil
}
