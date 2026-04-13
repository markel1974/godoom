package software

import (
	"fmt"
	"image/color"
	"math"
	"strings"
	"sync"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/model/geometry"
	"github.com/markel1974/godoom/pixels"
)

// Render is a struct responsible for handling software-based 2D rendering functionality.
type Render struct {
	win                *pixels.GLWindow
	mainSurface        *pixels.PictureRGBA
	mainMatrix         pixels.Matrix
	mainSprite         *pixels.Sprite
	enableClear        bool
	viewMode           int
	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string
	dp                 *DrawPolygon
	engine             *engine.Engine
	vi                 *model.ViewMatrix
	player             *model.ThingPlayer
	w                  int32
	h                  int32
	debug              bool
	debugIdx           int
}

// NewRender initializes and returns a new instance of Render with default values.
func NewRender(w, h int32) *Render {
	return &Render{
		w:                  w,
		h:                  h,
		targetIdx:          0,
		targetSectors:      map[int]bool{0: true},
		targetLastCompiled: 0,
		targetEnabled:      false,
		dp:                 nil,
		vi:                 model.NewViewMatrix(),
	}
}

// Setup initializes the Render instance with the specified portal, player, and textures.
func (w *Render) Setup(engine *engine.Engine) error {
	w.engine = engine
	w.dp = NewDrawPolygon(int(w.w), int(w.h))
	w.player = engine.GetPlayer()
	w.viewMode = -1
	w.enableClear = false
	return nil
}

// doInitialize initializes the rendering software window, surfaces, and matrices required for rendering.
// Configures the pixel window with predefined settings and handles any initialization errors.
// Sets up the main rendering surface, sprite, and transformation matrix for rendering.
// Logs a message if the window clear feature is enabled.
func (w *Render) doInitialize() {
	//VIEWMODE = -1 = Normal, 0 = Wireframe, 1 = Flat, 2 = Wireframe
	cfg := pixels.WindowConfig{
		Bounds:             pixels.R(0, 0, float64(w.w), float64(w.h)),
		VSync:              true,
		Undecorated:        false,
		Smooth:             false,
		DisableScissorTest: true,
	}
	var err error
	w.win, err = pixels.NewGLWindow(cfg)
	if err != nil {
		panic(err)
	}
	center := w.win.Bounds().Center()

	w.mainSurface = pixels.NewPictureRGBA(pixels.R(float64(0), float64(0), float64(w.w), float64(w.h)))
	w.mainSprite = pixels.NewSprite()
	w.mainSprite.SetCached(pixels.CacheModeUpdate)
	w.mainSprite.Set(w.mainSurface, w.mainSurface.Bounds())
	w.mainMatrix = pixels.IM.Moved(center).Scaled(center, 1.0)

	if w.enableClear {
		fmt.Println("WIN CLEAR IS ENABLE - DISABLE WHEN COMPLETE!!!!!!!!!!")
	}
}

// Start initializes and begins the main rendering loop for the Render instance.
func (w *Render) Start() {
	pixels.GLRun(w.doRun)
}

