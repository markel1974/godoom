package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Vertex represents a point in 3D space with additional 2D texture coordinates U and V.
type Vertex struct {
	Material *textures.Texture
	geometry.XYZ
	U, V        float64
	Origin      geometry.XYZ
	IsBillboard float64
}

// IThing defines an interface for a game entity with methods for retrieving identifiers, position, and physics properties.
type IThing interface {
	GetId() string

	SetIdentifier(identifier int)

	GetIdentifier() int

	GetKind() config.ThingType

	GetAABB() *physics.AABB

	GetAnimation() *textures.Animation

	GetPosition() (float64, float64, float64)

	GetVertices() [][3]Vertex

	GetLight() *Light

	GetMinZ() float64

	GetMaxZ() float64

	GetEntity() *physics.Entity

	Compute(playerX float64, playerY float64, playerZ float64)

	GetLocation() *Volume

	PhysicsApply()

	IsActive() bool

	SetActive(active bool)

	OnCollide(other IThing)
}
