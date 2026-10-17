package config

import (
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics"
)

type ThinkingFunc func(self IThingConfig, playerX, playerY, playerZ float64)

type CollisionFunc func(self IThingConfig, other IThingConfig)

type ImpactFunc func(self IThingConfig, other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)

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

	MoveTowards(dirX, dirY, targetSpeed, accelForce float64)

	Jump(leapX, leapY, zFactor float64) bool

	LaunchObject(throwableIndex int, cf CollisionFunc, mf ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64)

	FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64)

	Impact(other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)
}