// doRun executes the main rendering and game loop, handling initialization, input processing, rendering, and game logic updates.
func (w *Render) doRun() {
	const framerate = 30
	const frameInterval = 1.0 / framerate

	w.doInitialize()

	var currentTimer float64
	var lastTimer float64
	mouseConnected := true

	for !w.win.Closed() {
		currentTimer = pixels.GLGetTime()
		if (currentTimer - lastTimer) >= frameInterval {
			lastTimer = currentTimer
			w.doRender()
		}

		if mouseConnected && w.win.MouseInsideWindow() {
			mousePos := w.win.MousePosition()
			mousePrevPos := w.win.MousePreviousPosition()
			if mousePos.X != mousePrevPos.X || mousePos.Y != mousePrevPos.Y {
				mouseX := mousePos.X - mousePrevPos.X
				mouseY := mousePos.Y - mousePrevPos.Y
				w.doPlayerMouseMove(mouseX, mouseY)
			}
		}

		var up, down, left, right bool

		impulse := 0.2

		if scroll := w.win.MouseScroll(); scroll.Y != 0 {
			if scroll.Y > 0 {
				up = true
			} else {
				down = true
			}
		}

		for v := range w.win.KeysPressed() {
			switch v {
			case pixels.KeyEscape:
				return
			case pixels.KeyUp:
				up = true
			case pixels.KeyW:
				up = true
				impulse = 0.01
			case pixels.KeyDown:
				down = true
			case pixels.KeyS:
				down = true
				impulse = 0.01
			case pixels.KeyLeft:
				left = true
			case pixels.KeyRight:
				right = true
			case pixels.KeyV:
				w.doDebugMoveSector(true)
			case pixels.KeyB:
				w.doDebugMoveSector(false)
			}
		}

		w.doPlayerMoves(impulse, up, down, left, right)
		if w.win.JustPressed(pixels.KeyC) {
			w.enableClear = true
			w.doDebugMoveSectorToggle()
		}
		if w.win.JustPressed(pixels.KeyZ) {
			w.doDebugMoveSector(true)
		}
		if w.win.JustPressed(pixels.KeyX) {
			w.doDebugMoveSector(false)
		}
		if w.win.JustPressed(pixels.KeyTab) || w.win.Pressed(pixels.MouseButton2) {
			w.doPlayerDuckingToggle()
		}
		if w.win.JustPressed(pixels.KeySpace) {
			w.doPlayerJump(false)
		}
		if w.win.Pressed(pixels.MouseButton1) {
			w.doPlayerJump(true)
		}
		if w.win.JustPressed(pixels.Key8) {
			w.doDebug(0)
		}
		if w.win.JustPressed(pixels.Key0) {
			w.doDebug(1)
		}
		if w.win.JustPressed(pixels.Key9) {
			w.doDebug(-1)
		}
		if w.win.JustPressed(pixels.KeyM) {
			mouseConnected = !mouseConnected
		}
		w.win.Update()
		//text.Draw(win, g.mainMatrix)
	}
}

// doRender performs the main rendering process, updating the frame buffer and drawing visible game elements on the screen.
func (w *Render) doRender() {
	if w.enableClear {
		w.win.Clear(color.Black)
		w.mainSurface = pixels.NewPictureRGBA(pixels.R(float64(0), float64(0), float64(w.w), float64(w.h)))
		w.mainSprite.Set(w.mainSurface, w.mainSurface.Bounds())
	}
	w.win.Begin()
	fbW, fbH := w.win.GetFramebufferSize()
	w.engine.Compute(w.player, w.vi)
	cs, count := w.engine.Traverse(int32(fbW), int32(fbH), w.vi)
	w.targetLastCompiled = count
	w.doSerialRender(w.mainSurface, w.vi, cs, count)
	//w.parallelRender(surface, vi, css, compiled)
	w.mainSurface.ApplyFastAA(20)
	if w.debug {
		w.drawStub()
	}
	w.mainSprite.Draw(w.win, w.mainMatrix)
}

// RenderSector renders a given sector by processing its segments and drawing polygons on the main surface.
func (w *Render) RenderSector(volume *model.Volume) {
	maxX := float64(0)
	maxY := float64(0)
	for _, v := range volume.GetFaces() {
		sStart := v.GetStart()
		sEnd := v.GetEnd()
		x1 := math.Abs(sStart.X)
		y1 := math.Abs(sStart.Y)
		x2 := math.Abs(sEnd.X)
		y2 := math.Abs(sEnd.Y)
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

	xFactor := (float64(w.w) / 2) / maxX
	yFactor := (float64(w.h) / 2) / maxY

	var t []geometry.XYZ
	for _, v := range volume.GetFaces() {
		sStart := v.GetStart()
		sEnd := v.GetEnd()
		x1 := sStart.X
		if x1 == 0 {
			x1 = 1
		}
		x1 *= xFactor
		y1 := sStart.Y
		if y1 == 0 {
			y1 = 1
		}
		y1 *= yFactor
		x2 := sEnd.X
		if x2 == 0 {
			x2 = 1
		}
		x2 *= xFactor
		y2 := sEnd.Y
		if y2 == 0 {
			y2 = 1
		}
		y2 *= yFactor
		t = append(t, geometry.XYZ{X: x1, Y: y1, Z: 0})
		t = append(t, geometry.XYZ{X: x2, Y: y2, Z: 0})
	}

	if len(t) == 0 {
		return
	}
	dp := NewDrawPolygon(640, 480)
	dp.Setup(w.mainSurface, t, len(t), 0x00ff00)
	dp.DrawPoints(10)
	dp.Color = 0xff0000
	dp.DrawLines(false)
}

// doPlayerDuckingToggle toggles the ducking state of the player by invoking the player's SetDucking method.
func (w *Render) doPlayerDuckingToggle() {
	w.player.SetDucking()
}

// doPlayerJump triggers the player's jump behavior by calling the appropriate method on the player instance.
func (w *Render) doPlayerJump(multi bool) {
	w.player.SetJump(multi)
}

// doPlayerMoves handles movement for the player based on the directional and speed inputs provided.
func (w *Render) doPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, left, right)
}

