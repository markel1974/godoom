package model

/*
import (
	"math"

	"github.com/markel1974/godoom/mr_tech/physics"
)

// ThingWall represents a UI component or control typically used to select a value from a range by sliding a handle.
type ThingWall struct {
	wall    *physics.Entity
	volumes *Volumes
}

// NewThingWall initializes and returns a new ThingWall instance, associating it with the provided Volumes object.
func NewThingWall(volumes *Volumes, restitution, friction float64) *ThingWall {
	return &ThingWall{
		volumes: volumes,
		wall:    physics.NewEntity(0, 0, 0, 0, 0, 0, -1, restitution, friction),
	}
}

// GetAABB returns the axis-aligned bounding box (AABB) associated with the ThingWall instance.
func (s *ThingWall) GetAABB() *physics.AABB {
	return s.wall.GetAABB()
}

// GetEntity retrieves the underlying physics.Entity associated with the ThingWall instance.
func (s *ThingWall) GetEntity() *physics.Entity {
	return s.wall
}

// ClosestFace finds the nearest face in a 3D space based on the given positions, velocity, top, bottom, and ellipsoid dimensions (eRadX, eRadY, eRadZ).
// It updates the axis-aligned bounding box (AABB) and queries the volumes for the closest face intersection.
func (s *ThingWall) ClosestFace(viewX, viewY, viewZ, pX, pY, pZ, velX, velY, velZ, top, bottom, eRadX, eRadY, eRadZ float64) (*Face, float64, float64, float64, float64) {
	// L'AABB di ricerca ora viene espanso usando le corrette proporzioni dell'ellissoide (capsula)
	// Note:
	// In e-space, l'origine dello sweep (viewZ, pZ) è il CENTRO della capsula.
	// L'espansione Z verso l'alto e verso il basso usa eRadZ (metà altezza).
	minX := math.Min(viewX, pX) - eRadX
	maxX := math.Max(viewX, pX) + eRadX
	minY := math.Min(viewY, pY) - eRadY
	maxY := math.Max(viewY, pY) + eRadY
	minZ := math.Min(viewZ, pZ) - eRadZ
	maxZ := math.Max(viewZ, pZ) + eRadZ
	s.GetAABB().Rebuild(minX, minY, math.Min(bottom, minZ), maxX, maxY, math.Max(top, maxZ))
	return s.volumes.QueryClosestFace(s, viewX, viewY, viewZ, velX, velY, velZ, eRadX, eRadY, eRadZ)
}


*/
