package vertices_element_3d

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/materials"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// PackedVec3 represents a 3D vector with x, y, and z components using compact float32 storage.
type PackedVec3 struct {
	x, y, z float32
}

// ToPackedVec3 converts a geometry.XYZ vector to a PackedVec3 format by casting its components to float32.
func ToPackedVec3(v geometry.XYZ) PackedVec3 {
	return PackedVec3{x: float32(v.X), y: float32(v.Y), z: float32(v.Z)}
}

// ToXYZ converts a PackedVec3 to a geometry.XYZ by mapping its float32 components to float64 components.
func (p PackedVec3) ToXYZ() geometry.XYZ {
	return geometry.XYZ{X: float64(p.x), Y: float64(p.y), Z: float64(p.z)}
}

// TemplateTag represents a label associated with an origin point and a set of axes in a 3D space.

// ResolveFrameMaterials maps triangles to materials using the provided configuration and materials list, and returns the result.
func ResolveFrameMaterials(cfgTris []config.Model3DEntryTriangle, mats *materials.Materials, dst []*textures.Material) []*textures.Material {
	if cap(dst) < len(cfgTris) {
		dst = make([]*textures.Material, len(cfgTris))
	} else {
		dst = dst[:len(cfgTris)]
	}
	for i, tri := range cfgTris {
		if tri.Material != nil {
			dst[i] = mats.GetMaterial(tri.Material)
		} else {
			dst[i] = nil
		}
	}
	return dst
}

// CheckUniformTopology validates if all frames in a 3D model have identical topology, materials, and UV coordinates.
// It returns whether the topology is uniform, base materials, base UV coordinates, and the maximum number of triangles.
func CheckUniformTopology(model *config.Model3DEntry, mats *materials.Materials) (bool, []*textures.Material, [][3]float32, [][3]float32, int) {
	maxFaces := 0
	for _, cfgFrame := range model.Frames {
		if len(cfgFrame.Triangles) > maxFaces {
			maxFaces = len(cfgFrame.Triangles)
		}
	}
	if len(model.Frames) == 0 {
		return true, nil, nil, nil, 0
	}
	baseTris := model.Frames[0].Triangles
	baseLen := len(baseTris)
	baseMats := ResolveFrameMaterials(baseTris, mats, nil)
	baseU := make([][3]float32, baseLen)
	baseV := make([][3]float32, baseLen)
	for i, tri := range baseTris {
		baseU[i] = [3]float32{tri.Vertices[0].U, tri.Vertices[1].U, tri.Vertices[2].U}
		baseV[i] = [3]float32{tri.Vertices[0].V, tri.Vertices[1].V, tri.Vertices[2].V}
	}
	uniform := true
	var scratchMats []*textures.Material
	for fIdx := 1; fIdx < len(model.Frames); fIdx++ {
		fTris := model.Frames[fIdx].Triangles
		if len(fTris) != baseLen {
			uniform = false
			break
		}
		scratchMats = ResolveFrameMaterials(fTris, mats, scratchMats)
		for i, tri := range fTris {
			if scratchMats[i] != baseMats[i] {
				uniform = false
				break
			}
			for k := range 3 {
				if tri.Vertices[k].U != baseU[i][k] || tri.Vertices[k].V != baseV[i][k] {
					uniform = false
					break
				}
			}
			if !uniform {
				break
			}
		}
		if !uniform {
			break
		}
	}
	if !uniform {
		return false, nil, nil, nil, maxFaces
	}
	return true, baseMats, baseU, baseV, maxFaces
}

// AppendDedupVec appends a PackedVec3 to the slice if it is not already present and updates its index in the dedup map.
// It returns the index of the PackedVec3 in the slice.
func AppendDedupVec(vecs *[]PackedVec3, dedup map[PackedVec3]uint32, v PackedVec3) uint32 {
	if idx, ok := dedup[v]; ok {
		return idx
	}
	idx := uint32(len(*vecs))
	*vecs = append(*vecs, v)
	dedup[v] = idx
	return idx
}

type TemplateTag struct {
	idx    int
	origin geometry.XYZ
	axis   [3]geometry.XYZ
}

// TemplateFrame represents a 3D geometry frame including vertex data, normals, topology indices, and spatial properties.
type TemplateFrame struct {
	faceCount   int
	positions   []PackedVec3
	normals     []PackedVec3
	posIdx      [][3]uint32
	normIdx     [][3]uint32
	activeEdges []uint8
	materials   []*textures.Material
	u           [][3]float32
	v           [][3]float32
	tags        []TemplateTag
	minX        float64
	minY        float64
	minZ        float64
	maxX        float64
	maxY        float64
	maxZ        float64
	w           float64
	h           float64
	d           float64
	cx          float64
	cy          float64
	cz          float64
	ixx         float64
	ixy         float64
	ixz         float64
	iyy         float64
	iyz         float64
	izz         float64
	geomHash    uint64
}

