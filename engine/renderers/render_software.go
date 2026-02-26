package renderers

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
)

const scaleFactor = 10.0

// RenderSoftware represents a software-based renderer for managing and rendering 2D/3D scenes on a defined screen space.
type RenderSoftware struct {
	screenWidth        int
	screenHeight       int
	textures           *textures.Textures
	sectorsMaxHeight   float64
	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string
	dp                 *DrawPolygon
}

// NewSoftwareRender initializes and returns a new instance of RenderSoftware for software-based rendering.
// It sets the screen dimensions, textures, maximum Sector height, and initializes rendering utilities.
func NewSoftwareRender(screenWidth int, screenHeight int, textures *textures.Textures, sectorsMaxHeight float64) *RenderSoftware {
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

// DebugMoveSectorToggle toggles the Sector targeting mode by enabling or disabling the targetEnabled flag.
func (r *RenderSoftware) DebugMoveSectorToggle() {
	r.targetEnabled = !r.targetEnabled
}

// DebugMoveSector updates the target Sector index based on the direction and adjusts the active state of Sectors accordingly.
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

// Render processes the rendering pipeline for the provided surface, view item, and compiled Sectors.
func (r *RenderSoftware) Render(surface *pixels.PictureRGBA, vi *ViewItem, css []*model.CompiledSector, compiled int) {
	//r.stub(surface, r.dp)
	//return
	r.targetLastCompiled = compiled
	if compiled < 1 {
		return
	}
	r.serialRender(surface, vi, css, compiled)
	//r.parallelRender(surface, vi, css, compiled)
}

// serialRender processes and renders a series of compiled Sectors and their polygons onto the provided surface.
func (r *RenderSoftware) serialRender(surface *pixels.PictureRGBA, vi *ViewItem, css []*model.CompiledSector, compiled int) {
	//test := make(map[string]bool)
	for idx := compiled - 1; idx >= 0; idx-- {
		//if _, ok := test[css[idx].Sector.Id]; ok {
		//	fmt.Println("Already rendered")
		//	continue
		//}
		//test[css[idx].Sector.Id] = true
		mode := r.textures.GetViewMode()
		if r.targetEnabled {
			if f, _ := r.targetSectors[idx]; !f {
				mode = 2
			} else {
				if r.targetId != css[idx].Sector.Id {
					r.targetId = css[idx].Sector.Id
					var neighbors []string
					for _, z := range css[idx].Sector.Segments {
						id := ""
						if z != nil {
							id = z.Ref
						}
						neighbors = append(neighbors, id)
					}
					fmt.Println("Current target Sector:", r.targetId, strings.Join(neighbors, ","), css[idx].Sector.Tag)

				}
			}
		}
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			r.dp.Setup(surface, cp.Points, cp.PLen, cp.Kind, cp.Light1, cp.Light2)
			r.renderPolygon(vi, cp, r.dp, mode)
		}
	}
}

// parallelRender processes multiple Sectors concurrently using a worker pool to render polygons onto the given surface.
func (r *RenderSoftware) parallelRender(surface *pixels.PictureRGBA, vi *ViewItem, css []*model.CompiledSector, compiled int) {
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
		go func(polygons []*model.CompiledPolygon) {
			//TODO each renderer must have a separate DrawPolygon
			dp := NewDrawPolygon(r.screenWidth, r.screenHeight)
			for k := len(polygons) - 1; k >= 0; k-- {
				cp := polygons[k]
				dp.Setup(surface, cp.Points, cp.PLen, cp.Kind, cp.Light1, cp.Light2)
				r.renderPolygon(vi, cp, dp, mode)
			}
			wg.Done()
		}(css[idx].Get())
	}
	wg.Wait()
}

// renderPolygon processes and renders a polygon based on its type, rendering mode, and associated view and draw context.
func (r *RenderSoftware) renderPolygon(vi *ViewItem, cp *model.CompiledPolygon, dr *DrawPolygon, mode int) {
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

	// Scala per mappare 1:1 le dimensioni reali del tuo spazio

	switch cp.Kind {
	case model.IdWall:
		target := (cp.Sector.Ceil - cp.Sector.Floor) * scaleFactor
		dr.DrawTexture(cp.Sector.WallTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, target)
	case model.IdUpper:
		target := (cp.Sector.Ceil - cp.Neighbor.Ceil) * scaleFactor
		dr.DrawTexture(cp.Sector.UpperTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, math.Abs(target))
	case model.IdLower:
		target := (cp.Neighbor.Floor - cp.Sector.Floor) * scaleFactor
		dr.DrawTexture(cp.Sector.LowerTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, math.Abs(target))
	case model.IdCeil:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
	case model.IdFloor:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
	case model.IdFloorTest:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
	case model.IdCeilTest:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
	default:
		dr.DrawWireFrame(true)
	}
}

// renderPolygon processes and renders a polygon based on its type, rendering mode, and associated view and draw context.
func (r *RenderSoftware) renderPolygon_OLD(vi *ViewItem, cp *model.CompiledPolygon, dr *DrawPolygon, mode int) {
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

	switch cp.Kind {
	case model.IdWall:
		target := cp.Sector.Ceil - cp.Sector.Floor
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.WallTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef)
	case model.IdUpper:
		target := cp.Sector.Ceil - cp.Neighbor.Ceil
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.UpperTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef)
	case model.IdLower:
		target := cp.Sector.Floor - cp.Neighbor.Floor
		yRef := r.sectorsMaxHeight
		if target > 1 {
			yRef = r.sectorsMaxHeight / target
		}
		dr.DrawTexture(cp.Sector.LowerTexture, cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef)
	case model.IdCeil:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(cp.Sector.CeilTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case model.IdFloor:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case model.IdFloorTest:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.FloorTexture, cp.Sector.Floor)
		//dr.DrawTexture(cp.Sector.FloorTexture, cp.x1, cp.x2, cp.tz1, cp.tz2, cp.u0, cp.u1, 1.0)
	case model.IdCeilTest:
		dr.DrawPerspectiveTexture(vi.Where.X, vi.Where.Y, vi.Where.Z, vi.Yaw, vi.AngleSin, vi.AngleCos, cp.Sector.CeilTexture, cp.Sector.Ceil)
		//dr.DrawTexture(p.Sector.CeilTexture, p.x1, p.x2, p.tz1, p.tz2, p.u0, p.u1, 1.0)
	default:
		dr.DrawWireFrame(true)
	}
}
