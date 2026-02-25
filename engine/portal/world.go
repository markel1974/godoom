package portal

import (
	"fmt"
	"math"
	"strconv"

	"github.com/markel1974/godoom/engine/mathematic"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/polygons"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
)

/*
func pointInPolygonF(px float64, py float64, points []*model.Segment) bool {
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

*/

type World struct {
	screenWidth  int
	screenHeight int
	textures     *textures.Textures
	tree         *PortalRender
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
		tree:         NewPortalRender(screenWidth, screenHeight, maxQueue, t),
		render:       nil,
		vi:           &viewItem{},
	}
	return w
}

func (w *World) Setup(cfg *model.InputConfig) error {
	compiler := model.NewCompiler()
	err := compiler.Setup(cfg, w.textures)
	if err != nil {
		return err
	}
	playerSector, err := compiler.Get(cfg.Player.Sector)
	if err != nil {
		return err
	}
	if err := w.tree.Setup(compiler.GetSectors(), compiler.GetMaxHeight()); err != nil {
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
	for _, segment := range w.player.GetSector().Segments {
		start := segment.Start
		end := segment.End
		if mathematic.IntersectBoxF(px0, py0, px1, py1, start.X, start.Y, end.X, end.Y) {
			ps := mathematic.PointSideF(px1, py1, start.X, start.Y, end.X, end.Y)
			if ps < 0 {
				if segment.Sector != nil {
					w.player.SetSector(segment.Sector)
					if w.debug {
						fmt.Println("New Sector", segment.Ref)
						if i, err := strconv.Atoi(segment.Ref); err == nil {
							w.debugIdx = i
						}
					}
					found = true
				}
				break
			}
		}
	}

	if !w.debug {
		if !found {
			//TODO COMPLETARE
			//if !pointInPolygonF(px1, py1, sector.Segments) {
			//	return
			//}
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

	// Check if the player is about to cross one of the sector's edges
	for _, segment := range w.player.GetSector().Segments {
		start := segment.Start
		end := segment.End

		if mathematic.IntersectBoxF(px, py, p1, p2, start.X, start.Y, end.X, end.Y) &&
			mathematic.PointSideF(p1, p2, start.X, start.Y, end.X, end.Y) < 0 {

			// Check where the hole is.
			holeLow := 9e9
			holeHigh := -9e9
			if segment.Sector != nil {
				holeLow = mathematic.MaxF(sect.Floor, segment.Sector.Floor)
				holeHigh = mathematic.MinF(sect.Ceil, segment.Sector.Ceil)
			}

			// Check whether we're bumping into a wall
			if holeHigh < head || holeLow > knee {
				// Bumps into a wall! Slide along the wall
				// This formula is from Wikipedia article "vector projection"
				xd := end.X - start.X
				yd := end.Y - start.Y
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
	if idx < 0 || idx >= len(w.tree.sectors) {
		return
	}
	w.debugIdx = idx
	sector := w.tree.sectors[idx]
	x := sector.Segments[0].Start.X
	y := sector.Segments[0].Start.Y
	fmt.Println("CURRENT DEBUG IDX:", w.debugIdx, "total segments:", len(sector.Segments))
	w.player.SetSector(sector)
	w.player.SetCoords(x+5, y+5)
}

func (w *World) drawStub(surface *pixels.PictureRGBA) {
	if w.debugIdx >= 0 && w.debugIdx < len(w.tree.sectors) {
		sector := w.tree.sectors[w.debugIdx]
		w.drawSingleStub(surface, sector)
	}

	/*
		x := 320.0 / 300.0
		y := 320.0 / 300.0
		for idx, s := range w.tree.sectors {
			selected := false; if idx == w.debugIdx { selected = true }
			w.drawSingleStubScale(surface, s, x, y, selected)
		}

	*/
}

func (w *World) drawSingleStubScale(surface *pixels.PictureRGBA, sector *model.Sector, xFactor float64, yFactor float64, selected bool) {
	var t []model.XYZ
	for _, v := range sector.Segments {
		if v.Kind == model.DefinitionVoid || v.Kind == model.DefinitionUnknown {
			continue
		}
		x1 := v.Start.X
		if x1 == 0 {
			x1 = 1
		}
		x1 *= xFactor
		y1 := v.Start.Y
		if y1 == 0 {
			y1 = 1
		}
		y1 *= yFactor
		x2 := v.End.X
		if x2 == 0 {
			x2 = 1
		}
		x2 *= xFactor
		y2 := v.End.Y
		if y2 == 0 {
			y2 = 1
		}
		y2 *= yFactor
		t = append(t, model.XYZ{X: x1, Y: y1, Z: 0})
		t = append(t, model.XYZ{X: x2, Y: y2, Z: 0})
	}
	if len(t) == 0 {
		return
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
	dp.DrawLines(false)
}

func (w *World) drawSingleStub(surface *pixels.PictureRGBA, sector *model.Sector) {
	maxX := float64(0)
	maxY := float64(0)

	useConvexHull := false

	var segments []*model.Segment

	if useConvexHull {
		ch := &polygons.ConvexHull{}
		segments = ch.FromSector(sector)
	} else {
		segments = sector.Segments
	}

	for _, v := range segments {
		x1 := math.Abs(v.Start.X)
		y1 := math.Abs(v.Start.Y)
		x2 := math.Abs(v.End.X)
		y2 := math.Abs(v.End.Y)
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
		if x2 > maxX {
			maxX = x2
		}
		if y2 > maxY {
			maxY = y2
		}
	}

	xFactor := (float64(w.screenWidth) / 2) / maxX
	yFactor := (float64(w.screenHeight) / 2) / maxY

	var t []model.XYZ
	for _, v := range segments {
		x1 := v.Start.X
		if x1 == 0 {
			x1 = 1
		}
		x1 *= xFactor
		y1 := v.Start.Y
		if y1 == 0 {
			y1 = 1
		}
		y1 *= yFactor
		x2 := v.End.X
		if x2 == 0 {
			x2 = 1
		}
		x2 *= xFactor
		y2 := v.End.Y
		if y2 == 0 {
			y2 = 1
		}
		y2 *= yFactor
		t = append(t, model.XYZ{X: x1, Y: y1, Z: 0})
		t = append(t, model.XYZ{X: x2, Y: y2, Z: 0})
	}

	if len(t) == 0 {
		return
	}
	dp := NewDrawPolygon(640, 480)
	dp.Setup(surface, t, len(t), 0x00ff00, 1.0, 1.0)
	dp.DrawPoints(10)
	dp.color = 0xff0000
	dp.DrawLines(false)
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