// ElementTemplate represents a 3D model template composed of multiple frames with geometry and topology information.
type ElementTemplate struct {
	numFrames       int
	maxFaces        int
	uniformTopology bool
	tagMap          map[string]int
	materials       []*textures.Material
	u               [][3]float32
	v               [][3]float32
	frames          []TemplateFrame
}

// NewElementTemplate constructs and returns a new ElementTemplate based on the provided 3D model and materials data.
// It initializes geometry, topology, materials, texture coordinates, and frame data for the template.
func NewElementTemplate(model *config.Model3DEntry, mats *materials.Materials) *ElementTemplate {
	tagMap := model.BuildTagMap()
	uniform, baseMats, baseU, baseV, maxFaces := CheckUniformTopology(model, mats)
	tpl := &ElementTemplate{
		numFrames:       len(model.Frames),
		maxFaces:        maxFaces,
		uniformTopology: uniform,
		tagMap:          tagMap,
		materials:       baseMats,
		u:               baseU,
		v:               baseV,
		frames:          make([]TemplateFrame, len(model.Frames)),
	}

	// Build geometry & summary once per frame using a single scratch Faces container.
	scratchFaces := volumes.NewFacesSized(nil, maxFaces, false)
	scratchFaces.SetAllowTessellate(false)
	for range maxFaces {
		scratchFaces.Add(volumes.NewFaceEmpty(nil, [3]float64{}, [3]float64{}))
	}

	posDedup := make(map[PackedVec3]uint32, maxFaces)
	normDedup := make(map[PackedVec3]uint32, maxFaces)

	for fIdx, cfgFrame := range model.Frames {
		nTris := len(cfgFrame.Triangles)
		var fMats []*textures.Material
		var fU, fV [][3]float32
		if uniform {
			fMats = baseMats
			fU = baseU
			fV = baseV
		} else {
			fMats = ResolveFrameMaterials(cfgFrame.Triangles, mats, nil)
			fU = make([][3]float32, nTris)
			fV = make([][3]float32, nTris)
			for i, tri := range cfgFrame.Triangles {
				fU[i] = [3]float32{tri.Vertices[0].U, tri.Vertices[1].U, tri.Vertices[2].U}
				fV[i] = [3]float32{tri.Vertices[0].V, tri.Vertices[1].V, tri.Vertices[2].V}
			}
		}

		for i, tri := range cfgFrame.Triangles {
			fc := scratchFaces.At(i)
			pts := [3]geometry.XYZ{tri.Vertices[0].Pos, tri.Vertices[1].Pos, tri.Vertices[2].Pos}
			u64 := [3]float64{float64(fU[i][0]), float64(fU[i][1]), float64(fU[i][2])}
			v64 := [3]float64{float64(fV[i][0]), float64(fV[i][1]), float64(fV[i][2])}
			hasN := tri.Vertices[0].HasNormal || tri.Vertices[1].HasNormal || tri.Vertices[2].HasNormal
			fc.InitTemplateSource(pts, fMats[i], u64, v64, hasN, tri.Vertices[0].Normal, tri.Vertices[1].Normal, tri.Vertices[2].Normal)
		}
		scratchFaces.SetCount(nTris)
		scratchFaces.RebuildGeometry()

		clear(posDedup)
		clear(normDedup)
		positions := make([]PackedVec3, 0, nTris/2+8)
		normals := make([]PackedVec3, 0, nTris/2+8)
		posIdx := make([][3]uint32, nTris)
		normIdx := make([][3]uint32, nTris)
		activeEdges := make([]uint8, nTris)

		for i := range nTris {
			fc := scratchFaces.At(i)
			pts := fc.GetTri().Points
			nx0, ny0, nz0, nx1, ny1, nz1, nx2, ny2, nz2 := fc.GetVertexNormalsFlat()
			posIdx[i] = [3]uint32{
				AppendDedupVec(&positions, posDedup, ToPackedVec3(pts[0])),
				AppendDedupVec(&positions, posDedup, ToPackedVec3(pts[1])),
				AppendDedupVec(&positions, posDedup, ToPackedVec3(pts[2])),
			}
			normIdx[i] = [3]uint32{
				AppendDedupVec(&normals, normDedup, PackedVec3{x: float32(nx0), y: float32(ny0), z: float32(nz0)}),
				AppendDedupVec(&normals, normDedup, PackedVec3{x: float32(nx1), y: float32(ny1), z: float32(nz1)}),
				AppendDedupVec(&normals, normDedup, PackedVec3{x: float32(nx2), y: float32(ny2), z: float32(nz2)}),
			}
			activeEdges[i] = fc.ActiveEdgeMask()
		}

		var tags []TemplateTag
		if len(cfgFrame.Tags) > 0 {
			tags = make([]TemplateTag, 0, len(cfgFrame.Tags))
			for tagName, tagData := range cfgFrame.Tags {
				tags = append(tags, TemplateTag{
					idx:    tagMap[tagName],
					origin: tagData.Origin,
					axis:   tagData.Axis,
				})
			}
		}

		minX, minY, minZ := scratchFaces.GetMin()
		maxX, maxY, maxZ := scratchFaces.GetMax()
		w, h, d := scratchFaces.GetSize()
		cx, cy, cz := scratchFaces.GetCentroid()
		ixx, ixy, ixz, iyy, iyz, izz := scratchFaces.GetInertia()

		tf := TemplateFrame{
			faceCount:   nTris,
			positions:   positions,
			normals:     normals,
			posIdx:      posIdx,
			normIdx:     normIdx,
			activeEdges: activeEdges,
			tags:        tags,
			minX:        minX, minY: minY, minZ: minZ,
			maxX: maxX, maxY: maxY, maxZ: maxZ,
			w: w, h: h, d: d,
			cx: cx, cy: cy, cz: cz,
			ixx: ixx, ixy: ixy, ixz: ixz,
			iyy: iyy, iyz: iyz, izz: izz,
			geomHash: scratchFaces.GetGeomHash(),
		}
		if !uniform {
			tf.materials = fMats
			tf.u = fU
			tf.v = fV
		}
		tpl.frames[fIdx] = tf
	}

	return tpl
}

