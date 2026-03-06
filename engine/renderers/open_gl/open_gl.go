package open_gl

import (
	"fmt"
	"math"
	"strings"

	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
	"github.com/markel1974/godoom/engine/textures"
	"github.com/markel1974/godoom/pixels"
	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
	// "github.com/go-gl/gl/v4.1-core/gl" // Da decommentare per i binding nativi
)

// _scale defines a scaling factor used for adjusting the rendering dimensions of the OpenGL window configuration.
const _scale = 1

// glTexture represents an OpenGL texture, containing the hardware ID for binding and management within the GPU.
type glTexture struct {
	hwId uint32
}

// RenderOpenGL is responsible for managing OpenGL rendering, including textures, shaders, and rendering states.
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

// NewOpenGLRender creates and initializes a new instance of RenderOpenGL for OpenGL-based rendering.
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
		bufferCapacity:   65536 * 32,
		enableClear:      false,
		debug:            false,
		debugIdx:         0,
		shaderProgram:    0,
	}
	return r
}

// Setup initializes the RenderOpenGL instance with the given portal, player, and textures, and prepares it for rendering.
func (w *RenderOpenGL) Setup(portal *portal.Portal, player *model.Player, t textures.ITextures) error {
	w.portal = portal
	w.screenWidth = portal.ScreenWidth()
	w.screenHeight = portal.ScreenHeight()
	w.sectorsMaxHeight = portal.SectorsMaxHeight()
	w.player = player
	w.textures = t
	return nil
}

// glInit initializes OpenGL by setting up the VAO, VBO, vertex attributes, and enabling depth testing.
func (w *RenderOpenGL) glInit() error {
	if err := gl.Init(); err != nil {
		return err
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
	return nil
}

// glCompileShaderProgram compiles and links vertex and fragment shaders to create an OpenGL shader program.
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

// glCompileShader compiles a shader from the provided source code and shader type, returning its ID or an error.
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

// glCompileTextures compiles all textures in the internal texture manager into OpenGL textures and uploads them to the GPU.
func (w *RenderOpenGL) glCompileTextures() error {
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
	return nil
}

// glUpdateCameraUniforms updates the camera's uniform matrices (view and projection) for the active shader program.
func (w *RenderOpenGL) glUpdateCameraUniforms(vi *model.ViewItem) {
	gl.UseProgram(w.shaderProgram)

	fov := float32(math.Pi / 3.0)
	aspect := float32(w.screenWidth) / float32(w.screenHeight)
	near, far := float32(1.0), float32(100000.0)
	f := float32(1.0 / math.Tan(float64(fov)/2.0))

	// L'asse visuale verticale di Doom si ottiene tramite Y-Shearing della matrice
	// di proiezione, non con una reale rotazione Pitch (che altererebbe i vertici Near-Z)
	pitchShear := float32(-vi.Yaw * 0.01)

	proj := [16]float32{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, pitchShear, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}

	// La View Matrix diventa pura rotazione planare (Yaw 2D) e traslazione
	cosA, sinA := float32(vi.AngleCos), float32(vi.AngleSin)

	fX, fZ := cosA, -sinA
	rX, rZ := sinA, cosA

	ex := float32(vi.Where.X)
	ey := float32(vi.Where.Z)
	ez := float32(-vi.Where.Y)

	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := (fX*ex + fZ*ez)

	view := [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}

	gl.UniformMatrix4fv(gl.GetUniformLocation(w.shaderProgram, gl.Str("u_view\x00")), 1, false, &view[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(w.shaderProgram, gl.Str("u_projection\x00")), 1, false, &proj[0])
}

func (w *RenderOpenGL) createBatch(css []*model.CompiledSector, compiled int) map[uint32][]float32 {
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
			switch cp.Kind {
			case model.IdWall, model.IdUpper, model.IdLower:
				if cp.Texture == nil {
					continue
				}
				glTex := w.glTextures[cp.Texture]
				if glTex == nil {
					continue
				}
				tWidth, tHeight := cp.Texture.Size()
				var zBottom, zTop float64
				switch cp.Kind {
				case model.IdWall:
					zBottom, zTop = cp.Sector.Floor, cp.Sector.Ceil
				case model.IdUpper:
					zBottom, zTop = cp.Neighbor.Ceil, cp.Sector.Ceil
				case model.IdLower:
					zBottom, zTop = cp.Sector.Floor, cp.Neighbor.Floor
				}
				if vertices := w.buildWallQuad(cp, float32(tWidth), float32(tHeight), float32(zBottom), float32(zTop)); len(vertices) > 0 {
					batchMap[glTex.hwId] = append(batchMap[glTex.hwId], vertices...)
				}
			case model.IdCeil, model.IdCeilTest:
				if cp.Sector.TextureCeil == nil {
					continue
				}
				glTex := w.glTextures[cp.Sector.TextureCeil]
				if glTex == nil {
					continue
				}
				tWidth, tHeight := cp.Sector.TextureCeil.Size()
				if vertices := w.buildFlatPoly(cp, cp.Sector.Ceil, float32(tWidth), float32(tHeight)); len(vertices) > 0 {
					batchMap[glTex.hwId] = append(batchMap[glTex.hwId], vertices...)
				}
			case model.IdFloor, model.IdFloorTest:
				if cp.Sector.TextureFloor == nil {
					continue
				}
				texID := w.glTextures[cp.Sector.TextureFloor]
				if texID == nil {
					continue
				}
				tWidth, tHeight := cp.Sector.TextureFloor.Size()
				if vertices := w.buildFlatPoly(cp, cp.Sector.Floor, float32(tWidth), float32(tHeight)); len(vertices) > 0 {
					batchMap[texID.hwId] = append(batchMap[texID.hwId], vertices...)
				}
			}
		}
	}
	return batchMap
}

