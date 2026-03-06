package open_gl

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"

	"github.com/go-gl/gl/v3.3-core/gl"
	// "github.com/go-gl/gl/v4.1-core/gl" // Da decommentare per i binding nativi
)

const _scale = 1

type glTexture struct {
	hwId uint32
}

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

	enableClear   bool
	debug         bool
	debugIdx      int
	shaderProgram uint32

	glTextures map[*textures.Texture]*glTexture
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
		bufferCapacity:   65536 * 32,
		enableClear:      false,
		debug:            false,
		debugIdx:         0,
		shaderProgram:    0,
	}
	//r.initGL()
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

func (w *RenderOpenGL) compileShaderProgram() {
	vertexShaderSource := shaderVertex
	fragmentShaderSource := shaderFragment
	vertexShader, err := w.compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := w.compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}
	defer gl.DeleteShader(fragmentShader)

	w.shaderProgram = gl.CreateProgram()
	gl.AttachShader(w.shaderProgram, vertexShader)
	gl.AttachShader(w.shaderProgram, fragmentShader)
	gl.LinkProgram(w.shaderProgram)

	var status int32
	gl.GetProgramiv(w.shaderProgram, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		panic("Failed to link shader program")
	}

	gl.UseProgram(w.shaderProgram)

	w.compileTextures()
}

func (w *RenderOpenGL) compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	cSources, free := gl.Strs(source + "\x00")
	gl.ShaderSource(shader, 1, cSources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("failed to compile shader: %v", log)
	}
	return shader, nil
}

func (w *RenderOpenGL) compileTextures() {
	w.glTextures = make(map[*textures.Texture]*glTexture)
	for _, id := range w.textures.GetNames() {
		tex := w.textures.Get(id)
		if tex == nil {
			continue
		}
		glTex := &glTexture{hwId: 0}

		width, height := tex.Size()
		width = width + 1
		height = height + 1
		glPixels := make([]uint8, width*height*4)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				c := tex.Get(x, y)
				idx := (y*width + x) * 4
				glPixels[idx] = uint8(c >> 16)           // R
				glPixels[idx+1] = uint8((c >> 8) & 0xFF) // G
				glPixels[idx+2] = uint8(c & 0xFF)        // B
				glPixels[idx+3] = 255                    // A
			}
		}
		gl.GenTextures(1, &glTex.hwId)
		gl.BindTexture(gl.TEXTURE_2D, glTex.hwId)

		w.glTextures[tex] = glTex

		// Setup parametri di campionamento (GL_NEAREST preserva il look pixel-art nativo)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

		// Upload Host-To-Device
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(glPixels))
	}
}

//func (w *RenderOpenGL) initGL() {
// Setup VAO e VBO in modalità GL_DYNAMIC_DRAW
// gl.GenVertexArrays(1, &w.vao)
// gl.BindVertexArray(w.vao)
// gl.GenBuffers(1, &w.vbo)
// gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
// gl.BufferData(gl.ARRAY_BUFFER, w.bufferCapacity, nil, gl.DYNAMIC_DRAW)
// Layout: Layout 0 (Pos 3D), Layout 1 (UV), Layout 2 (Light/Normals)
//}

/*
func (w *RenderOpenGL) initGL() {
	if err := gl.Init(); err != nil {
		panic(err)
	}

	gl.GenVertexArrays(1, &w.vao)
	gl.BindVertexArray(w.vao)

	gl.GenBuffers(1, &w.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
	// Buffer dinamico pre-allocato
	gl.BufferData(gl.ARRAY_BUFFER, w.bufferCapacity, nil, gl.DYNAMIC_DRAW)

	// Layout: X, Y, Z (3x Float32), U, V (2x Float32), LightDist (1x Float32)
	stride := int32(6 * 4)

	// Position
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	// UVs
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)

	// Light
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	gl.Enable(gl.CULL_FACE) // Fondamentale, il nodebuilder o la CDT devono garantire il winding CCW
}
*/

func (w *RenderOpenGL) updateCameraUniforms(vi *model.ViewItem) {
	gl.UseProgram(w.shaderProgram)

	// 1. Projection Matrix (Perspective)
	fov := float32(math.Pi / 2.0) // 90 gradi
	aspect := float32(w.screenWidth) / float32(w.screenHeight)
	near, far := float32(1.0), float32(10000.0)

	f := float32(1.0 / math.Tan(float64(fov)/2.0))
	proj := [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}

	// 2. View Matrix (FPS Camera)
	// Estraiamo yaw (orizzontale) e pitch (verticale, memorizzato in vi.Yaw nel tuo modello software)
	pitch := float32(vi.Yaw * 0.1) // Fattore di scala empirico da testare

	// Rotazioni Euleriane combinate (traslazione -> yaw -> pitch -> swap assi Z/Y)
	cosPitch, sinPitch := float32(math.Cos(float64(pitch))), float32(math.Sin(float64(pitch)))
	cosYaw, sinYaw := float32(vi.AngleCos), float32(vi.AngleSin) // Sin/Cos precalcolati dal player

	// Matrice di View (Swappa Z con Y per allineare l'asse verticale di Doom a quello di OpenGL)
	view := [16]float32{
		sinYaw, cosPitch * cosYaw, sinPitch * cosYaw, 0,
		-cosYaw, cosPitch * sinYaw, sinPitch * sinYaw, 0,
		0, -sinPitch, cosPitch, 0,
		0, 0, 0, 1,
	}

	// Applica traslazione inversa (Camera Position)
	tx, ty, tz := float32(-vi.Where.X), float32(-vi.Where.Y), float32(-vi.Where.Z)

	// Moltiplicazione Traslazione * Rotazione (ottimizzata in-place per la colonna 3)
	view[12] = view[0]*tx + view[4]*ty + view[8]*tz
	view[13] = view[1]*tx + view[5]*ty + view[9]*tz
	view[14] = view[2]*tx + view[6]*ty + view[10]*tz
	view[15] = 1.0

	viewLoc := gl.GetUniformLocation(w.shaderProgram, gl.Str("u_view\x00"))
	gl.UniformMatrix4fv(viewLoc, 1, false, &view[0])

	projLoc := gl.GetUniformLocation(w.shaderProgram, gl.Str("u_projection\x00"))
	gl.UniformMatrix4fv(projLoc, 1, false, &proj[0])
}

