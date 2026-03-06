package open_gl

import (
	"fmt"
	"image/color"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
	// "github.com/go-gl/gl/v4.1-core/gl" // Da decommentare per i binding nativi
)

const _scale = 1

// RenderOpenGL gestisce la pipeline hardware, sfruttando la topologia
// strict-sector e il view-window clipping lato CPU per iniettare nella GPU
// esclusivamente la geometria visibile tramite VBO dinamici.
type RenderOpenGL struct {
	portal           *portal.Portal
	vi               *model.ViewItem
	player           *model.Player
	textures         textures.ITextures
	win              *pixels.GLWindow
	screenWidth      int
	screenHeight     int
	sectorsMaxHeight float64

	targetSectors      map[int]bool
	targetIdx          int
	targetLastCompiled int
	targetEnabled      bool
	targetId           string

	// Hardware Context
	vao            uint32
	vbo            uint32
	bufferCapacity int

	enableClear bool
	debug       bool
	debugIdx    int
}

// NewOpenGLRender inizializza il renderer hardware allocando la VRAM necessaria
// per lo stream dinamico dei vertici frame-by-frame.
func NewOpenGLRender() *RenderOpenGL {
	r := &RenderOpenGL{
		portal:           nil,
		vi:               nil,
		textures:         nil,
		player:           nil,
		win:              nil,
		screenWidth:      0,
		screenHeight:     0,
		sectorsMaxHeight: 0,
		targetSectors:    map[int]bool{0: true},
		bufferCapacity:   65536 * 32, // Dimensione preallocata per il Vertex Buffer (es. Float32 * attributi)
		enableClear:      false,
		debug:            false,
		debugIdx:         0,
	}
	r.initGL()
	return r
}

func (w *RenderOpenGL) Setup(portal *portal.Portal, player *model.Player, t textures.ITextures) error {
	w.portal = portal
	w.screenWidth = portal.ScreenWidth()
	w.screenHeight = portal.ScreenHeight()
	w.sectorsMaxHeight = portal.SectorsMaxHeight()
	w.player = player
	w.textures = t
	return nil
}

func (w *RenderOpenGL) initGL() {
	// Setup VAO e VBO in modalità GL_DYNAMIC_DRAW
	// gl.GenVertexArrays(1, &w.vao)
	// gl.BindVertexArray(w.vao)
	// gl.GenBuffers(1, &w.vbo)
	// gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
	// gl.BufferData(gl.ARRAY_BUFFER, w.bufferCapacity, nil, gl.DYNAMIC_DRAW)

	// Layout: Layout 0 (Pos 3D), Layout 1 (UV), Layout 2 (Light/Normals)
}

func (w *RenderOpenGL) updateCameraUniforms(vi *model.ViewItem) {
	// Trasmissione al Vertex Shader della Posizione (vi.Where),
	// Pitch (vi.Yaw), e Yaw (vi.AngleSin, vi.AngleCos) come matrici 4x4.
}

// streamRender mappa i poligoni logici su array di vertici, raggruppandoli
// per texture ID per minimizzare le transizioni di stato (Context Switch) in OpenGL.
func (w *RenderOpenGL) streamRender(css []*model.CompiledSector, compiled int, vi *model.ViewItem) {
	// batchMap := make(map[string][]float32)

	for idx := compiled - 1; idx >= 0; idx-- {
		if w.targetEnabled {
			if f, _ := w.targetSectors[idx]; f && w.targetId != css[idx].Sector.Id {
				w.targetId = css[idx].Sector.Id
				fmt.Println("GL Active Sector:", w.targetId)
			}
		}

		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]

			// lightAmbientDist := vi.LightDistance
			// lightDist := cp.Sector.LightDistance

			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				// Costruzione vertici verticali usando p.X1, p.X2 e quote cp.Sector.Floor/Ceil
				// textureId := cp.Texture
				// append(batchMap[textureId], cp.Points3D...)

			case model.IdCeil, model.IdFloor:
				// Costruzione vertici planari topologici
				// tex := cp.Sector.TextureFloor // (o Ceil)
				// append(batchMap[tex], cp.Points3D...)
			}
		}
	}

	// Flush dei batch sulla pipeline hardware
	// gl.BindVertexArray(w.vao)
	// for texID, vertices := range batchMap {
	//    gl.BindTexture(gl.TEXTURE_2D, texID)
	//    gl.BufferSubData(gl.ARRAY_BUFFER, 0, len(vertices)*4, gl.Ptr(vertices))
	//    gl.DrawArrays(gl.TRIANGLES, 0, int32(len(vertices)/vertexSize))
	// }
}

