package open_gl

import (
	"math"
	"sort"

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
*/

// scaleFactor defines the multiplier for scaling operations.
// maxBatchVertices specifies the maximum number of vertices for a batch operation.
// maxFrameCommands indicates the maximum number of commands processed per frame.
// vboMaxFloats sets the maximum number of float values allowed in the vertex buffer object.
const (
	scaleFactor = 1

	maxBatchVertices = 65536 * 2

	maxFrameCommands = 4096

	vboMaxFloats = 1024 * 1024 * 4
)

type SpriteNode struct {
	Thing  *model.Thing
	DistSq float64
}

// RenderOpenGL represents the main rendering engine using OpenGL for handling scene rendering and custom debug features.
type RenderOpenGL struct {
	portal           *portal.Portal
	vi               *model.ViewMatrix
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

	mainVao uint32
	mainVbo uint32

	skyVao uint32
	skyVbo uint32

	enableClear bool
	debug       bool
	debugIdx    int

	frameVertices *FrameVertices
	frameCommands *DrawCommands

	compiler *Compiler
}

// NewOpenGLRender initializes and returns a new RenderOpenGL instance with default configurations and settings.
func NewOpenGLRender() *RenderOpenGL {
	r := &RenderOpenGL{
		portal:           nil,
		vi:               model.NewViewMatrix(),
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

		frameVertices: NewFrameVertices(maxBatchVertices),
		frameCommands: NewDrawCommands(maxFrameCommands),

		compiler: NewCompiler(),
	}
	return r
}

// Setup initializes the RenderOpenGL instance with the provided portal, player, and textures. Returns an error if any setup fails.
func (w *RenderOpenGL) Setup(portal *portal.Portal, player *model.Player, t textures.ITextures) error {
	w.portal = portal
	w.screenWidth = portal.ScreenWidth()
	w.screenHeight = portal.ScreenHeight()
	w.sectorsMaxHeight = portal.SectorsMaxHeight()
	w.player = player
	w.textures = t
	return nil
}

// createBatch processes and batches compiled sector polygons for rendering based on their type and attributes.
func (w *RenderOpenGL) createBatch(css []*model.CompiledSector, compiled int, thing []*model.Thing) *textures.Texture {
	w.frameVertices.Reset()
	w.frameCommands.Reset()
	var cSky *textures.Texture = nil

	for idx := compiled - 1; idx >= 0; idx-- {
		polygons := css[idx].Get()
		for k := len(polygons) - 1; k >= 0; k-- {
			cp := polygons[k]

			switch cp.Kind {
			case model.IdWall:
				w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Sector.CeilY))
			case model.IdUpper:
				w.pushWall(cp, cp.Animation, float32(cp.Neighbor.CeilY), float32(cp.Sector.CeilY))
			case model.IdLower:
				w.pushWall(cp, cp.Animation, float32(cp.Sector.FloorY), float32(cp.Neighbor.FloorY))
			case model.IdCeil, model.IdCeilTest:
				if sky := w.pushFlat(cp, cp.AnimationCeil, cp.Sector.CeilY); sky != nil {
					cSky = sky
				}
			case model.IdFloor, model.IdFloorTest:
				if sky := w.pushFlat(cp, cp.AnimationFloor, cp.Sector.FloorY); sky != nil {
					cSky = sky
				}
			}
		}
	}

	//TODO
	w.pushThings(thing)
	return cSky
}

