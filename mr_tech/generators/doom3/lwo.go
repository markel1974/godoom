package doom3

import (
	"fmt"
	"io"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// BakeLWO parses a LightWave Object (LWO2) binary model file and converts it into a Config Model3DEntry.
func (t *Things) bakeLWO(r io.Reader) (*config.Model3DEntry, error) {
	lwd := proc.NewLWO()
	if err := lwd.Parse(r); err != nil {
		return nil, err
	}

	if len(lwd.Points) == 0 || len(lwd.Polygons) == 0 {
		return nil, fmt.Errorf("no geometry found in LWO")
	}

	// Build materials for each tag
	tagMaterials := make([]*config.Material, len(lwd.Tags))
	for i, tagName := range lwd.Tags {
		if matDef := t.materials.Resolve(tagName); matDef != nil {
			if guiMat, _ := ResolveGUIMaterial(t.arc, t.materials, matDef, nil); guiMat != nil {
				tagMaterials[i] = guiMat
			} else {
				tagMaterials[i] = ResolveConfigMaterial(matDef)
			}
		}
		if tagMaterials[i] == nil {
			tagMaterials[i] = config.NewConfigMaterial("lwo.part", []string{tagName}, config.MaterialKindSolid)
		}
		ConfigureThingMaterial(tagMaterials[i], tagName, t.materials.Resolve(tagName))
	}

	var triangles []config.Model3DEntryTriangle

	for polyIdx, poly := range lwd.Polygons {
		if len(poly) < 3 {
			continue
		}

		var mat *config.Material
		if tagIdx, ok := lwd.PolyTags[polyIdx]; ok && tagIdx >= 0 && tagIdx < len(tagMaterials) {
			mat = tagMaterials[tagIdx]
		}
		if mat == nil && len(tagMaterials) > 0 {
			mat = tagMaterials[0]
		}

		getVertex := func(vi int) config.Model3DEntryVertex {
			var p geometry.XYZ
			if vi >= 0 && vi < len(lwd.Points) {
				p = lwd.Points[vi]
			}
			var u, v float32
			if uv, ok := lwd.VMadUVs[proc.UVKey{Poly: polyIdx, Vert: vi}]; ok {
				u, v = uv[0], uv[1]
			} else if uv, ok = lwd.VMapUVs[vi]; ok {
				u, v = uv[0], uv[1]
			}
			return config.Model3DEntryVertex{Pos: p, U: u, V: v}
		}

		// Triangulate convex polygons (triangle fan)
		// Reverse winding (v0, v2, v1) due to coordinate system conversion (left-handed to right-handed)
		v0 := getVertex(poly[0])
		for i := 1; i < len(poly)-1; i++ {
			v1 := getVertex(poly[i])
			v2 := getVertex(poly[i+1])

			tri := config.NewModel3DEntryTriangle(mat)
			tri.Vertices[0] = v0
			tri.Vertices[1] = v2
			tri.Vertices[2] = v1
			triangles = append(triangles, tri)
		}
	}

	if len(triangles) == 0 {
		return nil, fmt.Errorf("no triangles generated from LWO")
	}

	cModel := config.NewModel3DEntry(1, []string{"idle"}, []string{"death", "die"})
	cModel.Frames[0] = config.NewModel3DEntryFrame(triangles)
	cModel.ActionDefinitions = []string{"idle"}
	cModel.ActionIntervals = [][2]int{{0, 0}}

	cModel.Center([]string{"stand", "idle", "legs_idle", "hover"})

	return cModel, nil
}