func (w *RenderOpenGL) streamRender(css []*model.CompiledSector, compiled int, vi *model.ViewItem) {
	batchMap := make(map[uint32][]float32)

	for idx := compiled - 1; idx >= 0; idx-- {
		if w.targetEnabled {
			if f, _ := w.targetSectors[idx]; f && w.targetId != css[idx].Sector.Id {
				w.targetId = css[idx].Sector.Id
			}
		}
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]

			// Fallback o selezione texture in base alla semantica
			texID := w.glTextures[cp.Texture] // //cp.Texture
			if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
				texID = w.glTextures[cp.Sector.TextureCeil]
			} else if cp.Kind == model.IdFloor || cp.Kind == model.IdFloorTest {
				texID = w.glTextures[cp.Sector.TextureFloor]
			}

			// Triangolazione logica su stream lineare
			var vertices []float32
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				vertices = w.buildWallQuad(cp)
			case model.IdCeil, model.IdFloor, model.IdCeilTest, model.IdFloorTest:
				vertices = w.buildFlatPoly(cp)
			}

			if len(vertices) > 0 {
				batchMap[texID.hwId] = append(batchMap[texID.hwId], vertices...)
			}
		}
	}

	gl.BindVertexArray(w.vao)

	for hwTex, vertices := range batchMap {
		if len(vertices) == 0 {
			continue
		}

		// Binding ID hardware (serve l'integrazione con textures.ITextures)
		//hwTex := w.textures.GetHardwareId(texID)
		gl.BindTexture(gl.TEXTURE_2D, hwTex)

		byteSize := len(vertices) * 4

		gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
		// Upload in-place della memoria host-to-device.
		// gl.BufferData con gl.DYNAMIC_DRAW e poi SubData previene le rilocazioni hardware.
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteSize, gl.Ptr(vertices))

		vertexCount := int32(len(vertices) / 6)
		gl.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	}
}

// buildWallQuad costruisce 2 triangoli (6 vertici) per il segmento verticale.
func (w *RenderOpenGL) buildWallQuad(cp *model.CompiledPolygon) []float32 {
	// L'assegnazione di Z1/Z2 dipende dal tipo di edge (muro pieno, upper gap o lower gap)
	var zBottom, zTop float64

	switch cp.Kind {
	case model.IdWall:
		zBottom, zTop = cp.Sector.Floor, cp.Sector.Ceil
	case model.IdUpper:
		zBottom, zTop = cp.Neighbor.Ceil, cp.Sector.Ceil
	case model.IdLower:
		zBottom, zTop = cp.Sector.Floor, cp.Neighbor.Floor
	}

	// Mappatura UV base, da scalare con cp.Sector.TextureScaleFactor lato shader o qua.
	u0, u1 := cp.U0, cp.U1
	v0, v1 := 0.0, (zTop - zBottom) // O map ratio esatto

	light := float32(cp.Sector.LightDistance)

	// Due triangoli CCW: A-B-C, A-C-D
	// Assumendo cp.Points[0] e cp.Points[1] siano Start ed End del segmento.
	if len(cp.Points) < 2 {
		return nil
	}
	p1, p2 := cp.Points[0], cp.Points[1]

	return []float32{
		float32(p1.X), float32(p1.Y), float32(zTop), float32(u0), float32(v1), light,
		float32(p1.X), float32(p1.Y), float32(zBottom), float32(u0), float32(v0), light,
		float32(p2.X), float32(p2.Y), float32(zBottom), float32(u1), float32(v0), light,

		float32(p1.X), float32(p1.Y), float32(zTop), float32(u0), float32(v1), light,
		float32(p2.X), float32(p2.Y), float32(zBottom), float32(u1), float32(v0), light,
		float32(p2.X), float32(p2.Y), float32(zTop), float32(u1), float32(v1), light,
	}
}

// buildFlatPoly tessella i poligoni planari estratti dalla CDT.
func (w *RenderOpenGL) buildFlatPoly(cp *model.CompiledPolygon) []float32 {
	if len(cp.Points) < 3 {
		return nil
	}

	z := cp.Sector.Floor
	if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
		z = cp.Sector.Ceil
	}

	light := float32(cp.Sector.LightDistance)
	var stream []float32

	// Assumendo che cp.Points contenga già una lista lineare di triangoli (multipli di 3)
	// dalla tua Constrained Delaunay Triangulation.
	for _, p := range cp.Points {
		// UV mapping banale su piani XY: U = X, V = Y.
		stream = append(stream,
			float32(p.X), float32(p.Y), float32(z),
			float32(p.X), float32(p.Y),
			light,
		)
	}

	return stream
}

func (w *RenderOpenGL) doInitialize() {
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
	if err = gl.Init(); err != nil {
		panic(err)
	}

	gl.GenVertexArrays(1, &w.vao)
	gl.BindVertexArray(w.vao)

	gl.GenBuffers(1, &w.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, w.bufferCapacity, nil, gl.DYNAMIC_DRAW)

	stride := int32(6 * 4) // X,Y,Z (3) + U,V (2) + Light (1)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	w.compileShaderProgram()
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