// pushWall appends a textured wall's vertices and lighting properties into the frame for rendering using OpenGL.
func (w *RenderOpenGL) pushWall(cp *model.CompiledPolygon, anim *textures.Animation, zBottom, zTop float32) {
	//prepare
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	texId, ok := w.compiler.GetTexture(tex)
	if !ok {
		return
	}
	texW, texH := tex.Size()
	startLen := w.frameVertices.Len()
	scaleW, scaleH := anim.ScaleFactor()

	u0 := float32(cp.U0) / (float32(texW) * float32(scaleW))
	u1 := float32(cp.U1) / (float32(texW) * float32(scaleW))

	vTop := float32(0.0)
	vBottom := ((zTop - zBottom) / float32(texH)) * float32(scaleH)
	light := float32((1.0 - cp.Sector.Light.GetIntensity()) * 5.0)
	lightPos := cp.Sector.Light.GetPos()

	_, _, lcX, lcZ := w.vi.TranslateXY(lightPos.X, lightPos.Y)
	viZ := w.vi.GetZ()
	viX, vizY := w.vi.GetXY()
	lcY := lightPos.Z - viZ

	sin, cos := w.vi.GetAngle()
	wx1 := float32((cp.Tx1 * sin) + (cp.Tz1 * cos) + viX)
	wy1 := float32(-(cp.Tx1 * cos) + (cp.Tz1 * sin) + vizY)
	wx2 := float32((cp.Tx2 * sin) + (cp.Tz2 * cos) + viX)
	wy2 := float32(-(cp.Tx2 * cos) + (cp.Tz2 * sin) + vizY)

	dx := float64(wx2 - wx1)
	dz := float64((-wy2) - (-wy1))
	length := math.Hypot(dx, dz)

	nX := float32(-dz / length)
	nY := float32(0.0)
	nZ := float32(dx / length)

	vLcX := float32(lcX)
	vLcY := float32(lcY)
	vLcZ := float32(-lcZ)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zTop, -wy2, u1, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

	//apply
	currentLen := w.frameVertices.Len()
	w.frameCommands.Compute(texId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())
}

// pushFlat renders a flat surface using vertices from a compiled polygon and texture, applying lighting and transformations.
func (w *RenderOpenGL) pushFlat(cp *model.CompiledPolygon, anim *textures.Animation, z float64) *textures.Texture {
	if anim.Kind() == int(model.AnimationKindSky) {
		return anim.CurrentFrame()
	}

	tex := anim.CurrentFrame()
	if tex == nil {
		return nil
	}
	segments := cp.Sector.Segments
	if len(segments) < 3 {
		return nil
	}
	//prepare
	texId, ok := w.compiler.GetTexture(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	startLen := w.frameVertices.Len()

	// 1. Allinea il fattore di scala anche per pavimenti e soffitti
	//scale := float32(cp.Sector.Animations.ScaleFactor())
	//if scale <= 0 {
	//	scale = 1.0
	//}
	_, scaleH := anim.ScaleFactor()

	lightPos := cp.Sector.Light.GetPos()
	light := float32((1.0 - cp.Sector.Light.GetIntensity()) * 5.0)
	_, _, lcX, lcZ := w.vi.TranslateXY(lightPos.X, lightPos.Y)
	lcY := lightPos.Z - w.vi.GetZ()

	vLcX := float32(lcX)
	vLcY := float32(lcY)
	vLcZ := float32(-lcZ)

	zF := float32(z)
	v0 := segments[0].Start

	u0 := (float32(v0.X) / float32(texW)) * float32(scaleH)
	v0V := (float32(-v0.Y) / float32(texH)) * float32(scaleH)

	nY := float32(1.0)
	if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
		nY = -1.0
	}

	for i := 1; i < len(segments)-1; i++ {
		v1, v2 := segments[i].Start, segments[i+1].Start

		u1 := (float32(v1.X) / float32(texW)) * float32(scaleH)
		v1V := (float32(-v1.Y) / float32(texH)) * float32(scaleH)
		u2 := (float32(v2.X) / float32(texW)) * float32(scaleH)
		v2V := (float32(-v2.Y) / float32(texH)) * float32(scaleH)

		w.frameVertices.AddVertex(float32(v0.X), zF, float32(-v0.Y), u0, v0V, light, vLcX, vLcY, vLcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v1.X), zF, float32(-v1.Y), u1, v1V, light, vLcX, vLcY, vLcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v2.X), zF, float32(-v2.Y), u2, v2V, light, vLcX, vLcY, vLcZ, 0, nY, 0)
	}

	//apply
	currentLen := w.frameVertices.Len()
	w.frameCommands.Compute(texId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())

	return nil
}