// doPlayerMouseMove adjusts the player's viewing angle and yaw based on mouse movement within predefined constraints.
func (w *Render) doPlayerMouseMove(mouseX float64, mouseY float64) {
	const offset = 10
	if mouseX > offset {
		mouseX = offset
	} else if mouseX < -offset {
		mouseX = -offset
	}
	if mouseY > offset {
		mouseY = offset
	} else if mouseY < -offset {
		mouseY = -offset
	}

	w.player.AddAngle(mouseX * 0.03)
	w.player.SetYaw(mouseY)

	//w.player.MoveApply(0, 0, 0)
}

// doDebug toggles the debug mode or enables it while navigating through sectors based on the `next` parameter value.
func (w *Render) doDebug(next int) {
	const offset = 5
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= w.engine.PortalLen() {
		return
	}
	w.debugIdx = idx
	sector := w.engine.PortalVolumeAt(idx)
	fmt.Println("CURRENT DEBUG IDX:", w.debugIdx, "total segments:", sector.GetId())

	/*
		sStart := sector.Faces[0].GetStart()
		x := sStart.X + offset
		y := sStart.Y + offset
		fmt.Println("CURRENT DEBUG IDX:", w.debugIdx, "total segments:", len(sector.GetId()))
		w.player.SetSector(sector)
		w.player.SetXY(x, y)
	*/
}

// doDebugMoveSectorToggle toggles the `targetEnabled` property, enabling or disabling sector targeting in debug mode.
func (w *Render) doDebugMoveSectorToggle() {
	w.targetEnabled = !w.targetEnabled
}

// doDebugMoveSector updates the target index for debugging sectors and refreshes the active target states.
func (w *Render) doDebugMoveSector(forward bool) {
	if forward {
		if w.targetIdx < w.targetLastCompiled {
			w.targetIdx++
		}
	} else {
		if w.targetIdx > 0 {
			w.targetIdx--
		}
	}
	for k := 0; k < w.targetLastCompiled; k++ {
		w.targetSectors[k] = k == w.targetIdx
	}
}

// drawStub renders the debug sector if the current debug index is within the range of available sectors.
func (w *Render) drawStub() {
	if w.debugIdx >= 0 && w.debugIdx < w.engine.PortalLen() {
		sector := w.engine.PortalVolumeAt(w.debugIdx)
		w.RenderSector(sector)
	}
}

// doSerialRender renders compiled sectors to the provided surface in a serial manner, processing from back to front.
// surface: The target rendering surface.
// vi: ViewMatrix data containing camera position, angle, and other view parameters.
// css: A slice of CompiledVolume objects representing visible sectors for rendering.
// compiled: The number of sectors available to render, processed in reverse order.
func (w *Render) doSerialRender(surface *pixels.PictureRGBA, vi *model.ViewMatrix, css []*model.CompiledVolume, compiled int) {
	for idx := compiled - 1; idx >= 0; idx-- {
		mode := -1 //w.textures.GetViewMode()
		if w.targetEnabled {
			if f, _ := w.targetSectors[idx]; !f {
				mode = 2
			} else {
				if w.targetId != css[idx].Volume.GetId() {
					w.targetId = css[idx].Volume.GetId()
					var neighbors []string
					for _, z := range css[idx].Volume.GetFaces() {
						neighbor := z.GetNeighbor()
						if z != nil && neighbor != nil {
							neighbors = append(neighbors, neighbor.GetId())
						}
					}
					fmt.Println("Current target Sector:", w.targetId, strings.Join(neighbors, ","), css[idx].Volume.GetTag())
				}
			}
		}
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			w.dp.Setup(surface, cp.Points, cp.PLen, cp.Kind)
			w.doRenderPolygon(vi, cp, w.dp, mode)
		}
	}
}

