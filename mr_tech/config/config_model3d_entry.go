package config

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Model3DEntryVertex represents a single vertex in an Model3DEntry 3D model with position and texture coordinates.
type Model3DEntryVertex struct {
	Pos       geometry.XYZ
	U         float32
	V         float32
	Normal    geometry.XYZ
	HasNormal bool
}

// Model3DEntryTriangle represents a triangular mesh with 3 vertices and an associated material.
type Model3DEntryTriangle struct {
	Vertices [3]Model3DEntryVertex
	Material *Material
}

// NewModel3DEntryTriangle creates a new Model3DEntryTriangle with the specified material and initializes its vertices to default values.
func NewModel3DEntryTriangle(material *Material) Model3DEntryTriangle {
	tri := Model3DEntryTriangle{
		Material: material,
	}
	return tri
}

// Model3DTag represents a coordinate attachment tag with spatial origin and 3x3 orientation axes.
type Model3DTag struct {
	Origin geometry.XYZ
	Axis   [3]geometry.XYZ
}

// NewModel3DTag creates a new Model3DTag with the specified origin and orientation axis.
func NewModel3DTag(origin geometry.XYZ, axis [3]geometry.XYZ) Model3DTag {
	return Model3DTag{
		Origin: origin,
		Axis:   axis,
	}
}

// Model3DEntryFrame represents a collection of triangles that define a single frame in an Model3DEntry animation sequence.
type Model3DEntryFrame struct {
	Triangles []Model3DEntryTriangle
	Tags      map[string]Model3DTag
}

// NewModel3DEntryFrame creates a new Model3DEntryFrame with the specified list of Model3DEntryTriangle structures.
func NewModel3DEntryFrame(triangles []Model3DEntryTriangle) Model3DEntryFrame {
	return Model3DEntryFrame{
		Triangles: triangles,
		Tags:      make(map[string]Model3DTag),
	}
}

// Model3DEntry represents a structure holding animation frames, action definitions, and corresponding action intervals.
type Model3DEntry struct {
	Frames            []Model3DEntryFrame
	ActionClamp       []string
	ActionDefinitions []string
	ActionIntervals   [][2]int
	GroupSize         int
}

// NewModel3DEntry creates a new Model3DEntry instance with the specified number of frames and initializes it using the provided frame names.
func NewModel3DEntry(numFrames int, frameNames []string, actionClamp []string) *Model3DEntry {
	m := &Model3DEntry{
		Frames: make([]Model3DEntryFrame, numFrames),
	}
	for i := 0; i < numFrames; i++ {
		m.Frames[i].Tags = make(map[string]Model3DTag)
	}
	m.ActionClamp = actionClamp
	m.GroupSize = 6
	m.compute(frameNames)
	return m
}

// compute processes the given frameNames, grouping frames into intervals based on their base names and populating relevant fields.
func (m *Model3DEntry) compute(frameNames []string) {
	if len(frameNames) == 0 {
		return
	}
	currentBase, counter := m.GetBaseName(frameNames[0])
	startIdx := 0
	for i := 1; i <= len(frameNames); i++ {
		var base string
		update := false
		if i < len(frameNames) {
			var nc int
			base, nc = m.GetBaseName(frameNames[i])
			if base != currentBase {
				update = true
			} else if nc != counter+1 {
				update = true
			}
			counter = nc
		}
		if i == len(frameNames) {
			update = true
		}
		if update {
			if startIdx < 0 || startIdx > len(m.Frames) {
				fmt.Println("startIdx out of range")
				continue
			}
			endIdx := i - 1
			if endIdx < 0 || endIdx >= len(m.Frames) {
				fmt.Println("endIdx out of range")
				continue
			}
			m.ActionIntervals = append(m.ActionIntervals, [2]int{startIdx, i - 1})
			name := currentBase + strconv.Itoa(len(m.ActionDefinitions))
			m.ActionDefinitions = append(m.ActionDefinitions, name)
			if i < len(frameNames) {
				currentBase = base
				startIdx = i
			}
		}
	}
}

