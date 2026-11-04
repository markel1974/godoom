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
	pFile       *proc.File
	chunkSize   float64
	materials   *proc.Materials
	mapEntities []*proc.MapEntity
	arc         proc.IPkArchive
}

// NewVolumes initializes and returns a pointer to a Volumes instance with the provided proc.File, chunk size, materials, map entities, and pk archive.
func NewVolumes(pFile *proc.File, chunkSize float64, materials *proc.Materials, mapEntities []*proc.MapEntity, arc proc.IPkArchive) *Volumes {
	return &Volumes{
		pFile:       pFile,
		chunkSize:   chunkSize,
		materials:   materials,
		mapEntities: mapEntities,
		arc:         arc,
	}
}

// Build processes the surfaces of models from the associated proc file and map entities, creating and returning 3D volume configurations.
func (vs *Volumes) Build() ([]*config.Volume, error) {
	chunks := make(map[string]*config.Volume)
	var volumes []*config.Volume

	addTriangle := func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string) {
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

	entitiesMap := make(map[string]*proc.MapEntity)
	for _, ent := range vs.mapEntities {
		if name, ok := ent.GetProperty("name"); ok && len(name) > 0 {
			entitiesMap[name] = ent
		}
		if model, ok := ent.GetProperty("model"); ok && len(model) > 0 {
			entitiesMap[model] = ent
		}
	}

	for _, model := range vs.pFile.Models {
		// In Doom 3 .proc files, _area* models contain the static world geometry in world coordinates.
		// Other models (func_static, func_door, movers, fractures) are entity submodels in local coordinates.
		isArea := strings.HasPrefix(model.Name, "_area")
		var ent *proc.MapEntity
		if !isArea {
			var ok bool
			ent, ok = entitiesMap[model.Name]
			if !ok {
				continue
			}
			if ent.IsDormant() {
				continue
			}
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
			effectiveMatName := matName
			if vs.materials != nil {
				if matDef := vs.materials.Resolve(matName); matDef != nil {
					if guiMat, guiName := ResolveGuiMaterial(vs.arc, vs.materials, matDef, ent); guiMat != nil {
						material = guiMat
						effectiveMatName = guiName
					} else {
						material = CreateConfigMaterial(matDef)
					}
				}
			}
			if material == nil {
				kind := ToMaterial(texNameLC)
				material = config.NewConfigMaterial(matName, []string{texNameLC}, kind)
			}

			for i := 0; i+2 < len(surface.Indices); i += 3 {
				i0, i1, i2 := surface.Indices[i], surface.Indices[i+1], surface.Indices[i+2]
				if i0 >= len(surface.Vertices) || i1 >= len(surface.Vertices) || i2 >= len(surface.Vertices) {
					continue
				}

				v0, v1, v2 := surface.Vertices[i0], surface.Vertices[i1], surface.Vertices[i2]
				p0, p1, p2 := v0.Pos, v1.Pos, v2.Pos
				if ent != nil {
					p0 = ent.TransformPoint(p0)
					p1 = ent.TransformPoint(p1)
					p2 = ent.TransformPoint(p2)
				}

				tri := []geometry.XYZ{p0, p1, p2}
				triUvs := [][2]float64{{v0.U, v0.V}, {v1.U, v1.V}, {v2.U, v2.V}}
				addTriangle(tri, triUvs, material, effectiveMatName)
			}
		}
	}

	// Process func_static entities with external .lwo models (door frames, computer terminals, shelves, etc.)
	if vs.arc != nil {
		for _, ent := range vs.mapEntities {
			classname, _ := ent.GetProperty("classname")
			if classname != "func_static" {
				continue
			}
			if ent.IsDormant() {
				continue
			}
			modelPath, ok := ent.GetProperty("model")
			if !ok || !strings.HasSuffix(strings.ToLower(modelPath), ".lwo") {
				continue
			}

			cleanPath := strings.TrimSpace(strings.ToLower(strings.ReplaceAll(modelPath, "\\", "/")))
			lf, err := vs.arc.Open(cleanPath)
			if err != nil {
				continue
			}
			lwd := proc.NewLWO()
			if err := lwd.Parse(lf); err != nil || len(lwd.Points) == 0 || len(lwd.Polygons) == 0 {
				continue
			}

			// Build materials for each tag
			tagMaterials := make([]*config.Material, len(lwd.Tags))
			tagNames := make([]string, len(lwd.Tags))
			for ti, tagName := range lwd.Tags {
				tagNames[ti] = tagName
				if vs.materials != nil {
					if matDef := vs.materials.Resolve(tagName); matDef != nil {
						if guiMat, guiName := ResolveGuiMaterial(vs.arc, vs.materials, matDef, ent); guiMat != nil {
							tagMaterials[ti] = guiMat
							tagNames[ti] = guiName
						} else {
							tagMaterials[ti] = CreateConfigMaterial(matDef)
						}
					}
				}
				if tagMaterials[ti] == nil {
					kind := ToMaterial(strings.ToLower(tagName))
					tagMaterials[ti] = config.NewConfigMaterial("lwo.part", []string{tagName}, kind)
				}
			}

			for polyIdx, poly := range lwd.Polygons {
				if len(poly) < 3 {
					continue
				}
				var mat *config.Material
				matName := "lwo.part"
				if tagIdx, ok := lwd.PolyTags[polyIdx]; ok && tagIdx >= 0 && tagIdx < len(tagMaterials) {
					mat = tagMaterials[tagIdx]
					if tagIdx < len(tagNames) {
						matName = tagNames[tagIdx]
					}
				}
				if mat == nil && len(tagMaterials) > 0 {
					mat = tagMaterials[0]
					matName = tagNames[0]
				}

				getVertex := func(vi int) (geometry.XYZ, float64, float64) {
					var p geometry.XYZ
					if vi >= 0 && vi < len(lwd.Points) {
						p = ent.TransformPoint(lwd.Points[vi])
					}
					var u, v float32
					if uv, ok := lwd.VMadUVs[proc.UVKey{Poly: polyIdx, Vert: vi}]; ok {
						u, v = uv[0], uv[1]
					} else if uv, ok := lwd.VMapUVs[vi]; ok {
						u, v = uv[0], uv[1]
					}
					return p, float64(u), float64(v)
				}

				p0, u0, v0 := getVertex(poly[0])
				for k := 1; k < len(poly)-1; k++ {
					p1, u1, v1 := getVertex(poly[k])
					p2, u2, v2 := getVertex(poly[k+1])

					tri := []geometry.XYZ{p0, p2, p1}
					triUvs := [][2]float64{{u0, v0}, {u2, v2}, {u1, v1}}
					addTriangle(tri, triUvs, mat, matName)
				}
			}
		}
	}

	return volumes, nil
}