// doParallelRender performs parallel rendering of compiled sectors using goroutines to improve rendering performance.
func (w *Render) doParallelRender(surface *pixels.PictureRGBA, vi *model.ViewMatrix, css []*model.CompiledVolume, compiled int) {
	//Experimental Render
	wg := &sync.WaitGroup{}
	wg.Add(compiled)

	for idx := compiled - 1; idx >= 0; idx-- {
		mode := -1 //w.textures.GetViewMode()
		if w.targetEnabled {
			if f, _ := w.targetSectors[idx]; !f {
				mode = 2
			}
		}
		//TODO queue
		go func(polygons []*model.CompiledPolygon) {
			//TODO each renderer must have a separate DrawPolygon
			dp := NewDrawPolygon(int(w.w), int(w.h))
			for k := len(polygons) - 1; k >= 0; k-- {
				cp := polygons[k]
				dp.Setup(surface, cp.Points, cp.PLen, cp.Kind)
				w.doRenderPolygon(vi, cp, dp, mode)
			}
			wg.Done()
		}(css[idx].Get())
	}
	wg.Wait()
}

// doRenderPolygon renders a polygon based on its type, mode, and lighting parameters, utilizing various drawing methods.
func (w *Render) doRenderPolygon(vi *model.ViewMatrix, cp *model.CompiledPolygon, dr *DrawPolygon, mode int) {
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
	lightAmbient := vi.GetLightIntensity()
	lightArtificial := cp.Volume.Light.GetIntensity()
	switch cp.Kind {
	case model.IdWall:
		_, scaleH := cp.Animation.ScaleFactor()
		yRef := (cp.Volume.GetMaxZ() - cp.Volume.GetMinZ()) * scaleH
		dr.DrawTexture(cp.Animation.CurrentFrame(), cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef, lightAmbient, lightArtificial)
	case model.IdUpper:
		_, scaleH := cp.Animation.ScaleFactor()
		yRef := math.Abs((cp.Volume.GetMaxZ() - cp.Neighbor.GetMaxZ()) * scaleH)
		dr.DrawTexture(cp.Animation.CurrentFrame(), cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef, lightAmbient, lightArtificial)
	case model.IdLower:
		_, scaleH := cp.Animation.ScaleFactor()
		yRef := math.Abs((cp.Neighbor.GetMinZ() - cp.Volume.GetMinZ()) * scaleH)
		dr.DrawTexture(cp.Animation.CurrentFrame(), cp.X1, cp.X2, cp.Tz1, cp.Tz2, cp.U0, cp.U1, yRef, lightAmbient, lightArtificial)
	case model.IdCeil:
		_, scaleH := cp.Animation.ScaleFactor()
		viX, viY, viZ := vi.GetXYZ()
		viSin, viCos := vi.GetAngle()
		viYaw := vi.GetYaw()
		dr.DrawPerspectiveTexture(viX, viY, viZ, viYaw, viSin, viCos, cp.AnimationCeil.CurrentFrame(), cp.Volume.GetMaxZ(), scaleH, lightAmbient, lightArtificial)
	case model.IdFloor:
		_, scaleH := cp.Animation.ScaleFactor()
		viX, viY, viZ := vi.GetXYZ()
		viSin, viCos := vi.GetAngle()
		viYaw := vi.GetYaw()
		dr.DrawPerspectiveTexture(viX, viY, viZ, viYaw, viSin, viCos, cp.AnimationFloor.CurrentFrame(), cp.Volume.GetMinZ(), scaleH, lightAmbient, lightArtificial)
	case model.IdFloorTest:
		_, scaleH := cp.Animation.ScaleFactor()
		viX, viY, viZ := vi.GetXYZ()
		viSin, viCos := vi.GetAngle()
		viYaw := vi.GetYaw()
		dr.DrawPerspectiveTexture(viX, viY, viZ, viYaw, viSin, viCos, cp.AnimationFloor.CurrentFrame(), cp.Volume.GetMinZ(), scaleH, lightAmbient, lightArtificial)
	case model.IdCeilTest:
		_, scaleH := cp.Animation.ScaleFactor()
		viX, viY, viZ := vi.GetXYZ()
		viSin, viCos := vi.GetAngle()
		viYaw := vi.GetYaw()
		dr.DrawPerspectiveTexture(viX, viY, viZ, viYaw, viSin, viCos, cp.AnimationCeil.CurrentFrame(), cp.Volume.GetMaxZ(), scaleH, lightAmbient, lightArtificial)
	default:
		dr.DrawWireFrame(true)
	}
}
