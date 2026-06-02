package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

// IThing represents a general-purpose interface for entities with configuration, physics, and rendering properties.
type IThing interface {
	config.IThingConfig

	IVertices

	GetBase() *ThingBase

	GetCage() *CollisionCage

	IsActive() bool

	SetActive(active bool)

	StageThinking(playerX float64, playerY float64, playerZ float64)

	StagePrepare() bool

	//StageCompute()

	StageResolve(solverIndex int, solverJitter float64)

	StageApply(solverJitter float64)

	StartLoop()

	PostMessage(ec *ThingEvent)
}