// AllocateSlotVolume creates and returns a new dynamic volume with optional tessellation using the template's face data.
func (tpl *ElementTemplate) AllocateSlotVolume(frameIdx int, sharedTree *aabb.Tree, sharedSolver *collisions.Solver, allowTessellate bool) *volumes.Volume {
	vol := volumes.NewVolumeDynamicSharedSized(frameIdx, "", "thing", sharedTree, sharedSolver, tpl.maxFaces)
	vol.SetAllowTessellate(allowTessellate)
	for j := 0; j < tpl.maxFaces; j++ {
		var mat *textures.Material
		var u64, v64 [3]float64
		if tpl.uniformTopology && j < len(tpl.materials) {
			mat = tpl.materials[j]
			u64 = [3]float64{float64(tpl.u[j][0]), float64(tpl.u[j][1]), float64(tpl.u[j][2])}
			v64 = [3]float64{float64(tpl.v[j][0]), float64(tpl.v[j][1]), float64(tpl.v[j][2])}
		}
		fc := volumes.NewFaceEmpty(mat, u64, v64)
		fc.LockUV(true)
		vol.AddFace(fc)
	}
	return vol
}

// ApplyTemplateFrameToVolume applies a specific frame of the template to the given volume, updating its geometry and metadata.
func (tpl *ElementTemplate) ApplyTemplateFrameToVolume(frameIdx int, vol *volumes.Volume) {
	tf := &tpl.frames[frameIdx]
	faces := vol.GetFaces()
	nTris := tf.faceCount
	if !tpl.uniformTopology {
		for j := range nTris {
			fc := faces.At(j)
			fc.SetMaterial(tf.materials[j])
			fc.SetUV(
				float64(tf.u[j][0]), float64(tf.v[j][0]),
				float64(tf.u[j][1]), float64(tf.v[j][1]),
				float64(tf.u[j][2]), float64(tf.v[j][2]),
			)
		}
	}
	pos := tf.positions
	norm := tf.normals
	for j := range nTris {
		pi := tf.posIdx[j]
		ni := tf.normIdx[j]
		fc := faces.At(j)
		fc.ApplyTemplateGeometry(
			pos[pi[0]].ToXYZ(), pos[pi[1]].ToXYZ(), pos[pi[2]].ToXYZ(),
			norm[ni[0]].ToXYZ(), norm[ni[1]].ToXYZ(), norm[ni[2]].ToXYZ(),
			tf.activeEdges[j],
		)
	}
	for _, tg := range tf.tags {
		vol.SetVertexTag(tg.idx, tg.origin, tg.axis)
	}
	faces.ApplyCachedSummary(
		nTris,
		tf.minX, tf.minY, tf.minZ,
		tf.maxX, tf.maxY, tf.maxZ,
		tf.w, tf.h, tf.d,
		tf.cx, tf.cy, tf.cz,
		tf.ixx, tf.ixy, tf.ixz,
		tf.iyy, tf.iyz, tf.izz,
		tf.geomHash,
	)
	vol.SetModelId(frameIdx)
	vol.SetDirty(true)
}
