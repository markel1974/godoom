package main

import (
	"fmt"
	textures2 "github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
	"strings"
	"sync"
)

type RenderSoftware struct {
	screenWidth        int
	screenHeight       int
	textures           *textures2.Textures
	sectorsMaxHeight   float64
	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string
	dp                 *DrawPolygon
}

func NewSoftwareRender(screenWidth int, screenHeight int, textures *textures2.Textures, sectorsMaxHeight float64) *RenderSoftware {
	return &RenderSoftware{
		screenWidth:        screenWidth,
		screenHeight:       screenHeight,
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
	//r.stub(surface, r.dp)
	//return
	r.targetLastCompiled = compiled
	if compiled < 1 {
		return
	}
	r.serialRender(surface, vi, css, compiled)
	//r.parallelRender(surface, vi, css, compiled)
}

func (r *RenderSoftware) serialRender(surface *pixels.PictureRGBA, vi *viewItem, css []*CompiledSector, compiled int) {
	//test := make(map[string]bool)
	for idx := compiled - 1; idx >= 0; idx-- {
		//if _, ok := test[css[idx].sector.Id]; ok {
		//	fmt.Println("Already rendered")
		//	continue
		//}
		//test[css[idx].sector.Id] = true
		mode := r.textures.GetViewMode()
		if r.targetEnabled {
			if f, _ := r.targetSectors[idx]; !f {
				mode = 2
			} else {
				if r.targetId != css[idx].sector.Id {
					r.targetId = css[idx].sector.Id
					var neighbors []string
					for _, z := range css[idx].sector.Segments {
						id := ""; if z != nil { id = z.Ref }
						neighbors = append(neighbors, id)
					}
					fmt.Println("Current target Sector:", r.targetId, strings.Join(neighbors, ","), css[idx].sector.Tag)

				}
			}
		}
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			r.dp.Setup(surface, cp.points, cp.pLen, cp.kind, cp.light1, cp.light2)
			r.renderPolygon(vi, cp, r.dp, mode)
		}
	}
}

func (r *RenderSoftware) parallelRender(surface *pixels.PictureRGBA, vi *viewItem, css []*CompiledSector, compiled int) {
	//Experimental Render
	wg := &sync.WaitGroup{}
	wg.Add(compiled)

	for idx := compiled - 1; idx >= 0; idx-- {
		mode := r.textures.GetViewMode()
		if r.targetEnabled {
			if f, _ := r.targetSectors[idx]; !f {
				mode = 2
			}
		}
		//TODO queue
		go func(polygons []*CompiledPolygon) {
			//TODO each renderer must have a separate DrawPolygon
			dp := NewDrawPolygon(r.screenWidth, r.screenHeight)
			for k := len(polygons) - 1; k >= 0; k-- {
				cp := polygons[k]
				dp.Setup(surface, cp.points, cp.pLen, cp.kind, cp.light1, cp.light2)
				r.renderPolygon(vi, cp, dp, mode)
			}
			wg.Done()
		}(css[idx].Get())
	}
	wg.Wait()
}

func (r *RenderSoftware) renderPolygon(vi *viewItem, cp *CompiledPolygon, dr *DrawPolygon, mode int) {
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
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.WallTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdUpper:
		target := cp.Sector.Ceil - cp.Neighbor.Ceil
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.UpperTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdLower:
		target := cp.Sector.Floor - cp.Neighbor.Floor
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.LowerTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, yRef)
	case IdCeil:
		dr.DrawPerspectiveTexture(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(cp.Sector.CeilTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdFloor:
		dr.DrawPerspectiveTexture(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdFloorTest:
		dr.DrawPerspectiveTexture(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case IdCeilTest:
		dr.DrawPerspectiveTexture(vi.where.X, vi.where.Y, vi.where.Z, vi.yaw, vi.angleSin, vi.angleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(p.Sector.CeilTexture, p.x1, p.x2, p.tz1, p.tz2, p.u0, p.u1, 1.0)
	default:
		dr.DrawWireFrame(true)
	}
}