// GetBaseName extracts the base name from a string by removing trailing numeric characters and returns the result.
func (m *Model3DEntry) GetBaseName(fn string) (string, int) {
	pos := -1
	for x := 0; x < len(fn); x++ {
		t := fn[x]
		if t >= '0' && t <= '9' {
			pos = x
			break
		}
	}
	if pos < 0 {
		return fn, 0
	}
	a := fn[:pos]
	num := fn[pos:]
	val, err := strconv.Atoi(num)
	if err != nil {
		fmt.Println("Error converting string to int:", err)
		return a, 0
	}
	return a, val
}

// Center repositions the 3D model by centering its vertices around the origin of the coordinate system.
func (m *Model3DEntry) Center(referenceFrames []string) {
	_, _ = m.CenterWithBounds(referenceFrames)
}

// BuildTagMap generates a map of unique tag names to sequential integer indices from the frames of the 3D model.
func (m *Model3DEntry) BuildTagMap() map[string]int {
	tagMap := make(map[string]int)
	tagCounter := 0
	for _, cfgFrame := range m.Frames {
		for tagName := range cfgFrame.Tags {
			if _, exists := tagMap[tagName]; !exists {
				tagMap[tagName] = tagCounter
				tagCounter++
			}
		}
	}
	return tagMap
}

// resolveReferenceFrame returns the start frame index of the first action matching referenceFrames, or 0.
func (m *Model3DEntry) resolveReferenceFrame(referenceFrames []string) int {
	if len(m.ActionDefinitions) > 0 && len(m.ActionIntervals) == len(m.ActionDefinitions) {
		for i, name := range m.ActionDefinitions {
			base, _ := m.GetBaseName(strings.ToLower(strings.TrimSpace(name)))
			for _, t := range referenceFrames {
				if strings.Contains(base, t) {
					if idx := m.ActionIntervals[i][0]; idx >= 0 && idx < len(m.Frames) {
						return idx
					}
				}
			}
		}
	}
	return 0
}

func (m *Model3DEntry) frameBounds(frameIdx int) (minX, minY, minZ, maxX, maxY, maxZ float64, ok bool) {
	minX, minY, minZ = math.Inf(1), math.Inf(1), math.Inf(1)
	maxX, maxY, maxZ = math.Inf(-1), math.Inf(-1), math.Inf(-1)
	for _, tri := range m.Frames[frameIdx].Triangles {
		for _, vertex := range tri.Vertices {
			p := vertex.Pos
			ok = true
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
			if p.Z < minZ {
				minZ = p.Z
			}
			if p.Z > maxZ {
				maxZ = p.Z
			}
		}
	}
	return minX, minY, minZ, maxX, maxY, maxZ, ok
}

// CenterWithBounds centers the 3D model around the origin and returns the original world center offset and bounding box size.
func (m *Model3DEntry) CenterWithBounds(referenceFrames []string) (geometry.XYZ, geometry.XYZ) {
	if m == nil || len(m.Frames) == 0 {
		return geometry.XYZ{}, geometry.XYZ{}
	}

	referenceFrame := m.resolveReferenceFrame(referenceFrames)
	minX, minY, minZ, maxX, maxY, maxZ, hasVertices := m.frameBounds(referenceFrame)
	if !hasVertices && referenceFrame != 0 {
		minX, minY, minZ, maxX, maxY, maxZ, hasVertices = m.frameBounds(0)
	}
	if !hasVertices {
		return geometry.XYZ{}, geometry.XYZ{}
	}

	centerX := (minX + maxX) * 0.5
	centerY := (minY + maxY) * 0.5
	centerZ := (minZ + maxZ) * 0.5

	for frameIdx := range m.Frames {
		for triIdx := range m.Frames[frameIdx].Triangles {
			for vertexIdx := range m.Frames[frameIdx].Triangles[triIdx].Vertices {
				p := &m.Frames[frameIdx].Triangles[triIdx].Vertices[vertexIdx].Pos
				p.X -= centerX
				p.Y -= centerY
				p.Z -= centerZ
			}
		}
		for tagName, tag := range m.Frames[frameIdx].Tags {
			tag.Origin.X -= centerX
			tag.Origin.Y -= centerY
			tag.Origin.Z -= centerZ
			m.Frames[frameIdx].Tags[tagName] = tag
		}
	}

	return geometry.XYZ{X: centerX, Y: centerY, Z: centerZ}, geometry.XYZ{X: maxX - minX, Y: maxY - minY, Z: maxZ - minZ}
}

