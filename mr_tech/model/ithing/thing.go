package ithing

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// IThing represents a general-purpose interface for entities with configuration, physics, and rendering properties.
type IThing interface {
	config.IThingConfig

	IVertices

	IsDeformable() bool

	GetSolver() *collisions.Solver

	IsEqual(other IThing) bool

	GetDistSq(pX, pY, pZ float64) float64

	IsActive() bool

	SetActive(active bool)

	IsSleeping() bool

	WakeUp()

	EnterSleep()

	CanSleep() bool

	UpdateSleepState() bool

	SetSolverIndex(idx int)

	GetSolverIndex() int

	HasThinking() bool

	StageThinking(playerX float64, playerY float64, playerZ float64)

	StagePrepare(session uint64)

	StageWarmStart()

	StageResolve()

	StageApply()

	StageResolvePosition()

	StartLoop()

	PostMessage(ec *ThingEvent)
}
