package config

import (
	"github.com/markel1974/godoom/mr_tech/model/geometry"
)

type ThinkingFunc func(self IThingConfig, playerX, playerY, playerZ float64)

type CollisionFunc func(self IThingConfig, other IThingConfig)

type ImpactFunc func(self IThingConfig, other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)

type IThingConfig interface {
	GetId() string

	GetKind() ThingType

	SetAction(idx int)

	IsOnGround() bool

	SetOnGround(g bool)

	GetBottomLeft() (float64, float64, float64)

	GetBottomCenter() (float64, float64, float64)

	GetCenter() (float64, float64, float64)

	GetAngle() float64

	SetAngle(angle float64)

	GetRadius() float64

	GetDepth() float64

	GetAcceleration() float64

	GetMaxStep() float64

	GetSpeed() float64

	GetWidth() float64

	GetMass() float64

	GetVelocity() (float64, float64, float64)

	AddForce(fx, fy, fz float64)

	MoveTowards(dirX, dirY, targetSpeed, accelForce float64)

	LaunchObject(throwableIndex int, cf CollisionFunc, mf ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64)

	Impact(other IThingConfig, id string, force, closestDist, dirX, dirY, dirZ float64)
}
