package main

import (
	"github.com/markel1974/godoom/pixels"
)

type RenderSoftware struct {
	textures           *Textures
	sectorsMaxHeight   float64
	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	dp                 *DrawPolygon
}

func NewSoftwareRender(screenWidth int, screenHeight int, textures *Textures, sectorsMaxHeight float64) *RenderSoftware {
	return &RenderSoftware{
		textures:           textures,
		sectorsMaxHeight:   sectorsMaxHeight,
		targetIdx:          0,
		targetSectors:      map[int]bool{0: true},
		targetLastCompiled: 0,
		targetEnabled:      false,
		dp:                 NewDrawPolygon(screenWidth, screenHeight),
	}
}

func (r *RenderSoftware) DebugMoveSectorToggle() {
	r.targetEnabled = !r.targetEnabled
}

func (r *RenderSoftware) DebugMoveSector(forward bool) {
	if forward {
		if r.targetIdx < r.targetLastCompiled {
			r.targetIdx++
		}
	} else {
		if r.targetIdx > 0 {
			r.targetIdx--
		}
	}
	for k := 0; k < r.targetLastCompiled; k++ {
		r.targetSectors[k] = k == r.targetIdx
	}
}

func (r *RenderSoftware) Render(surface *pixels.PictureRGBA, vi *viewItem, css []*CompiledSector, compiled int) {
	r.targetLastCompiled = compiled
	if compiled > 0 {
		//wg := &sync.WaitGroup{}
		//wg.Add(compiled)

		for idx := compiled - 1; idx >= 0; idx-- {
			mode := r.textures.viewMode
			if r.targetEnabled {
				if f, _ := r.targetSectors[idx]; !f {
					mode = 2
				}
			}

			polygons := css[idx].Get()
			//go func(polygons []*CompiledPolygon) {
			//dp := NewDrawPolygon(r.screenWidth, r.screenHeight)
			//for k := 0; k < len(polygons); k++ {
			for k := len(polygons) - 1; k >= 0; k-- {
				cp := polygons[k]
				//dp.Setup(surface, p.points, p.pLen, p.kind, p.light1, p.light2)
				//r.renderPolygon(w, p, dp, mode)
				r.dp.Setup(surface, cp.points, cp.pLen, cp.kind, cp.light1, cp.light2)
				r.renderPolygon(vi, cp, r.dp, mode, r.sectorsMaxHeight)
			}
			//	wg.Done()
			//}(polygons)
		}
		//wg.Wait()
	}
}

func (r *RenderSoftware) renderPolygon(vi *viewItem, cp *CompiledPolygon, dr *DrawPolygon, mode int, sectorsMaxHeight float64) {
	switch mode {
	case 0:
		dr.DrawWireFrame(false)
		return
	case 1:
		dr.DrawWireFrame(true)
		return
	case 2:
		dr.DrawRectangle()
		return
	case 3:
		dr.DrawPoints(5)
		return
	case 4:
		dr.DrawWireFrame(false)
		dr.DrawPoints(10)
		return
	case 5:
		dr.DrawWireFrame(true)
		dr.DrawPoints(10)
		return
	case 6:
		dr.DrawRectangle()
		dr.DrawPoints(10)
		return
	case 7:
		dr.DrawWireFrame(true)
		dr.DrawRectangle()
		return
	}

	switch cp.kind {
	case IdWall:
		target := cp.Sector.Ceil - cp.Sector.Floor
		yRef := sectorsMaxHeight
		if target > 1 {
			yRef = sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.WallTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdUpper:
		target := cp.Sector.Ceil - cp.Neighbor.Ceil
		yRef := sectorsMaxHeight
		if target > 1 {
			yRef = sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.UpperTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdLower:
		target := cp.Sector.Floor - cp.Neighbor.Floor
		yRef := sectorsMaxHeight
		if target > 1 {
			yRef = sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.LowerTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdCeil:
		dr.DrawTexturePlayer(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(cp.Sector.CeilTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdFloor:
		dr.DrawTexturePlayer(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdFloorTest:
		dr.DrawTexturePlayer(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdCeilTest:
		dr.DrawTexturePlayer(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(p.Sector.CeilTexture, p.x1, p.x2, p.tz1, p.tz2, p.u0, p.u1, 1.0)
	default:
		dr.DrawWireFrame(true)
	}
}
