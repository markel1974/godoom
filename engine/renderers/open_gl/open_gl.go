package open_gl

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
)

/*

// TODO

Utilizzare LightCenter del Settore

Materiali Avanzati (PBR leggero): Aggiungere Normal mapping e Specular mapping generando on-the-fly le normal map dalle texture dei WAD, per dare volume e riflessi dinamici ai mattoni sotto le point light.

Post-Processing: Implementare un pass di SSAO (Screen Space Ambient Occlusion) per scurire realisticamente gli angoli dei settori, o un Bloom HDR per far brillare le zone illuminate.

Completamento Engine: Adattare pushFlat per ricevere i 12 float (inclusa la luce del settore) e passare al rendering degli sprite/entità con billboarding istanziato.
*/

// _scale is a constant used as a multiplier to define scaling factors for rendering configurations.
const _scale = 1

// maxBatchVertices defines the maximum number of vertices allowed in a single batch for rendering operations.
const maxBatchVertices = 65536 * 2

// drawCmd represents a single drawing command, storing texture ID and vertex range information for rendering.
type drawCmd struct {
	texID       uint32
	firstVertex int32
	vertexCount int32
}

// batchEntry represents a single batch of rendering data with a texture ID, vertex buffer, and vertex count.
// texID is the identifier for the texture associated with this batch.
// vertices is a preallocated slice storing vertex data in a tightly packed format.
// count tracks the current number of vertices added to the batch.
type batchEntry struct {
	texID    uint32
	vertices []float32 // Slice preallocato
	count    int       // Indice di riempimento attuale
}

// glTexture represents an OpenGL texture with a unique hardware ID for GPU resource management.
type glTexture struct {
	hwId uint32
}

// RenderOpenGL represents a renderer using the OpenGL framework for rendering 3D sectors and managing rendering pipelines.
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

	vao uint32
	vbo uint32

	enableClear   bool
	debug         bool
	debugIdx      int
	shaderProgram uint32

	glTextures map[*textures.Texture]*glTexture

	frameVertices []float32
	frameCmds     []*drawCmd
}

// NewOpenGLRender initializes and returns a pointer to a new RenderOpenGL instance with default parameters.
func NewOpenGLRender() *RenderOpenGL {
	r := &RenderOpenGL{
		portal:           nil,
		vi:               model.NewViewItem(),
		textures:         nil,
		player:           nil,
		win:              nil,
		screenWidth:      0,
		screenHeight:     0,
		sectorsMaxHeight: 0,
		targetSectors:    map[int]bool{0: true},
		enableClear:      false,
		debug:            false,
		debugIdx:         0,
		shaderProgram:    0,
		frameVertices:    make([]float32, 0, 1024*1024),
		frameCmds:        make([]*drawCmd, 0, 4096),
	}
	return r
}

// Setup initializes the RenderOpenGL instance with portal, player, and texture data for rendering.
func (w *RenderOpenGL) Setup(portal *portal.Portal, player *model.Player, t textures.ITextures) error {
	w.portal = portal
	w.screenWidth = portal.ScreenWidth()
	w.screenHeight = portal.ScreenHeight()
	w.sectorsMaxHeight = portal.SectorsMaxHeight()
	w.player = player
	w.textures = t
	return nil
}

// setDrawCommand assigns or creates a new drawCmd for the specified texture ID and appends it to the frame commands list.
func (w *RenderOpenGL) setDrawCommand(texID uint32) *drawCmd {
	n := len(w.frameCmds)
	if n > 0 && w.frameCmds[n-1].texID == texID {
		return w.frameCmds[n-1]
	}
	w.frameCmds = append(w.frameCmds, &drawCmd{
		texID:       texID,
		firstVertex: int32(len(w.frameVertices) / 6),
		vertexCount: 0,
	})
	return w.frameCmds[len(w.frameCmds)-1]
}

