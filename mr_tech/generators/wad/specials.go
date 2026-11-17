package wad

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/wad/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// Doom linedef specials for teleports and lifts (doors are listed in _doors in sprite_dictionary.go).
var _teleports = map[int16]bool{
	39:  true, // W1 Teleport
	97:  true, // WR Teleport
	125: true, // W1 Teleport (monsters only)
	126: true, // WR Teleport (monsters only)
}

var _lifts = map[int16]float64{
	10:  120.0, // W1 Lift Lower Wait Raise
	21:  120.0, // S1 Lift Lower Wait Raise
	62:  120.0, // SR Lift Lower Wait Raise
	88:  120.0, // WR Lift Lower Wait Raise
	120: 240.0, // WR Lift Lower Wait Raise (fast)
	121: 240.0, // W1 Lift Lower Wait Raise (fast)
	122: 240.0, // S1 Lift Lower Wait Raise (fast)
	123: 240.0, // SR Lift Lower Wait Raise (fast)
}

// sectorSpecialInfo summarizes dynamic door/lift behavior detected for a WAD sector.
type sectorSpecialInfo struct {
	isDoor      bool
	doorSpec    *config.DoorSpec
	doorTex     string
	targetCeil  float64
	isLift      bool
	liftSpeed   float64
	lowestFloor float64
}

// analyzeSectorSpecials scans linedefs and sector edges to identify dynamic doors and lifts.
func (bld *Builder) analyzeSectorSpecials(level *Level, sectorsEdges [][]Edge) map[int]*sectorSpecialInfo {
	// Index linedef specials by sector tag for remote-triggered doors and lifts (Tag > 0).
	tagDoors := make(map[int16]int16)
	tagLifts := make(map[int16]float64)
	for _, ld := range level.LineDefs {
		if ld.Tag <= 0 {
			continue
		}
		if _, ok := _doors[ld.Function]; ok {
			tagDoors[ld.Tag] = ld.Function
		}
		if speed, ok := _lifts[ld.Function]; ok {
			tagLifts[ld.Tag] = speed
		}
	}

	result := make(map[int]*sectorSpecialInfo)
	for secIdx, edges := range sectorsEdges {
		if len(edges) == 0 {
			continue
		}
		sec := level.Sectors[secIdx]

		// Detect door: direct door linedef on sector or remote tag match, with collapsed ceiling.
		var doorSpecial int16
		hasDoorSpecial := false
		doorTex := ""
		lowestAdjCeil := int16(math.MaxInt16)
		lowestAdjFloor := int16(math.MaxInt16)
		hasAdjacent := false

		for _, e := range edges {
			if _, ok := _doors[e.LineDef.Function]; ok {
				doorSpecial = e.LineDef.Function
				hasDoorSpecial = true
			}
			if e.LineDef.SideDefRight != -1 && e.LineDef.SideDefLeft != -1 {
				adjSideIdx := e.LineDef.SideDefRight
				doorSideIdx := e.LineDef.SideDefLeft
				if level.SideDefs[adjSideIdx].SectorRef == uint16(secIdx) {
					adjSideIdx = e.LineDef.SideDefLeft
					doorSideIdx = e.LineDef.SideDefRight
				}
				adjSec := level.Sectors[level.SideDefs[adjSideIdx].SectorRef]
				if adjSec.CeilingHeight < lowestAdjCeil {
					lowestAdjCeil = adjSec.CeilingHeight
				}
				if adjSec.FloorHeight < lowestAdjFloor {
					lowestAdjFloor = adjSec.FloorHeight
				}
				hasAdjacent = true
				if ut := level.SideDefs[doorSideIdx].UpperTexture; ut != "" && ut != "-" {
					doorTex = ut
				}
			}
		}

		if !hasDoorSpecial && sec.Tag > 0 {
			if fn, ok := tagDoors[sec.Tag]; ok {
				doorSpecial = fn
				hasDoorSpecial = true
			}
		}

		if hasDoorSpecial && sec.CeilingHeight <= sec.FloorHeight && hasAdjacent && lowestAdjCeil != math.MaxInt16 {
			targetCeil := lowestAdjCeil - 4
			if targetCeil < sec.FloorHeight {
				targetCeil = lowestAdjCeil
			}
			height := float64(targetCeil - sec.FloorHeight)
			if height > 1.0 {
				result[secIdx] = &sectorSpecialInfo{
					isDoor:     true,
					targetCeil: float64(targetCeil),
					doorTex:    doorTex,
					doorSpec:   buildWadDoorSpec(doorSpecial, sec.Tag, height),
				}
				continue
			}
		}

		// Detect lift: tagged sector with floor higher than its lowest adjacent floor.
		if sec.Tag > 0 {
			if liftSpeed, ok := tagLifts[sec.Tag]; ok && hasAdjacent && lowestAdjFloor != math.MaxInt16 && sec.FloorHeight > lowestAdjFloor {
				result[secIdx] = &sectorSpecialInfo{
					isLift:      true,
					liftSpeed:   liftSpeed,
					lowestFloor: float64(lowestAdjFloor),
				}
			}
		}
	}
	return result
}

