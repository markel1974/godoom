package main

import (
	"math"
)

const (
	EyeHeight  = 6
	DuckHeight = 2.5
	HeadMargin = 2
	KneeHeight = 2
)

type Player struct {
	where    XYZ
	velocity XYZ
	angle    float64
	angleSin float64
	angleCos float64
	yaw      float64
	yawState float64
	sector   *Sector
	ducking  bool
	falling  bool
}

func NewPlayer(x float64, y float64, z float64, angle float64, sector *Sector) *Player {
	p := &Player{
		where:    XYZ{X: x, Y: y, Z: z + EyeHeight},
		velocity: XYZ{},
		yaw:      0,
		yawState: 0,
		sector:   sector,
	}
	p.SetAngle(angle)
	return p
}

func (p *Player) AddAngle(angle float64) {
	p.SetAngle(p.angle + angle)
}

func (p *Player) SetAngle(angle float64) {
	p.angle = angle
	p.angleSin = math.Sin(p.angle)
	p.angleCos = math.Cos(p.angle)
}

func (p *Player) GetAngle() (float64, float64, float64) {
	return p.angle, p.angleSin, p.angleCos
}

func (p *Player) SetYaw(y float64) {
	p.yawState = clampF(p.yawState-(y*0.05), -5, 5)
	p.yaw = p.yawState - (p.velocity.Z * 0.5)
}

func (p *Player) Move(up bool, down bool, left bool, right bool, slow bool) {
	var moveX float64
	var moveY float64
	var acceleration float64
	impulse := 0.2
	if slow {
		impulse = 0.01
	}
	if up || down || left || right {
		if up {
			moveX += p.angleCos * impulse
			moveY += p.angleSin * impulse
		}
		if down {
			moveX -= p.angleCos * impulse
			moveY -= p.angleSin * impulse
		}
		if left {
			moveX += p.angleSin * impulse
			moveY -= p.angleCos * impulse
		}
		if right {
			moveX -= p.angleSin * impulse
			moveY += p.angleCos * impulse
		}
		acceleration = 0.4
	} else {
		acceleration = 0.2
	}
	p.velocity.X = p.velocity.X*(1-acceleration) + (moveX * acceleration)
	p.velocity.Y = p.velocity.Y*(1-acceleration) + (moveY * acceleration)
}

func (p *Player) SetJump() {
	p.velocity.Z += 0.5
	p.falling = true
}

func (p *Player) SetDucking() {
	p.ducking = !p.ducking
	if p.ducking {
		p.falling = true
	}
}

func (p *Player) GetCoords() (float64, float64) {
	return p.where.X, p.where.Y
}

func (p *Player) AddCoords(x float64, y float64) {
	p.where.X += x
	p.where.Y += y
	p.falling = true
}

func (p *Player) GetZ() float64 {
	return p.where.Z
}

func (p *Player) GetVelocity() (float64, float64) {
	return p.velocity.X, p.velocity.Y
}

func (p *Player) GetSector() *Sector {
	return p.sector
}

func (p *Player) SetSector(sector *Sector) {
	p.sector = sector
}

func (p *Player) GetYaw() float64 {
	return p.yaw
}

func (p *Player) EyeHeight() float64 {
	if p.ducking {
		return DuckHeight
	} else {
		return EyeHeight
	}
}

func (p *Player) VerticalCollision() {
	if p.falling {
		eyeHeight := p.EyeHeight()
		p.velocity.Z -= 0.05
		nextZ := p.where.Z + p.velocity.Z
		if p.velocity.Z < 0 && nextZ < p.sector.Floor+eyeHeight {
			// down
			p.where.Z = p.sector.Floor + eyeHeight
			p.velocity.Z = 0
			p.falling = false
		} else if p.velocity.Z > 0 && nextZ > p.sector.Ceil {
			// up
			p.velocity.Z = 0
			p.falling = true
		}
		if p.falling {
			p.where.Z += p.velocity.Z
		}
	}
}

func (p *Player) IsMoving() bool {
	return !(p.velocity.X == 0 && p.velocity.Y == 0)
}

func (p *Player) Head() float64 {
	return p.where.Z + HeadMargin
}

func (p *Player) Knee() float64 {
	return p.where.Z - p.EyeHeight() + KneeHeight
}

func (p *Player) Update() {
	if math.Abs(p.velocity.X) < 0.001 {
		p.velocity.X = 0
	}
	if math.Abs(p.velocity.Y) < 0.001 {
		p.velocity.Y = 0
	}
}
