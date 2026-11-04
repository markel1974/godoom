package q3

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes represents a structure that works with Shaders to facilitate geometry processing and face generation.
type Volumes struct {
	shaders      *Shaders
	volumesCount float64
}

// NewVolumes initializes and returns a new Volumes instance with the provided Shaders dependency and volumes count.
func NewVolumes(shaders *Shaders, volumesCount float64) *Volumes {
	return &Volumes{
		shaders:      shaders,
		volumesCount: volumesCount,
	}
}

// Create generates and groups faces into 3D volumes based on spatial clustering, material properties, and shader configurations.
func (f *Volumes) Create(mIdx int, faces []*lumps.RawFace3) ([]*config.Volume, error) {
	var centroids []geometry.XYZ

	preFlight := func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string) {
		cx := (tri[0].X + tri[1].X + tri[2].X) / 3.0
		cy := (tri[0].Y + tri[1].Y + tri[2].Y) / 3.0
		cz := (tri[0].Z + tri[1].Z + tri[2].Z) / 3.0
		centroids = append(centroids, geometry.XYZ{X: cx, Y: cy, Z: cz})
	}

	f.doBuild(faces, preFlight)

	totalTriangles := len(centroids)
	if totalTriangles == 0 {
		return nil, nil
	}

	targetVolumes := int(f.volumesCount)
	if targetVolumes <= 0 {
		targetVolumes = 128
	}
	if targetVolumes > totalTriangles {
		targetVolumes = totalTriangles
	}

	cs := geometry.NewClusters()
	clusters := cs.Create(centroids, targetVolumes, totalTriangles)
	numVolumes := len(clusters)
	fmt.Printf("Preflight: created exactly %d volumes for %d triangles (target: %d)\n", numVolumes, totalTriangles, targetVolumes)

	clusterMap := make([]int, totalTriangles)
	volumes := make([]*config.Volume, numVolumes)
	vIdx := strconv.Itoa(mIdx)

	for cIdx, c := range clusters {
		chunkId := fmt.Sprintf("quake_world_%s_volume_%d", vIdx, cIdx)
		vol := config.NewConfigVolume(chunkId, "quake_bsp_chunk")
		vol.Faces = make([]*config.Face, 0, len(c.Indices))
		volumes[cIdx] = vol
		for _, triIdx := range c.Indices {
			clusterMap[triIdx] = cIdx
		}
	}

	triCounter := 0
	addTriangle := func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string) {
		if triCounter >= totalTriangles {
			fmt.Println("max triangles reached", triCounter, "total", totalTriangles)
			return
		}
		volIdx := clusterMap[triCounter]
		triCounter++
		volumes[volIdx].Faces = append(volumes[volIdx].Faces, config.NewConfigFace(tri, triUvs, material, matName))
	}

	f.doBuild(faces, addTriangle)

	return volumes, nil
}

func (f *Volumes) doBuild(faces []*lumps.RawFace3, addTriangle func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string)) {
	for _, v := range faces {
		animKind := config.MaterialKindSolid
		texNameLC := strings.ToLower(v.TexName)
		if v.Info.IsSky() {
			animKind = config.MaterialKindSky
		} else {
			if f.shaders.IsLiquid(texNameLC) {
				animKind = config.MaterialKindWater
			}
		}

		var material *config.Material
		scrollU, scrollV := f.shaders.GetScroll(texNameLC)

		if animMap := f.shaders.GetAnimMap(texNameLC); len(animMap) > 0 {
			animMapLC := make([]string, len(animMap))
			for i, frame := range animMap {
				animMapLC[i] = strings.ToLower(frame)
			}
			material = config.NewConfigMaterial(v.TexName, animMapLC, animKind)
			material.ScrollU = float32(scrollU)
			material.ScrollV = float32(scrollV)
		} else if v.Info.IsSky() {
			// Sky textures might have a specific editor image, or we try to extract the map
			if diffMap := f.shaders.GetDiffuseMap(texNameLC); diffMap != "" {
				material = config.NewConfigMaterial(v.TexName, []string{strings.ToLower(diffMap)}, animKind)
				material.ScrollU = float32(scrollU)
				material.ScrollV = float32(scrollV)
			} else {
				material = config.NewConfigMaterial(v.TexName, []string{texNameLC}, animKind)
				material.ScrollU = float32(scrollU)
				material.ScrollV = float32(scrollV)
			}
		} else {
			// For all other surfaces, extract the real texture map from the shader stages
			if diffMap := f.shaders.GetDiffuseMap(texNameLC); diffMap != "" {
				material = config.NewConfigMaterial(v.TexName, []string{strings.ToLower(diffMap)}, animKind)
				material.ScrollU = float32(scrollU)
				material.ScrollV = float32(scrollV)
			} else {
				material = config.NewConfigMaterial(v.TexName, []string{texNameLC}, animKind)
				material.ScrollU = float32(scrollU)
				material.ScrollV = float32(scrollV)
			}
		}

		f.shaders.MaterialBind(texNameLC, material)

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
			addTriangle(tri, triUvs, material, v.TexName)
		}
	}
}