// GetSize returns zero values when no vertices are present or computed dimensions do not exceed 1.0.
func (m *Model3DEntry) GetSize() (float64, float64) {
	var radius, height float64
	if len(m.Frames) > 0 {
		minX, minY, minZ := math.Inf(1), math.Inf(1), math.Inf(1)
		maxX, maxY, maxZ := math.Inf(-1), math.Inf(-1), math.Inf(-1)
		hasV := false
		for _, tri := range m.Frames[0].Triangles {
			for _, v := range tri.Vertices {
				hasV = true
				if v.Pos.X < minX {
					minX = v.Pos.X
				}
				if v.Pos.X > maxX {
					maxX = v.Pos.X
				}
				if v.Pos.Y < minY {
					minY = v.Pos.Y
				}
				if v.Pos.Y > maxY {
					maxY = v.Pos.Y
				}
				if v.Pos.Z < minZ {
					minZ = v.Pos.Z
				}
				if v.Pos.Z > maxZ {
					maxZ = v.Pos.Z
				}
			}
		}
		if hasV {
			modelH := maxZ - minZ
			if modelH > 1.0 {
				height = modelH
			}
			modelR := math.Max(maxX-minX, maxY-minY) * 0.5
			if modelR > 1.0 {
				radius = modelR
			}
		}
	}
	return radius, height
}

type modelVertKey struct {
	x, y, z int64
}

func makeModelVertKey(p geometry.XYZ) modelVertKey {
	return modelVertKey{
		x: int64(math.Round(p.X * 100.0)),
		y: int64(math.Round(p.Y * 100.0)),
		z: int64(math.Round(p.Z * 100.0)),
	}
}

