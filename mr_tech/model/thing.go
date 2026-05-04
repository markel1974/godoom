package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// IThing represents a general-purpose interface for entities with configuration, physics, and rendering properties.
type IThing interface {
	config.IThingConfig

	SetIdentifier(identifier int)

	GetIdentifier() int

	GetBase() *ThingBase

	GetAABB() *physics.AABB

	GetVertices() ([]*Face, int, []*Face, int, float64, float64)

	GetLocation() *Volume

	GetVolume() *Volume

	GetDisplacement() (float64, float64, float64)

	GetCage() *CollisionCage

	GetEntity() *physics.Entity

	IsActive() bool

	SetActive(active bool)

	StageThinking(playerX float64, playerY float64, playerZ float64)

	StageCompute()

	StageResolve(solverJitter float64)

	StageApply()

	StartLoop()

	PostMessage(ec *ThingEvent)
}