// glStreamRender processes compiled sector data, prepares vertex buffers, and renders them using OpenGL.
func (w *RenderOpenGL) glStreamRender(batchMap map[uint32][]float32) {
	gl.BindVertexArray(w.vao)

	for hwTex, vertices := range batchMap {
		if len(vertices) == 0 {
			continue
		}

		gl.BindTexture(gl.TEXTURE_2D, hwTex)

		byteSize := len(vertices) * 4

		gl.BindBuffer(gl.ARRAY_BUFFER, w.vbo)
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteSize, gl.Ptr(vertices))

		vertexCount := int32(len(vertices) / 6)
		gl.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	}
}

// buildWallQuad generates a vertex array for a wall based on its geometry, UV mapping, and lighting properties.
func (w *RenderOpenGL) buildWallQuad(cp *model.CompiledPolygon, texW float32, texH float32, zBottom float32, zTop float32) []float32 {
	u0 := float32(cp.U0) / texW
	u1 := float32(cp.U1) / texW
	v0 := float32(0.0)
	v1 := float32(zTop-zBottom) / texH
	light := float32(cp.Sector.LightDistance)

	// Inverse Transform: prendiamo i vertici clippati e ruotati in Camera Space
	// e li riportiamo nel World Space passandoli liberi alla GPU.
	sin, cos := w.vi.AngleSin, w.vi.AngleCos
	wx1 := float32((cp.Tx1 * sin) + (cp.Tz1 * cos) + w.vi.Where.X)
	wy1 := float32(-(cp.Tx1 * cos) + (cp.Tz1 * sin) + w.vi.Where.Y)
	wx2 := float32((cp.Tx2 * sin) + (cp.Tz2 * cos) + w.vi.Where.X)
	wy2 := float32(-(cp.Tx2 * cos) + (cp.Tz2 * sin) + w.vi.Where.Y)

	return []float32{
		wx1, zTop, -wy1, u0, v1, light,
		wx1, zBottom, -wy1, u0, v0, light,
		wx2, zBottom, -wy2, u1, v0, light,

		wx1, zTop, -wy1, u0, v1, light,
		wx2, zBottom, -wy2, u1, v0, light,
		wx2, zTop, -wy2, u1, v1, light,
	}
}