// createBatch processes a list of compiled sectors and generates vertex and draw command data for rendering.
func (w *RenderOpenGL) createBatch(css []*model.CompiledSector, compiled int) {
	w.frameVertices = w.frameVertices[:0]
	w.frameCmds = w.frameCmds[:0]

	for idx := compiled - 1; idx >= 0; idx-- {
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]
			var tex *textures.Texture
			var isWall bool

			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				tex = cp.Texture
				isWall = true
			case model.IdCeil, model.IdCeilTest:
				tex = cp.TextureCeil
			case model.IdFloor, model.IdFloorTest:
				tex = cp.TextureFloor
			default:
				continue
			}

			if tex == nil || w.glTextures[tex] == nil {
				continue
			}

			cmd := w.setDrawCommand(w.glTextures[tex].hwId)
			startLen := len(w.frameVertices)
			tW, tH := tex.Size()

			if isWall {
				var zB, zT float64
				switch cp.Kind {
				case model.IdWall:
					zB, zT = cp.Sector.Floor, cp.Sector.Ceil
				case model.IdUpper:
					zB, zT = cp.Neighbor.Ceil, cp.Sector.Ceil
				case model.IdLower:
					zB, zT = cp.Sector.Floor, cp.Neighbor.Floor
				}
				w.pushWall(cp, float32(tW), float32(tH), float32(zB), float32(zT))
			} else {
				z := cp.Sector.Ceil
				if cp.Kind == model.IdFloor || cp.Kind == model.IdFloorTest {
					z = cp.Sector.Floor
				}
				w.pushFlat(cp, z, float32(tW), float32(tH))
			}

			cmd.vertexCount += int32((len(w.frameVertices) - startLen) / 6)
		}
	}
}

func (w *RenderOpenGL) pushWall(cp *model.CompiledPolygon, texW, texH, zBottom, zTop float32) {
	//todo capire perche le texture devono essere scalate di 4 dai WAD
	scaleH := float32(4.0)
	scaleV := float32(cp.Sector.TextureScaleFactor)
	if scaleV <= 0 {
		scaleV = 1.0
	}

	// UV Orizzontali (Nessuna moltiplicazione: ereditano già la scala dai segmenti XY)
	u0 := float32(cp.U0) / (texW * scaleH)
	u1 := float32(cp.U1) / (texW * scaleH)

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / texH) * scaleV
	light := float32(cp.Sector.LightDistance)
	sin, cos := w.vi.AngleSin, w.vi.AngleCos
	wx1 := float32((cp.Tx1 * sin) + (cp.Tz1 * cos) + w.vi.Where.X)
	wy1 := float32(-(cp.Tx1 * cos) + (cp.Tz1 * sin) + w.vi.Where.Y)
	wx2 := float32((cp.Tx2 * sin) + (cp.Tz2 * cos) + w.vi.Where.X)
	wy2 := float32(-(cp.Tx2 * cos) + (cp.Tz2 * sin) + w.vi.Where.Y)

	w.frameVertices = append(w.frameVertices,
		wx1, zTop, -wy1, u0, vTop, light,
		wx1, zBottom, -wy1, u0, vBottom, light,
		wx2, zBottom, -wy2, u1, vBottom, light,

		wx1, zTop, -wy1, u0, vTop, light,
		wx2, zBottom, -wy2, u1, vBottom, light,
		wx2, zTop, -wy2, u1, vTop, light,
	)
}

func (w *RenderOpenGL) pushFlat(cp *model.CompiledPolygon, z float64, texW, texH float32) {
	segs := cp.Sector.Segments
	if len(segs) < 3 {
		return
	}

	// 1. Allinea il fattore di scala anche per pavimenti e soffitti
	scale := float32(cp.Sector.TextureScaleFactor)
	if scale <= 0 {
		scale = 1.0
	}

	light := float32(cp.Sector.LightDistance)
	zF := float32(z)
	v0 := segs[0].Start

	// 2. Applica la scala alle UV
	u0 := (float32(v0.X) / texW) * scale
	v0V := (float32(-v0.Y) / texH) * scale

	for i := 1; i < len(segs)-1; i++ {
		v1, v2 := segs[i].Start, segs[i+1].Start

		u1 := (float32(v1.X) / texW) * scale
		v1V := (float32(-v1.Y) / texH) * scale
		u2 := (float32(v2.X) / texW) * scale
		v2V := (float32(-v2.Y) / texH) * scale

		w.frameVertices = append(w.frameVertices,
			float32(v0.X), zF, float32(-v0.Y), u0, v0V, light,
			float32(v1.X), zF, float32(-v1.Y), u1, v1V, light,
			float32(v2.X), zF, float32(-v2.Y), u2, v2V, light,
		)
	}
}

