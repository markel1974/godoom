package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Volumes manages a collection of spatial volumes and cached frame center data.
type Volumes struct {
	vols       []*Volume
	centers    []geometry.XYZ
	centersLen int
}

// NewVolumes creates a Volumes collection sized to hold v volume pointers.
func NewVolumes(v int) *Volumes {
	return &Volumes{
		vols:       make([]*Volume, v),
		centers:    nil,
		centersLen: 0,
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

// GetFaceCount returns the maximum face count among the contained volumes.
func (v *Volumes) GetFaceCount() int {
	maxFaces := 0
	for _, vol := range v.vols {
		if vol != nil {
			if c := vol.GetFaceCount(); c > maxFaces {
				maxFaces = c
			}
		}
	}
	return maxFaces
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
	if idx >= 0 && idx < v.centersLen {
		return v.centers[idx]
	}
	return geometry.XYZ{}
}

// HasCenters reports whether frame center data is available.
func (v *Volumes) HasCenters() bool {
	return v.centersLen > 0
}

// BuildCenters recenter each volume around its bounding-box center and records the original centers.
func (v *Volumes) BuildCenters() {
	vLen := len(v.vols)
	if len(v.centers) < vLen {
		v.centers = make([]geometry.XYZ, vLen*2)
	}
	v.centersLen = vLen
	for fIdx, vol := range v.vols {
		if vol == nil {
			continue
		}
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
			vol.Rebuild()
		}
	}
}

// Deform applies an impulse deformation around impact to all volumes and reports whether any volume changed.
func (v *Volumes) Deform(impact, impulse geometry.XYZ, radius float64) bool {
	deformed := false
	for _, vol := range v.vols {
		if vol != nil && vol.Deform(impact, impulse, radius) {
			deformed = true
		}
	}
	return deformed
}
