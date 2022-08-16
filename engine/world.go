package main

import (
	"fmt"
	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
	"math"
	"strconv"
)



func pointInPolygonF(px float64, py float64, points []*model.XYKind2) bool {
	nVert := len(points)
	j := nVert - 1
	c := false
	for i := 0; i < nVert; i++ {
		if ((points[i].Y >= py) != (points[j].Y >= py)) && (px <= (points[j].X-points[i].X)*(py-points[i].Y)/(points[j].Y-points[i].Y)+points[i].X) {
			c = !c
		}
		j = i
	}
	return c
}


type World struct {
	screenWidth  int
	screenHeight int
	textures     *textures.Textures
	tree         *BSPTree
	render       IRender
	player       *Player
	vi           *viewItem
	debug        bool
	debugIdx     int
}

func NewWorld(screenWidth int, screenHeight int, maxQueue int, viewMode int) *World {
	t, _ := textures.NewTextures(viewMode)
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

func (w *World) Setup(cfg *model.Input) error {
	compiler := model.NewCompiler()
	err := compiler.Setup(cfg, w.textures)
	if err != nil { return err }
	playerSector, err := compiler.Get(cfg.Player.Sector)
	if err != nil { return err }
	if err := w.tree.Setup(compiler.GetSectors(), compiler.GetMaxHeight()); err != nil { return err }
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
		if neighbor := sector.Vertices[s]; neighbor != nil {
			curr := vert[s]
			next := vert[s+1]
			if mathematic.IntersectBoxF(px0, py0, px1, py1, curr.X, curr.Y, next.X, next.Y) {
				ps := mathematic.PointSideF(px1, py1, curr.X, curr.Y, next.X, next.Y)
				if ps < 0 {
					if neighbor.Sector != nil {
						w.player.SetSector(neighbor.Sector)
						if w.debug {
							fmt.Println("New Sector", neighbor.Ref)
							if i, err := strconv.Atoi(neighbor.Ref); err == nil {
								w.debugIdx = i
							}
						}
						found = true
					}
					break
				}
			}
		}
	}

	if !w.debug {
		if !found {
			if !pointInPolygonF(px1, py1, sector.Vertices[:sector.NPoints]) {
				return
			}
		}
	}
	w.player.AddCoords(dx, dy)
}

func (w *World) Update(surface *pixels.PictureRGBA) {
	_, sin, cos := w.player.GetAngle()
	px, py := w.player.GetCoords()
	pz := w.player.GetZ()
	w.vi.sector = w.player.GetSector()
	w.vi.where = model.XYZ{X: px, Y: py, Z: pz}
	w.vi.angleCos = cos
	w.vi.angleSin = sin
	w.vi.yaw = w.player.GetYaw()

	cs, count := w.tree.Compile(w.vi)
	w.render.Render(surface, w.vi, cs, count)
	if w.debug {
		w.drawStub(surface)
	}

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

		if mathematic.IntersectBoxF(px, py, p1, p2, curr.X, curr.Y, next.X, next.Y) &&
			mathematic.PointSideF(p1, p2, curr.X, curr.Y, next.X, next.Y) < 0 {

			neighbor := sect.Vertices[s].Sector

			// Check where the hole is.
			holeLow := 9e9
			holeHigh := -9e9
			if neighbor != nil {
				holeLow = mathematic.MaxF(sect.Floor, neighbor.Floor)
				holeHigh = mathematic.MinF(sect.Ceil, neighbor.Ceil)
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

func (w *World) DoDebug(next int) {
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= len(w.tree.sectors) { return }
	w.debugIdx = idx
	sector := w.tree.sectors[idx]
	x := sector.Vertices[0].X
	y := sector.Vertices[0].Y
	fmt.Println("CURRENT DEBUG IDX:", w.debugIdx, "total points:", sector.NPoints, "vertices:", len(sector.Vertices))
	w.player.SetSector(sector)
	w.player.SetCoords(x + 5, y + 5)
}

func  (w * World) drawStub(surface *pixels.PictureRGBA) {
	sector := w.tree.sectors[w.debugIdx]
	w.drawSingleStub(surface, sector)
	/*
	x := 640.0 / 300.0
	y := 640.0 / 300.0
	for idx, s := range w.tree.sectors {
		selected := false; if idx == w.debugIdx { selected = true }
		w.drawSingleStubScale(surface, s, x, y, selected)
	}
	*/
}


func  (w * World) drawSingleStubScale(surface *pixels.PictureRGBA, sector * model.Sector, xFactor float64, yFactor float64, selected bool) {
	t  := make([]model.XYZ, len(sector.Vertices))

	for idx := uint64(0); idx < sector.NPoints; idx++ {
		v := sector.Vertices[idx]
		x := v.X * xFactor
		y := v.Y * yFactor - 300
		t[idx].X = x
		t[idx].Y = y
	}
	colorLine := 0x00ff00
	colorPoint := 0xff0000
	if !selected {
		colorLine = 0xB0E0E6
		colorPoint = 0x00BFFF
	}
	dp := NewDrawPolygon(640, 480)
	dp.Setup(surface, t, len(t), colorLine, 1.0, 1.0)
	dp.DrawPoints(10)
	dp.color = colorPoint
	dp.DrawLines()
}

func  (w * World) drawSingleStub(surface *pixels.PictureRGBA, sector * model.Sector) {
	t  := make([]model.XYZ, len(sector.Vertices))
	maxX := 0.0
	maxY := 0.0
	for idx := uint64(0); idx < sector.NPoints; idx++ {
		v := sector.Vertices[idx]
		x := math.Abs(v.X)
		y := math.Abs(v.Y)
		if x > maxX { maxX = x }
		if y > maxY { maxY = y }
	}

	xFactor := float64(w.screenWidth) / maxX
	yFactor := float64(w.screenHeight) / maxY

	for idx := uint64(0); idx < sector.NPoints; idx++ {
		v := sector.Vertices[idx]
		x := (v.X * xFactor) - (float64(w.screenWidth) / 2)
 		y := (v.Y * yFactor) - (float64(w.screenHeight) / 2)
		t[idx].X = x
		t[idx].Y = y
	}
	dp := NewDrawPolygon(640, 480)
	dp.Setup(surface, t, len(t), 0x00ff00, 1.0, 1.0)
	dp.DrawPoints(10)
	dp.color = 0xff0000
	dp.DrawLines()
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