func (w *RenderOpenGL) doInitialize() {
	//VIEWMODE = -1 = Normal, 0 = Wireframe, 1 = Flat, 2 = Wireframe
	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(w.screenWidth)*_scale, float64(w.screenHeight)*_scale),
		VSync:       true,
		Undecorated: false,
		Smooth:      false,
	}
	var err error
	w.win, err = pixels.NewGLWindow(cfg)
	if err != nil {
		panic(err)
	}
	//center := w.win.Bounds().Center()

	//w.mainMatrix = pixels.IM.Moved(center).Scaled(center, _scale)

	//if w.enableClear {
	//	fmt.Println("WIN CLEAR IS ENABLE - DISABLE WHEN COMPLETE!!!!!!!!!!")
	//}
}

func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

func (w *RenderOpenGL) doRun() {
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

		var up, down, left, right, slow bool

		scroll := w.win.MouseScroll()
		if scroll.Y != 0 {
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
				slow = true
			case pixels.KeyDown:
				down = true
			case pixels.KeyS:
				down = true
				slow = true
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

		w.doPlayerMoves(up, down, left, right, slow)
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
		if w.win.JustPressed(pixels.KeySpace) || w.win.Pressed(pixels.MouseButton1) {
			w.doPlayerJump()
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

// Update updates the state of the world, including the player and rendering, based on the current frame's conditions.
func (w *RenderOpenGL) doRender() {
	if w.enableClear {
		w.win.Clear(color.Black)
	}

	_, w.vi.AngleSin, w.vi.AngleCos = w.player.GetAngle()
	w.vi.Sector = w.player.GetSector()
	w.vi.Where.X, w.vi.Where.Y = w.player.GetCoords()
	w.vi.Where.Z = w.player.GetZ()
	w.vi.Yaw = w.player.GetYaw()
	w.vi.LightDistance = w.player.GetLightDistance()

	cs, count := w.portal.Compile(w.vi)
	w.targetLastCompiled = count

	// 1. Setup Uniforms (Matrici di View e Projection dal vi)
	w.updateCameraUniforms(w.vi)
	// 2. Traversal e allocazione buffer
	w.streamRender(cs, count, w.vi)

	w.player.Compute(w.vi)
}

// DoPlayerDuckingToggle toggles the ducking state of the player.
func (w *RenderOpenGL) doPlayerDuckingToggle() {
	w.player.SetDucking()
}

// DoPlayerJump makes the player perform a jump by altering vertical velocity and setting the falling state.
func (w *RenderOpenGL) doPlayerJump() {
	w.player.SetJump()
}

// DoPlayerMoves processes the player's movement based on directional input (up, down, left, right) and movement speed (slow).
func (w *RenderOpenGL) doPlayerMoves(up bool, down bool, left bool, right bool, slow bool) {
	w.player.Move(up, down, left, right, slow)
}

// doPlayerMouseMove adjusts the player's viewing angle and yaw based on mouse movement within a constrained range.
func (w *RenderOpenGL) doPlayerMouseMove(mouseX float64, mouseY float64) {
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

	w.player.MoveApply(0, 0)
}

// DoZoom adjusts the current zoom level of the view by adding the specified zoom value.
func (w *RenderOpenGL) doZoom(zoom float64) {
	w.vi.Zoom += zoom
}

// DoDebug toggles the debug mode or switches the debug sector based on the provided index increment.
func (w *RenderOpenGL) doDebug(next int) {
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= len(w.portal.Sectors) {
		return
	}
	w.debugIdx = idx
	sector := w.portal.Sectors[idx]
	x := sector.Segments[0].Start.X
	y := sector.Segments[0].Start.Y
	fmt.Println("CURRENT DEBUG IDX:", w.debugIdx, "total segments:", len(sector.Segments))
	w.player.SetSector(sector)
	w.player.SetCoords(x+5, y+5)
}

// DebugMoveSectorToggle toggles the Sector targeting mode by enabling or disabling the targetEnabled flag.
func (w *RenderOpenGL) doDebugMoveSectorToggle() {
	w.targetEnabled = !w.targetEnabled
}

// DebugMoveSector updates the target Sector index based on the direction and adjusts the active state of Sectors accordingly.
func (w *RenderOpenGL) doDebugMoveSector(forward bool) {
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