// pushThings processes a list of things, applies culling, depth sorting, and billboarding, then batches them for rendering.
func (w *RenderOpenGL) pushThings(things []*model.Thing) {
	if len(things) == 0 {
		return
	}

	camX, camY := w.vi.GetXY()
	queue := make([]SpriteNode, 0, len(things))

	// 1. Culling e calcolo distanza quadrica
	for _, t := range things {
		if t.Animation == nil {
			continue
		}

		dx := t.Position.X - camX
		dy := t.Position.Y - camY

		queue = append(queue, SpriteNode{
			Thing:  t,
			DistSq: dx*dx + dy*dy,
		})
	}

	// 2. Depth Sort (Painter's Algorithm)
	sort.Slice(queue, func(i, j int) bool {
		return queue[i].DistSq > queue[j].DistSq
	})

	fv := w.frameVertices

	// 3. Billboarding Cilindrico e Batching VBO
	for _, node := range queue {
		t := node.Thing
		tex := t.Animation.CurrentFrame()
		if tex == nil {
			continue
		}
		texId, ok := w.compiler.GetTexture(tex)
		if !ok {
			continue
		}

		texW, texH := tex.Size()
		scaleW, scaleH := t.Animation.ScaleFactor()
		width := (float64(texW) * scaleW) / 70
		height := (float64(texH) * scaleH) / 70

		dist := math.Sqrt(node.DistSq)
		if dist < 0.0001 {
			dist = 0.0001
		}

		// Vettore Right normalizzato e scalato per l'estensione del quad
		halfW := width / 2.0
		rX := -((camY - t.Position.Y) / dist) * halfW
		rY := ((camX - t.Position.X) / dist) * halfW

		// Coordinate planari dei due spigoli
		v1x := float32(t.Position.X - rX)
		v1y := float32(t.Position.Y - rY)
		v2x := float32(t.Position.X + rX)
		v2y := float32(t.Position.Y + rY)

		// Quota verticale
		zBottom := float32(t.Sector.FloorY)
		zTop := zBottom + float32(height)

		// --- LUCE IDENTICA A PUSH WALL ---
		light := float32((1.0 - t.Sector.Light.GetIntensity()) * 5.0)
		lightPos := t.Sector.Light.GetPos()
		_, _, lcX, lcZ := w.vi.TranslateXY(lightPos.X, lightPos.Y)
		viZ := w.vi.GetZ()
		lcY := lightPos.Z - viZ

		vLcX := float32(lcX)
		vLcY := float32(lcY)
		vLcZ := float32(-lcZ)

		// --- CALCOLO NORMALE IDENTICO A PUSH WALL ---
		dxNorm := float64(v2x - v1x)
		dzNorm := float64((-v2y) - (-v1y))
		length := math.Hypot(dxNorm, dzNorm)

		nX := float32(-dzNorm / length)
		nY := float32(0.0)
		nZ := float32(dxNorm / length)

		startLen := int32(fv.Len())

		// --- BATCHING NEL VBO ---
		// A differenza di un muro che ripete la texture, lo sprite mappa l'intera texture (UV 0.0 -> 1.0)
		u0, u1 := float32(0.0), float32(1.0)
		vTop, vBottom := float32(0.0), float32(1.0)

		// Triangolo 1 (Top-Left -> Bottom-Left -> Bottom-Right)
		w.frameVertices.AddVertex(v1x, zTop, -v1y, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v1x, zBottom, -v1y, u0, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zBottom, -v2y, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

		// Triangolo 2 (Top-Left -> Bottom-Right -> Top-Right)
		w.frameVertices.AddVertex(v1x, zTop, -v1y, u0, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zBottom, -v2y, u1, vBottom, light, vLcX, vLcY, vLcZ, nX, nY, nZ)
		w.frameVertices.AddVertex(v2x, zTop, -v2y, u1, vTop, light, vLcX, vLcY, vLcZ, nX, nY, nZ)

		w.frameCommands.Compute(texId, startLen, int32(fv.Len()), fv.Alignment())
	}
}

// glStreamRender streams vertex and command data to the GPU and executes rendering of frame vertices and textures.
func (w *RenderOpenGL) glStreamRender() {
	if w.frameVertices.Len() == 0 {
		return
	}

	gl.BindVertexArray(w.mainVao)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.mainVbo)

	// Aggiornamento parziale in-place, zero allocazioni
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, w.frameVertices.Len()*4, gl.Ptr(w.frameVertices.Get()))

	gl.ActiveTexture(gl.TEXTURE0)

	for _, cmd := range w.frameCommands.Get() {
		if cmd.vertexCount > 0 {
			gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}

// glInit initializes OpenGL resources, including VAOs, VBOs, shaders, and vertex attributes for rendering.
func (w *RenderOpenGL) glInit() error {
	stride := w.frameVertices.Alignment() * 4
	gl.GenVertexArrays(1, &w.mainVao)
	gl.BindVertexArray(w.mainVao)
	gl.GenBuffers(1, &w.mainVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, vboMaxFloats*4, nil, gl.DYNAMIC_DRAW)
	// Location 0: aPos (vec3)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Location 1: aTexCoords (vec2)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(3*4))
	gl.EnableVertexAttribArray(1)
	// Location 2: aLightIntensity (float)
	gl.VertexAttribPointer(2, 1, gl.FLOAT, false, stride, gl.PtrOffset(5*4))
	gl.EnableVertexAttribArray(2)
	// Location 3: aLightCenterView (vec3)
	gl.VertexAttribPointer(3, 3, gl.FLOAT, false, stride, gl.PtrOffset(6*4))
	gl.EnableVertexAttribArray(3)
	// Location 4: aNormal (vec3)
	gl.VertexAttribPointer(4, 3, gl.FLOAT, false, stride, gl.PtrOffset(9*4))
	gl.EnableVertexAttribArray(4)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	//sky
	gl.GenVertexArrays(1, &w.skyVao)
	gl.BindVertexArray(w.skyVao)
	gl.GenBuffers(1, &w.skyVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.skyVbo)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	gl.BufferData(gl.ARRAY_BUFFER, len(skyQuadVertices)*4, gl.Ptr(skyQuadVertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	// Restore default state
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	shaderProgram := w.compiler.GetShaderProgram(shaderMain)
	gl.UseProgram(shaderProgram)
	texLoc := gl.GetUniformLocation(shaderProgram, gl.Str("u_texture\x00"))
	gl.Uniform1i(texLoc, 0)
	// Binding sampler Normal Map
	normLoc := gl.GetUniformLocation(shaderProgram, gl.Str("u_normalMap\x00"))
	gl.Uniform1i(normLoc, 1)
	return nil
}

// glUpdateCameraUniforms updates the camera's projection and view uniform matrices in the shader and returns them.
func (w *RenderOpenGL) glUpdateCameraUniforms(vi *model.ViewMatrix) ([16]float32, [16]float32) {
	shaderProgram := w.compiler.GetShaderProgram(shaderMain)
	gl.UseProgram(shaderProgram)
	aspect := float32(w.screenWidth) / float32(w.screenHeight)
	near, far := float32(1.0), float32(100000.0)
	scaleX := float32(-(2.0 / float64(aspect)) * model.HFov)
	scaleY := float32(2.0 * model.VFov)
	pitchShear := float32(-vi.GetYaw())
	proj := [16]float32{
		scaleX, 0, 0, 0,
		0, scaleY, 0, 0,
		0, pitchShear, (far + near) / (near - far), -1,
		0, 0, (2 * far * near) / (near - far), 0,
	}

	sinA, cosA := vi.GetAngle()

	fX, fZ := float32(cosA), float32(-sinA)
	rX, rZ := float32(sinA), float32(cosA)

	viX, viY := vi.GetXY()
	viZ := vi.GetZ()

	ex := float32(viX)
	ey := float32(viZ)
	ez := float32(-viY)

	tx := -(rX*ex + rZ*ez)
	ty := -ey
	tz := fX*ex + fZ*ez

	view := [16]float32{
		rX, 0, -fX, 0,
		0, 1, 0, 0,
		rZ, 0, -fZ, 0,
		tx, ty, tz, 1,
	}

	gl.UniformMatrix4fv(gl.GetUniformLocation(shaderProgram, gl.Str("u_view\x00")), 1, false, &view[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(shaderProgram, gl.Str("u_projection\x00")), 1, false, &proj[0])
	gl.Uniform1f(gl.GetUniformLocation(shaderProgram, gl.Str("u_ambient_light\x00")), float32(vi.GetLightIntensity()))
	timeLoc := gl.GetUniformLocation(shaderProgram, gl.Str("u_time\x00"))
	gl.Uniform1f(timeLoc, float32(pixels.GLGetTime()))

	// --- NUOVI UNIFORM PER LA LUCE/TORCIA ---
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("u_cameraPos\x00")), ex, ey, ez)
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("u_cameraFront\x00")), fX, 0.0, fZ)

	return proj, view

}