// glStreamRender uploads vertex data dynamically to the GPU and renders frame commands using OpenGL.
func (w *RenderOpenGL) glStreamRender() {
	if len(w.frameVertices) == 0 {
		return
	}

	gl.BindVertexArray(w.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)

	// Single Upload DMA: Nessuna collisione, il driver non ha motivo di bloccare la CPU.
	gl.BufferData(gl.ARRAY_BUFFER, len(w.frameVertices)*4, gl.Ptr(w.frameVertices), gl.DYNAMIC_DRAW)

	for _, cmd := range w.frameCmds {
		if cmd.vertexCount > 0 {
			gl.BindTexture(gl.TEXTURE_2D, cmd.texID)
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}

// vboMaxFloats defines the maximum number of float values allocated for the vertex buffer object (VBO) in the GPU memory.
const vboMaxFloats = 1024 * 1024 * 4

// glInit initializes OpenGL resources for rendering, including VAO, VBO, and buffer data, and sets up attribute pointers.
func (w *RenderOpenGL) glInit() error {
	gl.GenVertexArrays(1, &w.vao)
	gl.BindVertexArray(w.vao)

	gl.GenBuffers(1, &w.vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)

	// Alloca il mega-buffer in VRAM senza inizializzare i dati
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)

	stride := int32(6 * 4)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	return nil
}

// glCompileShaderProgram compiles vertex and fragment shaders, links them into a program, and makes it the active program.
func (w *RenderOpenGL) glCompileShaderProgram() error {
	vertexShaderSource := shaderVertex
	fragmentShaderSource := shaderFragment
	vertexShader, err := w.glCompileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return err
	}
	defer gl.DeleteShader(vertexShader)

	fragmentShader, err := w.glCompileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return err
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
	return nil
}

// glCompileShader compiles the given GLSL shader source code for the specified shader type and returns the shader object ID.
func (w *RenderOpenGL) glCompileShader(source string, shaderType uint32) (uint32, error) {
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

// glCompileTextures initializes and compiles textures from a texture manager, storing OpenGL texture metadata.
func (w *RenderOpenGL) glCompileTextures() error {
	w.glTextures = make(map[*textures.Texture]*glTexture)
	for _, id := range w.textures.GetNames() {
		tn := w.textures.Get([]string{id})
		if tn == nil {
			continue
		}
		tex := tn[0]
		glTex := &glTexture{hwId: 0}
		width, height, glPixels := tex.RGBA()
		gl.GenTextures(1, &glTex.hwId)
		gl.BindTexture(gl.TEXTURE_2D, glTex.hwId)
		w.glTextures[tex] = glTex

		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(width), int32(height), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(glPixels))

		gl.GenerateMipmap(gl.TEXTURE_2D)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

		// 2. Tenta il filtro anisotropico via brute-force (valore 0x84FE)
		// Questo rimuove il blur "fangoso" sulle texture dei muri viste radenti
		gl.TexParameterf(gl.TEXTURE_2D, 0x84FE, 4.0)

		//var maxAnisotropy float32
		//gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY_EXT, &maxAnisotropy)
		//gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAX_ANISOTROPY_EXT, maxAnisotropy)
	}
	return nil
}

// glUpdateCameraUniforms updates the camera's view and projection matrices for the current OpenGL shader program.
func (w *RenderOpenGL) glUpdateCameraUniforms(vi *model.ViewItem) {
	gl.UseProgram(w.shaderProgram)

	aspect := float32(w.screenWidth) / float32(w.screenHeight)
	near, far := float32(1.0), float32(100000.0)

	// Aggiungiamo il segno MENO a scaleX per invertire l'asse orizzontale
	// ed emulare nativamente la logica di draw_polygon.go (halfW - pixelX)
	scaleX := float32(-(2.0 / float64(aspect)) * model.HFov)
	scaleY := float32(2.0 * model.VFov)

	pitchShear := float32(-vi.Yaw)

	proj := [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}

	cosA, sinA := float32(vi.AngleCos), float32(vi.AngleSin)

	fX, fZ := cosA, -sinA
	rX, rZ := sinA, cosA

	ex := float32(vi.Where.X)
	ey := float32(vi.Where.Z)
	ez := float32(-vi.Where.Y)

	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := fX*ex + fZ*ez

	view := [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}

	gl.UniformMatrix4fv(gl.GetUniformLocation(w.shaderProgram, gl.Str("u_view\x00")), 1, false, &view[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(w.shaderProgram, gl.Str("u_projection\x00")), 1, false, &proj[0])
	gl.Uniform1f(gl.GetUniformLocation(w.shaderProgram, gl.Str("u_ambient_light\x00")), float32(vi.LightDistance))
	timeLoc := gl.GetUniformLocation(w.shaderProgram, gl.Str("u_time\x00"))
	gl.Uniform1f(timeLoc, float32(pixels.GLGetTime()))
}

// doInitialize initializes the OpenGL renderer, sets up the window, compiles shaders, and loads textures.
func (w *RenderOpenGL) doInitialize() error {
	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(w.screenWidth)*_scale, float64(w.screenHeight)*_scale),
		VSync:       true,
		Undecorated: false,
		Smooth:      false,
	}
	var winErr error
	w.win, winErr = pixels.NewGLWindow(cfg)
	if winErr != nil {
		return winErr
	}

	thErr := executor.Thread.CallErr(func() error {
		w.win.Begin()
		if err := w.glInit(); err != nil {
			return err
		}
		if err := w.glCompileShaderProgram(); err != nil {
			return err
		}
		if err := w.glCompileTextures(); err != nil {
			return err
		}
		return nil
	})

	if thErr != nil {
		return thErr
	}
	return nil
}

