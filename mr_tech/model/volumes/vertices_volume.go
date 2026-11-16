package volumes

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/articulated_dynamics"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
	"github.com/markel1974/godoom/mr_tech/physics/rigid_body"
)

// VerticesVolume wraps a single rigid Volume as an IVertices implementation,
// rotating its cached UI faces only when the underlying Entity orientation changes.
type VerticesVolume struct {
	volume   *Volume
	solver   *collisions.Solver
	ent      *rigid_body.Entity
	facesOut *Faces
	rot      *articulated_dynamics.TagAxis
	compiled bool
	lastQx   float64
	lastQy   float64
	lastQz   float64
	lastQw   float64
	minX     float64
	minY     float64
	minZ     float64
	maxX     float64
	maxY     float64
	maxZ     float64
}

// NewVerticesVolume creates an IVertices adapter backed directly by volume's geometry and solver.
func NewVerticesVolume(volume *Volume) *VerticesVolume {
	srcFaces := volume.GetFaces()
	n := srcFaces.Len()
	facesOut := NewFaces(n, true)
	for i := 0; i < n; i++ {
		src := srcFaces.At(i)
		facesOut.Set(i, NewFaceEmpty(src.material, src.u, src.v))
	}
	return &VerticesVolume{
		volume:   volume,
		solver:   volume.GetSolver(),
		ent:      volume.GetSolver().GetEntity(),
		facesOut: facesOut,
		rot:      articulated_dynamics.NewTagAxisIdentity(),
		lastQw:   1.0,
	}
}

// Setup binds the collision solver to thing, rebuilds volume bounds and inertia, and initializes face bounds.
func (v *VerticesVolume) Setup(thing IThing) error {
	if err := v.volume.Setup(thing); err != nil {
		return err
	}
	v.solver = v.volume.GetSolver()
	v.ent = v.solver.GetEntity()
	v.volume.Rebuild()
	v.compiled = false
	v.Compile(0)
	return nil
}

// Compile updates transformed UI faces and enclosing AABB when entity orientation has changed.
func (v *VerticesVolume) Compile(tick uint64) bool {
	ent := v.ent
	q := ent.GetQRotation()
	qx, qy, qz, qw := q.Get()

	if !v.compiled || qx != v.lastQx || qy != v.lastQy || qz != v.lastQz || qw != v.lastQw {
		v.compiled = true
		v.lastQx, v.lastQy, v.lastQz, v.lastQw = qx, qy, qz, qw
		isIdent := qx == 0.0 && qy == 0.0 && qz == 0.0 && qw == 1.0
		if !isIdent {
			v.rot.SetFromQuaternion(q)
		}

		minX, minY, minZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
		maxX, maxY, maxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

		srcFaces := v.volume.GetFaces()
		for i := 0; i < srcFaces.Len(); i++ {
			src := srcFaces.At(i)
			dst := v.facesOut.At(i)
			if isIdent {
				dst.CopyPoints(src)
			} else {
				v.rot.TransformPointsWithMatrix(dst.tri, src.tri, 0, 0, 0)
			}
			minX, minY, minZ, maxX, maxY, maxZ = dst.GetMinMax(minX, minY, minZ, maxX, maxY, maxZ)
		}
		if minX > maxX {
			minX, minY, minZ = 0, 0, 0
			maxX, maxY, maxZ = 0, 0, 0
		}
		v.minX, v.minY, v.minZ = minX, minY, minZ
		v.maxX, v.maxY, v.maxZ = maxX, maxY, maxZ
	}

	cx, cy, cz := ent.GetCenter()
	ent.GetAABB().Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	return false
}

// GetAABB returns the entity's current world-space AABB updated with rotated mesh extents.
func (v *VerticesVolume) GetAABB() *aabb.AABB {
	ent := v.ent
	b := ent.GetAABB()
	if v.minX <= v.maxX {
		cx, cy, cz := ent.GetCenter()
		b.Rebuild(cx+v.minX, cy+v.minY, cz+v.minZ, cx+v.maxX, cy+v.maxY, cz+v.maxZ)
	}
	return b
}

// GetSolver returns the volume's collision solver.
func (v *VerticesVolume) GetSolver() *collisions.Solver {
	return v.solver
}

// QueryOverlaps queries face overlaps in local space against object.
func (v *VerticesVolume) QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool) {
	v.volume.QueryOverlaps(object, callback)
}

// GetUIFaces returns the transformed face buffer for rendering in model-3D space.
func (v *VerticesVolume) GetUIFaces() (*Faces, *Faces, float64, float64) {
	return v.facesOut, v.facesOut, 0.0, RenderModeModel3D
}

// SetAction is a no-op for rigid single-volume geometry.
func (v *VerticesVolume) SetAction(idx int) {}

// ApplyImpact wakes the rigid volume on localized hits (impulse is applied by the caller via Entity.AddForceAtPoint).
func (v *VerticesVolume) ApplyImpact(fx, fy, fz, px, py, pz float64) {
	v.ent.WakeUp()
}

// SetRagdollMode is a no-op for single rigid volumes.
func (v *VerticesVolume) SetRagdollMode() {}
