package main

import (
	"github.com/markel1974/godoom/engine/config"
	"github.com/markel1974/godoom/pixels"
)

type XY struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type XYZ struct {
	X float64 `json:"x"`
	Y float64 `json:"x"`
	Z float64 `json:"z"`
}

type World struct {
	screenWidth  int
	screenHeight int
	textures     *Textures
	tree         *BSPTree
	render       IRender
	player       *Player
	vi           *viewItem
}

func NewWorld(screenWidth int, screenHeight int, maxQueue int, viewMode int) *World {
	t, _ := NewTextures(viewMode)
	w := &World{
		screenWidth:  screenWidth,
		screenHeight: screenHeight,
		textures:     t,
		tree:         NewBSPTree(screenWidth, screenHeight, maxQueue, t),
		render:       nil,
		vi:           &viewItem{},
	}
	return w
}

func (w *World) Setup(cfg *config.Config) error {
	playerSector, err := w.tree.Setup(cfg)
	if err != nil {
		return err
	}
	w.player = NewPlayer(cfg.Player.Position.X, cfg.Player.Position.Y, playerSector.Floor, cfg.Player.Angle, playerSector)
	w.render = NewSoftwareRender(w.screenWidth, w.screenHeight, w.textures, w.tree.sectorsMaxHeight)
	return nil
}

func (w *World) movePlayer(dx float64, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	px0, py0 := w.player.GetCoords()
	px1 := px0 + dx
	py1 := py0 + dy
	found := false
	sector := w.player.GetSector()
	vert := sector.Vertices
	for s := uint64(0); s < sector.NPoints; s++ {
		if neighbor := sector.Neighbors[s]; neighbor != nil {
			curr := vert[s]
			next := vert[s+1]
			if intersectBoxF(px0, py0, px1, py1, curr.X, curr.Y, next.X, next.Y) {
				ps := pointSideF(px1, py1, curr.X, curr.Y, next.X, next.Y)
				if ps < 0 {
					w.player.SetSector(neighbor)
					found = true
					break
				}
			}
		}
	}

	if !found {
		if !pointInPolygonF(px1, py1, sector.Vertices[:sector.NPoints]) {
			return
		}
	}
	w.player.AddCoords(dx, dy)
}

func (w *World) Update(surface *pixels.PictureRGBA) {
	_, sin, cos := w.player.GetAngle()
	px, py := w.player.GetCoords()
	pz := w.player.GetZ()
	w.vi.sector = w.player.GetSector()
	w.vi.where = XYZ{X: px, Y: py, Z: pz}
	w.vi.angleCos = cos
	w.vi.angleSin = sin
	w.vi.yaw = w.player.GetYaw()

	cs, count := w.tree.Compile(w.vi)
	w.render.Render(surface, w.vi, cs, count)

	w.player.VerticalCollision()
	if !w.player.IsMoving() {
		return
	}
	head := w.player.Head()
	knee := w.player.Knee()
	//px,	py := w.player.GetCoords()
	dx, dy := w.player.GetVelocity()
	p1 := px + dx
	p2 := py + dy
	sect := w.player.GetSector()
	vert := sect.Vertices

	// Check if the player is about to cross one of the sector's edges
	for s := uint64(0); s < sect.NPoints; s++ {
		curr := vert[s]
		next := vert[s+1]

		if intersectBoxF(px, py, p1, p2, curr.X, curr.Y, next.X, next.Y) &&
			pointSideF(p1, p2, curr.X, curr.Y, next.X, next.Y) < 0 {

			neighbor := sect.Neighbors[s]

			// Check where the hole is.
			holeLow := 9e9
			holeHigh := -9e9
			if neighbor != nil {
				holeLow = maxF(sect.Floor, neighbor.Floor)
				holeHigh = minF(sect.Ceil, neighbor.Ceil)
			}

			// Check whether we're bumping into a wall
			if holeHigh < head || holeLow > knee {
				// Bumps into a wall! Slide along the wall
				// This formula is from Wikipedia article "vector projection"
				xd := next.X - curr.X
				yd := next.Y - curr.Y
				dx = xd * (dx*xd + yd*dy) / (xd*xd + yd*yd)
				dy = yd * (dx*xd + yd*dy) / (xd*xd + yd*yd)
			}
			break
		}
	}

	w.player.Update()

	w.movePlayer(dx, dy)
}

func (w *World) DoPlayerDuckingToggle() {
	w.player.SetDucking()
}

func (w *World) DoPlayerJump() {
	w.player.SetJump()
}

func (w *World) DoPlayerMoves(up bool, down bool, left bool, right bool, slow bool) {
	w.player.Move(up, down, left, right, slow)
}

func (w *World) DoPlayerMouseMove(mouseX float64, mouseY float64) {
	if mouseX > 10 {
		mouseX = 10
	} else if mouseX < -10 {
		mouseX = -10
	}
	if mouseY > 10 {
		mouseY = 10
	} else if mouseY < -10 {
		mouseY = -10
	}

	w.player.AddAngle(mouseX * 0.03)
	w.player.SetYaw(mouseY)

	w.movePlayer(0, 0)
}

func (w *World) DebugMoveSector(forward bool) {
	w.render.DebugMoveSector(forward)
}

func (w *World) DebugMoveSectorToggle() {
	w.render.DebugMoveSectorToggle()
}

func (w *World) DoZoom(zoom float64) {
	w.vi.zoom += zoom
}

/*
func (w * World) ComputePlayer(moves[4]bool, ducking bool, jumping bool,  mouseX float64, mouseY float64) {
	w.player.Angle += mouseX * 0.03
	w.player.YawState = clampF(w.player.YawState - mouseY * 0.05, -5, 5)
	w.player.Yaw = w.player.YawState - w.player.Velocity.Z * 0.5

	w.movePlayer(0,0)

	if ducking {
		w.player.Ducking = true
		w.player.Falling = true
	}

	if jumping {
		w.player.Velocity.Z += 0.5
		w.player.Falling = true
	}
	var moveVec [2]float64

	if moves[0] { moveVec[0] += w.player.AngleCos*0.2; moveVec[1] += w.player.AngleSin*0.2 }
	if moves[1] { moveVec[0] -= w.player.AngleCos*0.2; moveVec[1] -= w.player.AngleSin*0.2 }
	if moves[2] { moveVec[0] += w.player.AngleSin*0.2; moveVec[1] -= w.player.AngleCos*0.2 }
	if moves[3] { moveVec[0] -= w.player.AngleSin*0.2; moveVec[1] += w.player.AngleCos*0.2 }

	var acceleration float64
	if moves[0] || moves[1] || moves[2] || moves[3] {
		acceleration = 0.4
		w.player.Moving = true
	} else {
		acceleration = 0.2
	}

	w.player.Velocity.X = w.player.Velocity.X * (1-acceleration) + moveVec[0] * acceleration
	w.player.Velocity.Y = w.player.Velocity.Y * (1-acceleration) + moveVec[1] * acceleration
}

*/