// glRenderSky renders the skybox using the provided projection and view matrices.
func (w *RenderOpenGL) glRenderSky(proj [16]float32, view [16]float32, cSky *textures.Texture) {
	skyProg := w.compiler.GetShaderProgram(shaderSky)
	gl.UseProgram(skyProg)

	// Z_Quad == Z_Clear: viene eseguito SOLO dove non c'è geometria solida!
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false) // Read-only

	gl.UniformMatrix4fv(gl.GetUniformLocation(skyProg, gl.Str("u_projection\x00")), 1, false, &proj[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(skyProg, gl.Str("u_view\x00")), 1, false, &view[0])

	gl.BindVertexArray(w.skyVao)

	if texId, ok := w.compiler.GetTexture(cSky); ok {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texId)
		gl.Uniform1i(gl.GetUniformLocation(skyProg, gl.Str("u_sky\x00")), 0)
	}

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	// Ripristina lo stato standard per il frame successivo
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

// doInitialize initializes the OpenGL rendering environment, window, and related resources. Returns an error if the setup fails.
func (w *RenderOpenGL) doInitialize() error {
	cfg := pixels.WindowConfig{
		Bounds:      pixels.R(0, 0, float64(w.screenWidth)*scaleFactor, float64(w.screenHeight)*scaleFactor),
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
		if err := w.compiler.Compile(w.textures); err != nil {
			return err
		}
		return nil
	})

	if thErr != nil {
		return thErr
	}
	return nil
}