// Start initializes and runs the OpenGL rendering workflow for the RenderOpenGL instance.
func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

// doRun starts the main rendering and input handling loop for the OpenGL context and manages player interactions.
func (w *RenderOpenGL) doRun() {
	if err := w.doInitialize(); err != nil {
		panic(err)
	}

	mouseConnected := true

	for !w.win.Closed() {
		w.doRender()

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
			case pixels.KeyUp, pixels.KeyW:
				up = true
				if v == pixels.KeyW {
					slow = true
				}
			case pixels.KeyDown, pixels.KeyS:
				down = true
				if v == pixels.KeyS {
					slow = true
				}
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

		w.win.UpdateInputAndSwap()
	}
}

// doRender performs the rendering process for the OpenGL context, updating visual components based on player state and view info.
func (w *RenderOpenGL) doRender() {
	_, w.vi.AngleSin, w.vi.AngleCos = w.player.GetAngle()
	w.vi.Sector = w.player.GetSector()
	w.vi.Where.X, w.vi.Where.Y = w.player.GetCoords()
	w.vi.Where.Z = w.player.GetZ()
	w.vi.Yaw = w.player.GetYaw()
	w.vi.LightDistance = w.player.GetLightDistance()

	cs, count := w.portal.Compile(w.vi)
	w.targetLastCompiled = count
	w.createBatch(cs, count)

	executor.Thread.Call(func() {
		w.win.Begin()
		fbW, fbH := w.win.GetFramebufferSize()
		gl.Viewport(0, 0, int32(fbW), int32(fbH))
		//gl.Viewport(0, 0, int32(w.screenWidth*2), int32(w.screenHeight*2))
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		w.glUpdateCameraUniforms(w.vi)
		w.glStreamRender()
	})

	w.player.Compute(w.vi)
}

// doPlayerDuckingToggle toggles the player's ducking state by invoking the SetDucking method on the player instance.
func (w *RenderOpenGL) doPlayerDuckingToggle() { w.player.SetDucking() }

// doPlayerJump triggers the player's jump action by invoking the SetJump method on the player object.
func (w *RenderOpenGL) doPlayerJump() { w.player.SetJump() }

// doPlayerMoves processes player movement based on directional and speed modifiers provided as boolean parameters.
func (w *RenderOpenGL) doPlayerMoves(up bool, down bool, left bool, right bool, slow bool) {
	w.player.Move(up, down, left, right, slow)
}

// doPlayerMouseMove adjusts the player's view angle and yaw based on mouse movement within clamped boundaries.
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

// doZoom adjusts the current zoom level of the RenderOpenGL object by adding the given zoom value to it.
func (w *RenderOpenGL) doZoom(zoom float64) { w.vi.Zoom += zoom }

// doDebug toggles debug mode or navigates through sectors based on the provided next parameter.
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
	w.player.SetSector(sector)
	w.player.SetCoords(sector.Segments[0].Start.X+5, sector.Segments[0].Start.Y+5)
}

// doDebugMoveSectorToggle toggles the state of the targetEnabled field within the RenderOpenGL instance.
func (w *RenderOpenGL) doDebugMoveSectorToggle() { w.targetEnabled = !w.targetEnabled }

// doDebugMoveSector updates the target sector index based on the movement direction and flag state for rendering debug.
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