// buildFlatPoly generates a flat polygon vertex stream for OpenGL rendering from the provided compiled polygon.
// It calculates Z based on whether the polygon is a floor or ceiling, appends UV and light data, and returns a float32 slice.
func (w *RenderOpenGL) buildFlatPoly(cp *model.CompiledPolygon, z float64, texW, texH float32) []float32 {
	segs := cp.Sector.Segments
	if len(segs) < 3 {
		return nil
	}

	light := float32(cp.Sector.LightDistance)
	var stream []float32

	// Abbandoniamo cp.Points (che contiene trapezi con coordinate in pixel dello schermo!)
	// Triangoliamo nativamente i segmenti del settore con un Triangle Fan.
	v0 := segs[0].Start
	u0 := float32(v0.X) / texW
	v0_v := float32(-v0.Y) / texH

	for i := 1; i < len(segs)-1; i++ {
		v1 := segs[i].Start
		u1 := float32(v1.X) / texW
		v1_v := float32(-v1.Y) / texH

		v2 := segs[i+1].Start
		u2 := float32(v2.X) / texW
		v2_v := float32(-v2.Y) / texH

		stream = append(stream,
			float32(v0.X), float32(z), float32(-v0.Y), u0, v0_v, light,
			float32(v1.X), float32(z), float32(-v1.Y), u1, v1_v, light,
			float32(v2.X), float32(z), float32(-v2.Y), u2, v2_v, light,
		)
	}

	return stream
}

// doInitialize initializes the OpenGL rendering context, compiles shaders, and prepares textures. Returns an error on failure.
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

// Start begins the OpenGL rendering process by initializing and running the main execution loop.
func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

// doRun starts the main rendering loop for the application, managing frame updates, input handling, and game logic execution.
func (w *RenderOpenGL) doRun() {
	const framerate = 30
	const frameInterval = 1.0 / framerate

	if err := w.doInitialize(); err != nil {
		panic(err)
	}

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
		//w.win.Update()
		w.win.UpdateInputAndSwap()
		//text.Draw(win, g.mainMatrix)
	}
}

// doRender executes the primary rendering process, including clearing the screen, camera setup, and rendering traversal.
func (w *RenderOpenGL) doRender() {
	_, w.vi.AngleSin, w.vi.AngleCos = w.player.GetAngle()
	w.vi.Sector = w.player.GetSector()
	w.vi.Where.X, w.vi.Where.Y = w.player.GetCoords()
	w.vi.Where.Z = w.player.GetZ()
	w.vi.Yaw = w.player.GetYaw()
	w.vi.LightDistance = w.player.GetLightDistance()

	cs, count := w.portal.Compile(w.vi)
	w.targetLastCompiled = count
	batchMap := w.createBatch(cs, count)

	thErr := executor.Thread.CallErr(func() error {
		w.win.Begin()
		//gl.Enable(gl.DEPTH_TEST)
		//gl.DepthFunc(gl.LEQUAL)
		//gl.Disable(gl.BLEND)
		//gl.Disable(gl.CULL_FACE) // Tienilo disabilitato finché non confermiamo il winding dei triangoli

		gl.Viewport(0, 0, int32(w.screenWidth), int32(w.screenHeight))
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// 1. Setup Uniforms (Matrici di View e Projection dal vi)
		w.glUpdateCameraUniforms(w.vi)
		// 2. Traversal e allocazione buffer
		w.glStreamRender(batchMap)
		return nil
	})
	if thErr != nil {
		panic(thErr)
	}

	w.player.Compute(w.vi)
}

// doPlayerDuckingToggle toggles the player's ducking state by invoking the SetDucking method on the player object.
func (w *RenderOpenGL) doPlayerDuckingToggle() {
	w.player.SetDucking()
}

// doPlayerJump makes the player perform a jump action by invoking the player's SetJump method.
func (w *RenderOpenGL) doPlayerJump() {
	w.player.SetJump()
}

// doPlayerMoves processes player movement based on directional inputs (up, down, left, right) and movement speed (slow).
func (w *RenderOpenGL) doPlayerMoves(up bool, down bool, left bool, right bool, slow bool) {
	w.player.Move(up, down, left, right, slow)
}

// doPlayerMouseMove adjusts player orientation based on mouse movement values within defined limits.
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

// doZoom adjusts the zoom level of the current view by the specified amount.
func (w *RenderOpenGL) doZoom(zoom float64) {
	w.vi.Zoom += zoom
}

// doDebug toggles debug mode or navigates through debug sectors based on the `next` parameter.
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

// doDebugMoveSectorToggle toggles the debug mode for rendering target sectors by enabling or disabling it.
func (w *RenderOpenGL) doDebugMoveSectorToggle() {
	w.targetEnabled = !w.targetEnabled
}

// doDebugMoveSector updates the target sector index based on the direction and updates the targetSectors map accordingly.
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
