package ithing

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// IVertices represents the interface for handling vertices, including retrieval, transformations, and related operations.
type IVertices interface {
	Setup(thing IThing) error

	Compile(uint64) bool

	GetAABB() *aabb.AABB

	GetSolver() *collisions.Solver

	QueryOverlaps(object aabb.IObject, callback func(object aabb.IObject) bool)

	GetUIFaces() (*volumes.Faces, *volumes.Faces, float64, float64)

	SetAction(idx int)

	SetRagdollMode()

	Deform(impact geometry.XYZ, impulse geometry.XYZ, radius float64) bool

	DeformDent(impact geometry.XYZ, normal geometry.XYZ, depth, radius float64) bool

	IntersectRay(oX, oY, oZ, dirX, dirY, dirZ, maxDist float64) (float64, geometry.XYZ, bool)
}

// IPartOwner is implemented by articulated owners whose sub-parts live as independent nodes in Things.tree.
type IPartOwner interface {
	GetPartLen() int

	GetPartAt(idx int) IThing

	SyncAttachedParts()
}

// ISubPart is implemented by individual body-part Things owned by a parent IPartOwner until detached/ragdolled.
type ISubPart interface {
	GetOwner() IThing

	IsPhysicsFree() bool
}
