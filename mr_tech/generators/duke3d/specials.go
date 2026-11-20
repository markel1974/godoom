package duke3d

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// findLowestAdjacentHeights returns the lowest ceiling and floor heights among sectors adjacent to sec.
func (w *World) findLowestAdjacentHeights(sec *lumps.Sector) (lowCeil float64, lowFloor float64, hasAdj bool) {
	lowCeil = math.MaxFloat64
	lowFloor = math.MaxFloat64
	start := int(sec.WallPtr)
	count := int(sec.WallNum)
	for i := 0; i < count; i++ {
		wall := w.level.GetWall(start + i)
		if wall == nil || wall.NextSector < 0 {
			continue
		}
		adj := w.level.GetSector(int(wall.NextSector))
		if adj == nil {
			continue
		}
		adjCeil := w.computeHeight(adj.CeilingZ)
		adjFloor := w.computeHeight(adj.FloorZ)
		if adjCeil < lowCeil {
			lowCeil = adjCeil
		}
		if adjFloor < lowFloor {
			lowFloor = adjFloor
		}
		hasAdj = true
	}
	return lowCeil, lowFloor, hasAdj
}

// buildDukeDoorThing constructs a dynamic ThingDoorDef box spanning a Build sector's XY bounds from minZ to maxZ.
func (w *World) buildDukeDoorThing(id string, sec *lumps.Sector, minZ, maxZ float64, sidePicNum, capPicNum int16, spec *config.DoorSpec) *config.Thing {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	start := int(sec.WallPtr)
	count := int(sec.WallNum)
	for i := 0; i < count; i++ {
		wall := w.level.GetWall(start + i)
		if wall == nil {
			continue
		}
		wx := float64(wall.X) * w.scaleXY
		wy := float64(wall.Y) * w.scaleXY
		if w.flipY {
			wy = -wy
		}
		if wx < minX {
			minX = wx
		}
		if wy < minY {
			minY = wy
		}
		if wx > maxX {
			maxX = wx
		}
		if wy > maxY {
			maxY = wy
		}
	}
	width := maxX - minX
	depth := maxY - minY
	height := maxZ - minZ
	if width <= 0 || depth <= 0 || height <= 0 {
		return nil
	}

	cx := (minX + maxX) * 0.5
	cy := (minY + maxY) * 0.5
	cz := (minZ + maxZ) * 0.5
	hw, hd, hh := width*0.5, depth*0.5, height*0.5

	sideFrames := w.textures.GetTileFrames(int(sidePicNum))
	capFrames := w.textures.GetTileFrames(int(capPicNum))
	sideMat := config.NewConfigMaterial("duke.door.side", sideFrames, config.MaterialKindSolid)
	capMat := config.NewConfigMaterial("duke.door.cap", capFrames, config.MaterialKindSolid)

	tris := buildDukeBoxTriangles(hw, hd, hh, sideMat, capMat)
	model3d := config.NewModel3DEntry(1, []string{"default"}, []string{"death", "die"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(tris)

	const heavyMass = 1000.0
	radius := math.Max(hw, hd)
	cfgThing := config.NewConfigThing(id, geometry.XYZ{X: cx, Y: cy, Z: cz}, 0, config.ThingDoorDef, heavyMass, radius, height, 0)
	cfgThing.GForce = 0
	cfgThing.Element3D = model3d
	cfgThing.Door = spec
	_ = common.NewItemFromConfig(cfgThing)
	return cfgThing
}

func buildDukeBoxTriangles(hw, hd, hh float64, sideMat, capMat *config.Material) []config.Model3DEntryTriangle {
	p := [8]geometry.XYZ{
		{X: -hw, Y: -hd, Z: -hh},
		{X: hw, Y: -hd, Z: -hh},
		{X: hw, Y: hd, Z: -hh},
		{X: -hw, Y: hd, Z: -hh},
		{X: -hw, Y: -hd, Z: hh},
		{X: hw, Y: -hd, Z: hh},
		{X: hw, Y: hd, Z: hh},
		{X: -hw, Y: hd, Z: hh},
	}
	quad := func(a, b, c, d int, mat *config.Material) [2]config.Model3DEntryTriangle {
		t1 := config.NewModel3DEntryTriangle(mat)
		t1.Vertices[0] = config.Model3DEntryVertex{Pos: p[a], U: 0, V: 0}
		t1.Vertices[1] = config.Model3DEntryVertex{Pos: p[b], U: 1, V: 0}
		t1.Vertices[2] = config.Model3DEntryVertex{Pos: p[c], U: 1, V: 1}
		t2 := config.NewModel3DEntryTriangle(mat)
		t2.Vertices[0] = config.Model3DEntryVertex{Pos: p[a], U: 0, V: 0}
		t2.Vertices[1] = config.Model3DEntryVertex{Pos: p[c], U: 1, V: 1}
		t2.Vertices[2] = config.Model3DEntryVertex{Pos: p[d], U: 0, V: 1}
		return [2]config.Model3DEntryTriangle{t1, t2}
	}
	var tris []config.Model3DEntryTriangle
	for _, q := range [][5]int{
		{0, 1, 5, 4, 0},
		{1, 2, 6, 5, 0},
		{2, 3, 7, 6, 0},
		{3, 0, 4, 7, 0},
		{4, 5, 6, 7, 1},
		{3, 2, 1, 0, 1},
	} {
		mat := sideMat
		if q[4] == 1 {
			mat = capMat
		}
		pair := quad(q[0], q[1], q[2], q[3], mat)
		tris = append(tris, pair[0], pair[1])
	}
	return tris
}

// buildDukeTeleports converts paired Build Engine SE 7 (PicNum == 1, LoTag == 7) sprites
// sharing the same HiTag into bidirectional ThingTeleportDef triggers.
func buildDukeTeleports(level *lumps.Level, w *World) []*config.Thing {
	type seNode struct {
		pos   geometry.XYZ
		angle float64
	}
	groups := make(map[int16][]seNode)
	for _, sp := range level.Sprites {
		if sp.PicNum != 1 || sp.LoTag != 7 {
			continue
		}
		sx := float64(sp.X) * w.scaleXY
		sy := float64(sp.Y) * w.scaleXY
		if w.flipY {
			sy = -sy
		}
		sz := w.computeHeight(sp.Z)
		if sec := level.GetSector(int(sp.SectNum)); sec != nil {
			sz = w.computeHeight(sec.FloorZ)
		}
		groups[sp.HiTag] = append(groups[sp.HiTag], seNode{
			pos:   geometry.XYZ{X: sx, Y: sy, Z: sz},
			angle: ToDegrees(sp.Ang) * math.Pi / 180.0,
		})
	}

	var out []*config.Thing
	for hiTag, nodes := range groups {
		if len(nodes) < 2 {
			continue
		}
		for i := 0; i < 2; i++ {
			src := nodes[i]
			dst := nodes[1-i]
			tpCfg := config.NewConfigThing(
				fmt.Sprintf("duke_se7_%d_%d", hiTag, i),
				src.pos,
				0,
				config.ThingTeleportDef,
				1000.0,
				20.0,
				56.0,
				0,
			)
			tpCfg.GForce = 0
			_ = common.NewItemFromConfig(tpCfg)
			tpCfg.Teleport = &config.TeleportSpec{
				Destination:   dst.pos,
				TargetAngle:   dst.angle,
				ResetVelocity: true,
				Cooldown:      0.5,
			}
			out = append(out, tpCfg)
		}
	}
	return out
}