// buildWadDoorSpec maps a Doom linedef door special to a DoorSpec.
func buildWadDoorSpec(special int16, tag int16, height float64) *config.DoorSpec {
	spec := &config.DoorSpec{
		OpenOffset:  geometry.XYZ{Z: height},
		Speed:       120.0,
		WaitSeconds: 4.0,
		Activation:  config.DoorActivateTouch,
	}
	switch special {
	case 26, 32:
		spec.RequiredKey = "blue"
	case 27, 34:
		spec.RequiredKey = "yellow"
	case 28, 33:
		spec.RequiredKey = "red"
	}
	switch special {
	case 2, 31, 32, 33, 34:
		spec.WaitSeconds = -1 // Stay open
	case 117, 118, 202, 203, 204, 205, 206, 207:
		spec.Speed = 240.0
	}
	if tag > 0 {
		spec.TargetName = fmt.Sprintf("wad_tag_%d", tag)
	}
	return spec
}

// buildSectorDoorThing creates a dynamic ThingDoorDef box matching a WAD door or lift sector.
func (bld *Builder) buildSectorDoorThing(
	id string,
	edges []Edge,
	minZ, maxZ float64,
	sideTex, capTex string,
	spec *config.DoorSpec,
	texHandler *Textures,
) *config.Thing {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, e := range edges {
		for _, p := range [2]geometry.XY{e.P1, e.P2} {
			if p.X < minX {
				minX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y > maxY {
				maxY = p.Y
			}
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

	var sideFrames []string
	var sideMatName string
	if sideTex != "" && sideTex != "-" {
		sideFrames = texHandler.TextureCreateAnimation(sideTex)
		sideMatName = sideTex
	} else {
		sideFrames = texHandler.FlatCreateAnimation(capTex)
		sideMatName = capTex
	}
	sideMat := config.NewConfigMaterial("door.side", sideFrames, config.MaterialKindSolid)
	bld.configureMaterial(sideMat, sideMatName)
	capMat := config.NewConfigMaterial("door.cap", texHandler.FlatCreateAnimation(capTex), config.MaterialKindSolid)
	bld.configureMaterial(capMat, capTex)

	triangles := buildBoxTriangles(hw, hd, hh, sideMat, capMat)
	model3d := config.NewModel3DEntry(1, []string{"default"}, []string{"death", "die"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(triangles)

	const heavyMass = 1000.0
	radius := math.Max(hw, hd)
	cfgThing := config.NewConfigThing(id, geometry.XYZ{X: cx, Y: cy, Z: cz}, 0, config.ThingDoorDef, heavyMass, radius, height, 0)
	cfgThing.GForce = 0
	cfgThing.Model3DEntry = model3d
	cfgThing.Door = spec
	_ = common.NewItem(cfgThing)
	return cfgThing
}

// buildBoxTriangles creates a 12-triangle axis-aligned box centered at the origin.
func buildBoxTriangles(hw, hd, hh float64, sideMat, capMat *config.Material) []config.Model3DEntryTriangle {
	p := [8]geometry.XYZ{
		{X: -hw, Y: -hd, Z: -hh}, // 0: minX, minY, minZ
		{X: hw, Y: -hd, Z: -hh},  // 1: maxX, minY, minZ
		{X: hw, Y: hd, Z: -hh},   // 2: maxX, maxY, minZ
		{X: -hw, Y: hd, Z: -hh},  // 3: minX, maxY, minZ
		{X: -hw, Y: -hd, Z: hh},  // 4: minX, minY, maxZ
		{X: hw, Y: -hd, Z: hh},   // 5: maxX, minY, maxZ
		{X: hw, Y: hd, Z: hh},    // 6: maxX, maxY, maxZ
		{X: -hw, Y: hd, Z: hh},   // 7: minX, maxY, maxZ
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
		{0, 1, 5, 4, 0}, // South
		{1, 2, 6, 5, 0}, // East
		{2, 3, 7, 6, 0}, // North
		{3, 0, 4, 7, 0}, // West
		{4, 5, 6, 7, 1}, // Top
		{3, 2, 1, 0, 1}, // Bottom
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

// buildWadTeleports converts Doom teleport linedefs (39, 97, 125, 126) and ThingTypeTeleportLanding (14)
// into ThingTeleportDef entities.
func (bld *Builder) buildWadTeleports(level *Level, sectorsEdges [][]Edge, vertexes geometry.Polygon) []*config.Thing {
	type landing struct {
		pos   geometry.XYZ
		angle float64
	}
	landingsByTag := make(map[int16]landing)
	for _, lt := range level.Things {
		if lt.GetType() != lumps.ThingTypeTeleportLanding {
			continue
		}
		pt := geometry.XY{X: float64(lt.X), Y: float64(lt.Y)}
		for secIdx, edges := range sectorsEdges {
			if len(edges) == 0 {
				continue
			}
			sec := level.Sectors[secIdx]
			if sec.Tag <= 0 {
				continue
			}
			if pointInEdges(pt, edges) {
				landingsByTag[sec.Tag] = landing{
					pos:   geometry.XYZ{X: pt.X, Y: pt.Y, Z: float64(sec.FloorHeight)},
					angle: float64(lt.Angle) * math.Pi / 180.0,
				}
				break
			}
		}
	}

	var out []*config.Thing
	for idx, ld := range level.LineDefs {
		if !_teleports[ld.Function] || ld.Tag <= 0 {
			continue
		}
		dest, ok := landingsByTag[ld.Tag]
		if !ok {
			continue
		}
		if int(ld.VertexStart) >= len(vertexes) || int(ld.VertexEnd) >= len(vertexes) {
			continue
		}
		v1 := vertexes[ld.VertexStart]
		v2 := vertexes[ld.VertexEnd]
		midX := (v1.X + v2.X) * 0.5
		midY := (v1.Y + v2.Y) * 0.5

		var floorZ float64
		if ld.SideDefRight >= 0 && int(ld.SideDefRight) < len(level.SideDefs) {
			secIdx := level.SideDefs[ld.SideDefRight].SectorRef
			if int(secIdx) < len(level.Sectors) {
				floorZ = float64(level.Sectors[secIdx].FloorHeight)
			}
		}

		tpCfg := config.NewConfigThing(
			fmt.Sprintf("wad_teleport_%d", idx),
			geometry.XYZ{X: midX, Y: midY, Z: floorZ},
			0,
			config.ThingTeleportDef,
			1000.0,
			16.0,
			56.0,
			0,
		)
		tpCfg.GForce = 0
		_ = common.NewItem(tpCfg)
		tpCfg.Teleport = &config.TeleportSpec{
			Destination:   dest.pos,
			TargetAngle:   dest.angle,
			ResetVelocity: true,
			Cooldown:      0.5,
			PlayerOnly:    ld.Function == 39 || ld.Function == 97,
		}
		out = append(out, tpCfg)
	}
	return out
}

// pointInEdges reports whether p lies inside the 2D region bounded by edges using ray casting.
func pointInEdges(p geometry.XY, edges []Edge) bool {
	inside := false
	for _, e := range edges {
		y1, y2 := e.P1.Y, e.P2.Y
		if (y1 > p.Y) == (y2 > p.Y) {
			continue
		}
		xIntersect := e.P1.X + (p.Y-y1)*(e.P2.X-e.P1.X)/(y2-y1)
		if p.X < xIntersect {
			inside = !inside
		}
	}
	return inside
}