// NormalizeNormalsAndWinding ensures consistent CCW outward winding and populates
// crease-aware smooth vertex normals for vertices without explicit artist normals.
func NormalizeNormalsAndWinding(tris []Model3DEntryTriangle, checkSignedVolume bool, cosCrease float64) {
	nTris := len(tris)
	if nTris == 0 {
		return
	}

	triCross := func(t *Model3DEntryTriangle) (cx, cy, cz float64) {
		p0, p1, p2 := t.Vertices[0].Pos, t.Vertices[1].Pos, t.Vertices[2].Pos
		e1x, e1y, e1z := p1.X-p0.X, p1.Y-p0.Y, p1.Z-p0.Z
		e2x, e2y, e2z := p2.X-p0.X, p2.Y-p0.Y, p2.Z-p0.Z
		return e1y*e2z - e1z*e2y, e1z*e2x - e1x*e2z, e1x*e2y - e1y*e2x
	}

	anyExplicit := false
	for i := range tris {
		t := &tris[i]
		hasTriNormal := t.Vertices[0].HasNormal || t.Vertices[1].HasNormal || t.Vertices[2].HasNormal
		if hasTriNormal {
			anyExplicit = true
			avgX := t.Vertices[0].Normal.X + t.Vertices[1].Normal.X + t.Vertices[2].Normal.X
			avgY := t.Vertices[0].Normal.Y + t.Vertices[1].Normal.Y + t.Vertices[2].Normal.Y
			avgZ := t.Vertices[0].Normal.Z + t.Vertices[1].Normal.Z + t.Vertices[2].Normal.Z
			cx, cy, cz := triCross(t)
			if cx*avgX+cy*avgY+cz*avgZ < 0 {
				t.Vertices[1], t.Vertices[2] = t.Vertices[2], t.Vertices[1]
			}
		}
	}

	if !anyExplicit && checkSignedVolume {
		var cxSum, cySum, czSum float64
		for i := range tris {
			for k := range 3 {
				cxSum += tris[i].Vertices[k].Pos.X
				cySum += tris[i].Vertices[k].Pos.Y
				czSum += tris[i].Vertices[k].Pos.Z
			}
		}
		invVerts := 1.0 / float64(nTris*3)
		centroidX, centroidY, centroidZ := cxSum*invVerts, cySum*invVerts, czSum*invVerts

		var signedVol float64
		for i := range tris {
			t := &tris[i]
			p0 := t.Vertices[0].Pos
			nx, ny, nz := triCross(t)
			signedVol += (p0.X-centroidX)*nx + (p0.Y-centroidY)*ny + (p0.Z-centroidZ)*nz
		}
		if signedVol < -1e-6 {
			for i := range tris {
				t := &tris[i]
				t.Vertices[1], t.Vertices[2] = t.Vertices[2], t.Vertices[1]
			}
		}
	}

	// Precompute area-weighted and unit face normals for all triangles.
	type faceNData struct {
		areaX, areaY, areaZ float64
		unitX, unitY, unitZ float64
		valid               bool
	}
	faceNormals := make([]faceNData, nTris)
	needsSmooth := false
	for i := range tris {
		t := &tris[i]
		ax, ay, az := triCross(t)
		l := math.Sqrt(ax*ax + ay*ay + az*az)
		if l > 1e-12 {
			invL := 1.0 / l
			faceNormals[i] = faceNData{
				areaX: ax, areaY: ay, areaZ: az,
				unitX: ax * invL, unitY: ay * invL, unitZ: az * invL,
				valid: true,
			}
		}
		if !t.Vertices[0].HasNormal || !t.Vertices[1].HasNormal || !t.Vertices[2].HasNormal {
			needsSmooth = true
		}
	}

	var vertHead map[modelVertKey]int32
	var vertNext []int32
	if needsSmooth {
		vertHead = make(map[modelVertKey]int32, nTris)
		vertNext = make([]int32, nTris*3)
		for i := range tris {
			if !faceNormals[i].valid {
				continue
			}
			t := &tris[i]
			for k := range 3 {
				e := int32(i*3 + k)
				key := makeModelVertKey(t.Vertices[k].Pos)
				if prev, ok := vertHead[key]; ok {
					vertNext[e] = prev
				} else {
					vertNext[e] = -1
				}
				vertHead[key] = e
			}
		}
	}

	for i := range tris {
		t := &tris[i]
		fd := faceNormals[i]
		for k := range 3 {
			if t.Vertices[k].HasNormal {
				n := t.Vertices[k].Normal
				nl := math.Sqrt(n.X*n.X + n.Y*n.Y + n.Z*n.Z)
				if nl > 1e-12 {
					inv := 1.0 / nl
					t.Vertices[k].Normal = geometry.XYZ{X: n.X * inv, Y: n.Y * inv, Z: n.Z * inv}
				} else if fd.valid {
					t.Vertices[k].Normal = geometry.XYZ{X: fd.unitX, Y: fd.unitY, Z: fd.unitZ}
				}
				continue
			}
			if !fd.valid {
				t.Vertices[k].Normal = geometry.XYZ{X: 0, Y: 0, Z: 1}
				t.Vertices[k].HasNormal = true
				continue
			}
			key := makeModelVertKey(t.Vertices[k].Pos)
			e, ok := vertHead[key]
			var sx, sy, sz float64
			for ok && e >= 0 {
				ofd := faceNormals[int(e)/3]
				if ofd.valid && fd.unitX*ofd.unitX+fd.unitY*ofd.unitY+fd.unitZ*ofd.unitZ >= cosCrease {
					sx += ofd.areaX
					sy += ofd.areaY
					sz += ofd.areaZ
				}
				e = vertNext[e]
			}
			sl := math.Sqrt(sx*sx + sy*sy + sz*sz)
			if sl > 1e-12 {
				inv := 1.0 / sl
				t.Vertices[k].Normal = geometry.XYZ{X: sx * inv, Y: sy * inv, Z: sz * inv}
			} else {
				t.Vertices[k].Normal = geometry.XYZ{X: fd.unitX, Y: fd.unitY, Z: fd.unitZ}
			}
			t.Vertices[k].HasNormal = true
		}
	}
}
