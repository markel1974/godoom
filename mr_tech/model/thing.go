package model

import (
	"github.com/markel1974/godoom/mr_tech/config"
)

type ICollisionCage any

// IThing represents a general-purpose interface for entities with configuration, physics, and rendering properties.
type IThing interface {
	config.IThingConfig

	IVertices

	IsEqual(other IThing) bool

	GetCage() ICollisionCage

	GetDistSq(pX, pY, pZ float64) float64

	IsActive() bool

	SetActive(active bool)

	HasThinking() bool

	StageThinking(playerX float64, playerY float64, playerZ float64)

	StagePrepare()

	StageResolve()

	StageApply()

	StartLoop()

	PostMessage(ec *ThingEvent)
}
