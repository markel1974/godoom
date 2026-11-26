package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// Volumes manages a collection of spatial volumes and cached frame center data.
type Volumes struct {
	vols          []*Volume
	centers       []geometry.XYZ
	recenteredLen int
	//maxFaces   int
	sharesTree bool
}

// NewVolumes creates a Volumes collection sized to hold v volume pointers.
func NewVolumes(v int) *Volumes {
	return &Volumes{
		vols:          make([]*Volume, v),
		centers:       nil,
		recenteredLen: 0,
		//maxFaces:   0,
		sharesTree: false,
	}
}

// NewVolumesFromSlice creates a Volumes collection wrapping the provided slice of volumes.
func NewVolumesFromSlice(vols []*Volume) *Volumes {
	return &Volumes{
		vols:          vols,
		centers:       nil,
		recenteredLen: 0,
	}
}

func (v *Volumes) Len() int {
	return len(v.vols)
}

// Set assigns vol to the volume slot at idx.
func (v *Volumes) Set(idx int, vol *Volume) {
	v.vols[idx] = vol
}

// Get returns the volume at idx.
func (v *Volumes) Get(idx int) *Volume {
	return v.vols[idx]
}

func (v *Volumes) Rebuild() {
	v.computeSharesTree()
	//v.computeMaxFaces()
}

// Add appends a non-nil volume to the collection.
func (v *Volumes) Add(vol *Volume) {
	if vol != nil {
		v.vols = append(v.vols, vol)
	}
	v.Rebuild()
}

// Append appends multiple volumes to the collection.
func (v *Volumes) Append(vols ...*Volume) {
	for _, vol := range vols {
		if vol != nil {
			v.vols = append(v.vols, vol)
		}
	}
	v.Rebuild()
}

// Slice returns the underlying slice of Volume pointers.
func (v *Volumes) Slice() []*Volume {
	if v == nil {
		return nil
	}
	return v.vols
}

// SetAllowTessellate sets whether all non-nil volumes may tessellate.
func (v *Volumes) SetAllowTessellate(t bool) {
	for _, vol := range v.vols {
		if vol != nil {
			vol.SetAllowTessellate(t)
		}
	}
}

// GetCenter returns the stored centering offset for idx, or a zero vector if idx is out of range.
func (v *Volumes) GetCenter(idx int) geometry.XYZ {
	if idx >= 0 && idx < v.recenteredLen {
		return v.centers[idx]
	}
	return geometry.XYZ{}
}

// IsRecentered reports whether frame center data is available.
func (v *Volumes) IsRecentered() bool {
	return v.recenteredLen > 0
}

// Recenter shifts each non-nil volume so its bounding-box center is at the origin and caches each offset.
// Recenter rebuilds updated geometry, preserving shared tree synchronization for callers to complete.
func (v *Volumes) Recenter() {
	vLen := len(v.vols)
	if len(v.centers) < vLen {
		v.centers = make([]geometry.XYZ, vLen*2)
	}
	v.recenteredLen = vLen
	for fIdx, vol := range v.vols {
		faces := vol.GetFaces()
		minX, minY, minZ := faces.GetMin()
		maxX, maxY, maxZ := faces.GetMax()
		mcx := (minX + maxX) * 0.5
		mcy := (minY + maxY) * 0.5
		mcz := (minZ + maxZ) * 0.5
		center := &v.centers[fIdx]
		center.X, center.Y, center.Z = mcx, mcy, mcz
		if math.Abs(mcx) > 1e-6 || math.Abs(mcy) > 1e-6 || math.Abs(mcz) > 1e-6 {
			for j := 0; j < faces.Len(); j++ {
				f := faces.At(j)
				p0X, p0Y, p0Z := f.Get0()
				p1X, p1Y, p1Z := f.Get1()
				p2X, p2Y, p2Z := f.Get2()
				f.Set0(p0X-mcx, p0Y-mcy, p0Z-mcz)
				f.Set1(p1X-mcx, p1Y-mcy, p1Z-mcz)
				f.Set2(p2X-mcx, p2Y-mcy, p2Z-mcz)
				f.Rebuild()
			}
			if v.sharesTree {
				// Callers with a shared AABB tree synchronize the tree for the active frame afterwards.
				vol.RebuildGeometry()
			} else {
				vol.Rebuild()
			}
		}
	}
}

// Deform applies an impulse deformation around impact to all volumes and reports whether any volume changed.
func (v *Volumes) Deform(impact, impulse geometry.XYZ, radius float64) bool {
	deformed := false
	for _, vol := range v.vols {
		if vol.Deform(impact, impulse, radius) {
			deformed = true
		}
	}
	if deformed && v.sharesTree {
		v.vols[0].SyncTree()
	}
	return deformed
}

// sharesTree reports whether at least two non-nil volumes in v point to the same AABB tree.
func (v *Volumes) computeSharesTree() {
	v.sharesTree = false
	var first *aabb.Tree
	for _, vol := range v.vols {
		t := vol.GetFacesTree()
		if t == nil {
			continue
		}
		if first == nil {
			first = t
		} else if t == first {
			v.sharesTree = true
			return
		}
	}
}

// GetFaceCount returns the maximum face count among the contained volumes.
//func (v *Volumes) computeMaxFaces() {
//	maxFaces := 0
//	for _, vol := range v.vols {
//		if c := vol.GetFaceCount(); c > maxFaces {
//			maxFaces = c
//		}
//	}
//	v.maxFaces = maxFaces
//}
