package model

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Compiler3d is a type responsible for compiling 3D volumes and their associated materials and lighting configurations.
type Compiler3d struct {
	cfg       *config.Root
	materials *Materials
}

// NewCompiler3d creates and returns a new instance of the Compiler3d type.
func NewCompiler3d(cfg *config.Root, materials *Materials) *Compiler3d {
	return &Compiler3d{
		cfg:       cfg,
		materials: materials,
	}
}

// Compile processes the 3D configuration, generating volumes and lights, and returns them along with any encountered error.
func (r *Compiler3d) Compile() ([]*Volume, []*Light, error) {
	if len(r.cfg.Volumes) == 0 {
		return nil, nil, nil
	}
	allVolumes := r.compile3d()
	return allVolumes, nil, nil
}

// compile3d processes a list of 3D volumes, triangulates their geometry, applies materials, and organizes them into a container.
func (r *Compiler3d) compile3d() []*Volume {
	volumes, anim := r.cfg.Volumes, r.materials
	totalFaces := 0
	var container []*Volume
	var fixFaces []*Face
	modelSectorId := 0
	facesTree := aabb.NewTree(1024, 0.001)
	for _, cv := range volumes {
		// cv.Id and cv.Tag come from the BSP parser
		volume := NewVolumeConcrete(modelSectorId, cv.Id, cv.Tag)
		modelSectorId++
		for _, cf := range cv.Faces {
			pts := cf.Points
			pLen := len(pts)
			if pLen < 3 {
				fmt.Println("wrong points configuration", cf.Points)
				continue
			}
			material := anim.GetMaterial(cf.Material)
			// Robust polygon decomposition (Supports concave N-Gons)
			triangles := geometry.Triangulate3d(pts)
			for _, t := range triangles {
				if len(t) != 3 {
					fmt.Println("wrong tri", t)
					continue
				}
				tri := [3]geometry.XYZ{t[0], t[1], t[2]}
				face := NewFace(tri, cf.Tag, material)

				// Map UVs if provided by the config face
				if len(cf.UVs) > 0 {
					var triUvs [3][2]float64
					for k := 0; k < 3; k++ {
						for idx, pt := range cf.Points {
							if pt.X == t[k].X && pt.Y == t[k].Y && pt.Z == t[k].Z {
								if len(cf.UVs) > idx {
									triUvs[k] = cf.UVs[idx]
								}
								break
							}
						}
					}
					face.SetUV(triUvs[0][0], triUvs[0][1], triUvs[1][0], triUvs[1][1], triUvs[2][0], triUvs[2][1])
					face.LockUV(true)
				}

				volume.AddFace(face)
				fixFaces = append(fixFaces, face)
				facesTree.InsertObject(face)
				totalFaces++
			}
		}
		// Initialize default light (will be calculated later in compileVolumesLights)
		volume.light = NewLight()
		volume.Rebuild()
		container = append(container, volume)
	}
	return container
}
