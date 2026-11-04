package q2

import (
	"fmt"
	"strconv"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes is a structure for managing 3D spatial data partitioning, identified by an index and volumes count.
type Volumes struct {
	vIdx         string
	volumesCount float64
}

// NewVolumes creates and initializes a new Volumes instance with the provided index and volumes count.
func NewVolumes(mIdx int, volumesCount float64) *Volumes {
	return &Volumes{
		vIdx:         strconv.Itoa(mIdx),
		volumesCount: volumesCount,
	}
}

// Create processes a list of RawFace instances, organizes them into spatial clusters, and converts them into Volumes.
func (vs *Volumes) Create(faces []*lumps.RawFace2) []*config.Volume {
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

func (vs *Volumes) doBuild(faces []*lumps.RawFace2, addTriangle func(tri []geometry.XYZ, triUvs [][2]float64, material *config.Material, matName string)) {
	for _, v := range faces {
		animKind := TextInfo2ToMaterialKind(v.Info)
		material := config.NewConfigMaterial("volume", []string{v.TexName}, animKind)
		ConfigureFaceMaterial(material, v.TexName, v.Info)
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
