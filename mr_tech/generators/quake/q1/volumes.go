package q1

import (
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes represents a container for cluster-based 3D volumes with unique identifiers and configurable volumes count.
type Volumes struct {
	volumesCount float64
	vIdx         string
}

// NewVolumes initializes a Volumes instance with specified volumes count and a unique identifier based on the given index.
func NewVolumes(mIdx int, volumesCount float64) *Volumes {
	return &Volumes{
		volumesCount: volumesCount,
		vIdx:         strconv.Itoa(mIdx),
	}
}

// Generate processes raw face data to create volumes and faces, assigning materials and clustering into balanced spatial volumes.
func (vs *Volumes) Generate(faces []*lumps.RawFace) []*config.Volume {
	var centroids []geometry.XYZ

	preFlight := func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string) {
		cx := (tri[0].X + tri[1].X + tri[2].X) / 3.0
		cy := (tri[0].Y + tri[1].Y + tri[2].Y) / 3.0
		cz := (tri[0].Z + tri[1].Z + tri[2].Z) / 3.0
		centroids = append(centroids, geometry.XYZ{X: cx, Y: cy, Z: cz})
	}

	vs.doBuild(faces, preFlight)

	totalTriangles := len(centroids)
	if totalTriangles == 0 {
		return nil
	}

	targetVolumes := int(vs.volumesCount)
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

	for cIdx, c := range clusters {
		chunkId := fmt.Sprintf("quake_world_%s_volume_%d", vs.vIdx, cIdx)
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

	vs.doBuild(faces, addTriangle)

	return volumes
}

func (vs *Volumes) doBuild(faces []*lumps.RawFace, addTriangle func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string)) {
	for _, v := range faces {
		isLiquid := false
		animKind := config.MaterialKindSolid
		velX, velY := 0.0, 0.0
		if v.IsSky {
			animKind = config.MaterialKindSky
			velX, velY = 0.05, 0.05
		} else if len(v.TexName) > 0 && v.TexName[0] == '*' {
			isLiquid = true
		}

		if isLiquid {
			tName := v.TexName
			if len(tName) > 0 && tName[0] == '*' {
				tName = tName[1:]
			}
			switch tName {
			case "lava":
				animKind = config.MaterialKindLava
			case "slime":
				animKind = config.MaterialKindSlime
			default:
				animKind = config.MaterialKindWater
			}
		}
		material := config.NewConfigMaterial("volume", []string{v.TexName}, animKind)
		material.U = velX
		material.V = velY
		ConfigureFaceMaterial(material, v.TexName, v.IsSky)
		triangles := lumps.TriangulateConvex3d(v.Points)

		for _, rawTri := range triangles {
			var rawTriUvs [][2]float64
			if len(v.UVs) > 0 {
				rawTriUvs = make([][2]float64, 3)
				for k := 0; k < 3; k++ {
					pos := rawTri[k]
					for idx, pt := range v.Points {
						if pt.X == pos.X && pt.Y == pos.Y && pt.Z == pos.Z {
							if len(v.UVs) > idx {
								uv := v.UVs[idx]
								uv[0] *= 1.0
								uv[1] *= 1.0
								rawTriUvs[k] = uv
							}
							break
						}
					}
				}
			}
			addTriangle([]geometry.XYZ{rawTri[0], rawTri[1], rawTri[2]}, rawTriUvs, material, v.TexName)
		}
	}
}