// Start initializes and runs the primary rendering loop using the OpenGL context prepared by pixels.GLRun.
func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

// doRun manages the main rendering loop, player input handling, and game state updates for the OpenGL renderer.
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

		var up, down, left, right bool

		if scroll := w.win.MouseScroll(); scroll.Y != 0 {
			if scroll.Y > 0 {
				up = true
			} else {
				down = true
			}
		}

		var impulse = 0.06
		for v := range w.win.KeysPressed() {
			switch v {
			case pixels.KeyEscape:
				return
			case pixels.KeyW:
				up = true
				impulse = 0.01
			case pixels.KeyUp:
				up = true
			case pixels.KeyS:
				down = true
				impulse = 0.01
			case pixels.KeyDown:
				down = true
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

// doRender executes the rendering process, updating the framebuffer and rendering the scene using OpenGL commands.
func (w *RenderOpenGL) doRender() {
	cs, count, things := w.portal.Compile(w.player, w.vi)
	w.targetLastCompiled = count
	cSky := w.createBatch(cs, count, things)

	executor.Thread.Call(func() {
		w.win.Begin()
		fbW, fbH := w.win.GetFramebufferSize()
		gl.Viewport(0, 0, int32(fbW), int32(fbH))
		gl.ClearColor(0.0, 0.0, 0.0, 1.0)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		proj, view := w.glUpdateCameraUniforms(w.vi)
		w.glStreamRender()
		if cSky != nil {
			w.glRenderSky(proj, view, cSky)
		}
	})
}

// doPlayerDuckingToggle toggles the ducking state of the player by invoking the SetDucking method on the player instance.
func (w *RenderOpenGL) doPlayerDuckingToggle() { w.player.SetDucking() }

// doPlayerJump triggers the player jump action by setting the appropriate state in the player object.
func (w *RenderOpenGL) doPlayerJump() { w.player.SetJump() }

// doPlayerMoves processes the player's movement based on the given impulse and directional inputs.
func (w *RenderOpenGL) doPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, left, right)
}

// doPlayerMouseMove updates the player's viewing angle and yaw based on mouse movement, with clamped input values.
func (w *RenderOpenGL) doPlayerMouseMove(mouseX float64, mouseY float64) {
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
	w.player.MoveApply(0, 0)
}

// doDebug toggles debug mode or enables it and navigates through sectors based on the provided index delta.
func (w *RenderOpenGL) doDebug(next int) {
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= w.portal.Len() {
		return
	}
	w.debugIdx = idx
	sector := w.portal.SectorAt(idx)
	const offset = 5
	x := sector.Segments[0].Start.X + offset
	y := sector.Segments[0].Start.Y + offset
	w.player.SetSector(sector)
	w.player.SetXY(x, y)
}

// doDebugMoveSectorToggle toggles the debug mode for moving between sectors by flipping the targetEnabled flag.
func (w *RenderOpenGL) doDebugMoveSectorToggle() { w.targetEnabled = !w.targetEnabled }

// doDebugMoveSector adjusts the currently selected sector for debugging purposes, moving forward or backward as specified.
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
