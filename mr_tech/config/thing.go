package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

// BindFunc is a function type used to bind and configure an IThingConfig instance for initialization or setup.
type BindFunc func(self IThingConfig)

// ThinkingFunc represents a function type that processes positional data of a player in 3D space.
type ThinkingFunc func(playerX, playerY, playerZ float64)

// CollisionFunc defines a function type for handling collisions with another IThingConfig entity.
type CollisionFunc func(other IThingConfig)

// ImpactFunc defines a function type for handling the impact between two configurable entities with force and direction inputs.
type ImpactFunc func(other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)

// IThingConfig defines the configuration and behavior contract for entities within the system.
type IThingConfig interface {
	GetId() string

	GetKind() ThingType

	GetEntity() *physics.Entity

	SetAction(idx int)

	GetAngle() float64

	SetAngle(angle float64)

	GetAcceleration() float64

	GetMaxStep() float64

	GetSpeed() float64

	GetImpulse() float64

	GetWakeUpDistance() float64

	IsVisible() bool

	MoveTowards(dirX, dirY, targetSpeed, accelForce float64)

	Jump(leapX, leapY, zFactor float64) bool

	LaunchObject(throwableIndex int, cf CollisionFunc, mf ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64)

	FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64)

	Impact(other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)
}
