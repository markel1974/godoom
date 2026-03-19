package open_gl

import (
	"math"
	"sort"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/textures"
	"github.com/markel1974/godoom/pixels"
	"github.com/markel1974/godoom/pixels/executor"

	"github.com/go-gl/gl/v3.3-core/gl"
)

// scaleFactor defines a constant value for scaling factors used in the application.
// maxBatchVertices specifies the maximum number of vertices that can be processed in a single batch.
// maxFrameCommands sets the limit on the number of commands that can be issued per frame.
// vboMaxFloats determines the maximum number of floats that can be stored in the vertex buffer object.
const (
	scaleFactor = 1

	maxBatchVertices = 65536 * 2

	maxFrameCommands = 4096

	vboMaxFloats = 1024 * 1024 * 4
)

// SpriteNode represents a renderable entity in a scene, including its associated model and squared distance from the camera.
type SpriteNode struct {
	Thing  model.IThing
	DistSq float64
}

// RenderOpenGL is responsible for managing and executing OpenGL rendering operations for the game environment.
type RenderOpenGL struct {
	engine       *engine.Engine
	vi           *model.ViewMatrix
	player       *model.ThingPlayer
	textures     textures.ITextures
	win          *pixels.GLWindow
	screenWidth  int
	screenHeight int

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

// NewOpenGLRender initializes and returns a new instance of RenderOpenGL with default settings and prepared resources.
func NewOpenGLRender() *RenderOpenGL {
	r := &RenderOpenGL{
		engine:        nil,
		vi:            model.NewViewMatrix(),
		textures:      nil,
		player:        nil,
		win:           nil,
		screenWidth:   0,
		screenHeight:  0,
		targetSectors: map[int]bool{0: true},
		enableClear:   false,
		debug:         false,
		debugIdx:      0,

		frameVertices: NewFrameVertices(maxBatchVertices),
		frameCommands: NewDrawCommands(maxFrameCommands),

		compiler: NewCompiler(),
	}
	return r
}

// Setup initializes the RenderOpenGL instance by configuring essential properties based on the provided engine instance.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.screenWidth = w.engine.GetWidth()
	w.screenHeight = w.engine.GetHeight()
	w.player = en.GetPlayer()
	w.textures = en.GetTextures()
	return nil
}

// glInit initializes OpenGL state, buffers, shaders, and samplers required for rendering and SSAO processing.
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

	// sky
	gl.GenVertexArrays(1, &w.skyVao)
	gl.BindVertexArray(w.skyVao)
	gl.GenBuffers(1, &w.skyVbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.skyVbo)
	skyQuadVertices := []float32{-1.0, -1.0, 1.0, -1.0, -1.0, 1.0, 1.0, 1.0}
	gl.BufferData(gl.ARRAY_BUFFER, len(skyQuadVertices)*4, gl.Ptr(skyQuadVertices), gl.STATIC_DRAW)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)

	// Setup SSAO Samplers
	progSSAO := w.compiler.GetShaderProgram(shaderSSAO)
	gl.UseProgram(progSSAO)
	gl.Uniform1i(gl.GetUniformLocation(progSSAO, gl.Str("gPosition\x00")), 0)
	gl.Uniform1i(gl.GetUniformLocation(progSSAO, gl.Str("gNormal\x00")), 1)
	gl.Uniform1i(gl.GetUniformLocation(progSSAO, gl.Str("texNoise\x00")), 2)

	// Setup Blur Sampler
	progBlur := w.compiler.GetShaderProgram(shaderBlur)
	gl.UseProgram(progBlur)
	gl.Uniform1i(gl.GetUniformLocation(progBlur, gl.Str("ssaoInput\x00")), 0)

	// Setup Main Samplers
	progMain := w.compiler.GetShaderProgram(shaderMain)
	gl.UseProgram(progMain)
	gl.Uniform1i(gl.GetUniformLocation(progMain, gl.Str("u_texture\x00")), 0)
	gl.Uniform1i(gl.GetUniformLocation(progMain, gl.Str("u_normalMap\x00")), 1)
	gl.Uniform1i(gl.GetUniformLocation(progMain, gl.Str("u_ssao\x00")), 2)

	// Configurazione Sampler Uniforms
	texLoc := gl.GetUniformLocation(progMain, gl.Str("u_texture\x00"))
	gl.Uniform1i(texLoc, 0)
	normLoc := gl.GetUniformLocation(progMain, gl.Str("u_normalMap\x00"))
	gl.Uniform1i(normLoc, 1)

	// --- SETUP FALLBACK NORMAL MAP (TEXTURE1) ---
	var defaultNormalMap uint32
	gl.GenTextures(1, &defaultNormalMap)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, defaultNormalMap)

	// Creazione pixel indaco piatto (Z-Up) per annullare perturbazioni vettoriali
	flatNormalPixel := []uint8{128, 128, 255, 255}
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, 1, 1, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(flatNormalPixel))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	// --- RIPRISTINO FLUSSO STANDARD (TEXTURE0) ---
	gl.ActiveTexture(gl.TEXTURE0)
	// I parametri qui sotto ora si applicano correttamente alla TEXTURE0 di default
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)

	// Inizializzazione buffer SSAO tramite il compiler
	fbW, fbH := w.win.GetFramebufferSize()

	w.compiler.Setup(int32(fbW), int32(fbH))

	return nil
}

// createBatch processes compiled sectors and things to create a batch of renderable geometry with optional sky texture.
func (w *RenderOpenGL) createBatch(css []*model.CompiledSector, compiled int, thing []model.IThing) *textures.Texture {
	w.frameVertices.Reset()
	w.frameCommands.Reset()
	var cSky *textures.Texture = nil

	for idx := compiled - 1; idx >= 0; idx-- {
		current := css[idx]

		polygons := current.Get()
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
				if sky := w.pushFlat(cp, cp.AnimationCeil, float32(cp.Sector.CeilY)); sky != nil {
					cSky = sky
				}
			case model.IdFloor, model.IdFloorTest:
				if sky := w.pushFlat(cp, cp.AnimationFloor, float32(cp.Sector.FloorY)); sky != nil {
					cSky = sky
				}
			}
		}
	}

	//TODO
	w.pushThings(thing)
	return cSky
}

// pushWall generates and adds wall vertices to the frame buffer with appropriate texture mapping and lighting calculations.
func (w *RenderOpenGL) pushWall(cp *model.CompiledPolygon, anim *textures.Animation, zBottom, zTop float32) {
	//prepare
	tex := anim.CurrentFrame()
	if tex == nil {
		return
	}
	texId, normTexId, ok := w.compiler.GetTexture(tex)
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

	sin, cos := w.vi.GetAngle()
	viX, vizY := w.vi.GetXY()
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

	light, lcX, lcY, lcZ := w.createLight(cp.Sector.Light)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx1, zBottom, -wy1, u0, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)

	w.frameVertices.AddVertex(wx1, zTop, -wy1, u0, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zBottom, -wy2, u1, vBottom, light, lcX, lcY, lcZ, nX, nY, nZ)
	w.frameVertices.AddVertex(wx2, zTop, -wy2, u1, vTop, light, lcX, lcY, lcZ, nX, nY, nZ)

	//apply
	currentLen := w.frameVertices.Len()
	w.frameCommands.Compute(texId, normTexId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())
}

// pushFlat processes a flat surface polygon, computes its vertices for rendering, and generates associated render commands.
func (w *RenderOpenGL) pushFlat(cp *model.CompiledPolygon, anim *textures.Animation, zF float32) *textures.Texture {
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
	texId, normTexId, ok := w.compiler.GetTexture(tex)
	if !ok {
		return nil
	}
	texW, texH := tex.Size()
	startLen := w.frameVertices.Len()
	_, scaleH := anim.ScaleFactor()
	v0 := segments[0].Start

	u0 := (float32(v0.X) / float32(texW)) * float32(scaleH)
	v0V := (float32(-v0.Y) / float32(texH)) * float32(scaleH)

	nY := float32(1.0)
	if cp.Kind == model.IdCeil || cp.Kind == model.IdCeilTest {
		nY = -1.0
	}

	light, lcX, lcY, lcZ := w.createLight(cp.Sector.Light)

	for i := 1; i < len(segments)-1; i++ {
		v1, v2 := segments[i].Start, segments[i+1].Start

		u1 := (float32(v1.X) / float32(texW)) * float32(scaleH)
		v1V := (float32(-v1.Y) / float32(texH)) * float32(scaleH)
		u2 := (float32(v2.X) / float32(texW)) * float32(scaleH)
		v2V := (float32(-v2.Y) / float32(texH)) * float32(scaleH)

		w.frameVertices.AddVertex(float32(v0.X), zF, float32(-v0.Y), u0, v0V, light, lcX, lcY, lcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v1.X), zF, float32(-v1.Y), u1, v1V, light, lcX, lcY, lcZ, 0, nY, 0)
		w.frameVertices.AddVertex(float32(v2.X), zF, float32(-v2.Y), u2, v2V, light, lcX, lcY, lcZ, 0, nY, 0)
	}

	//apply
	currentLen := w.frameVertices.Len()
	w.frameCommands.Compute(texId, normTexId, int32(startLen), int32(currentLen), w.frameVertices.Alignment())

	return nil
}

// pushThings processes a list of Thing objects for rendering, handling culling, depth sorting, and batching into the VBO.
func (w *RenderOpenGL) pushThings(things []model.IThing) {
	if len(things) == 0 {
		return
	}

	camX, camY := w.vi.GetXY()
	queue := make([]SpriteNode, 0, len(things))

	// 1. Culling e calcolo distanza quadrica
	for _, t := range things {
		if t.GetAnimation() == nil {
			continue
		}
		tPosX, tPosY := t.GetPosition()
		dx := tPosX - camX
		dy := tPosY - camY

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
		tex := t.GetAnimation().CurrentFrame()
		if tex == nil {
			continue
		}
		texId, normTexId, ok := w.compiler.GetTexture(tex)
		if !ok {
			continue
		}

		texW, texH := tex.Size()
		scaleW, scaleH := t.GetAnimation().ScaleFactor()
		width := float64(texW) * scaleW
		height := float64(texH) * scaleH

		dist := math.Sqrt(node.DistSq)
		if dist < 0.0001 {
			dist = 0.0001
		}

		// Vettore Right normalizzato e scalato per l'estensione del quad
		halfW := width / 2.0
		tPosX, tPosY := t.GetPosition()
		rX := -((camY - tPosY) / dist) * halfW
		rY := ((camX - tPosX) / dist) * halfW

		// Coordinate planari dei due spigoli
		v1x := float32(tPosX - rX)
		v1y := float32(tPosY - rY)
		v2x := float32(tPosX + rX)
		v2y := float32(tPosY + rY)

		// Quota verticale
		zBottom := float32(t.GetFloorY())
		zTop := zBottom + float32(height)

		// --- LUCE IDENTICA A PUSH WALL ---
		light, vLcX, vLcY, vLcZ := w.createLight(t.GetLight())

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

		w.frameCommands.Compute(texId, normTexId, startLen, int32(fv.Len()), fv.Alignment())
	}
}

// glStreamRender performs the full OpenGL rendering pipeline for the scene, including geometry, SSAO, blur, and lighting passes.
func (w *RenderOpenGL) glStreamRender() {
	if w.frameVertices.Len() == 0 {
		return
	}

	// 0. UPLOAD VERTICI (Orphaning)
	gl.BindBuffer(gl.ARRAY_BUFFER, w.mainVbo)
	gl.BufferData(gl.ARRAY_BUFFER, w.frameVertices.Len()*4, nil, gl.STREAM_DRAW)
	gl.BufferSubData(gl.ARRAY_BUFFER, 0, w.frameVertices.Len()*4, gl.Ptr(w.frameVertices.Get()))

	// Calcolo e recupero matrici (w.glUpdateCameraUniforms inietta già su shaderMain)
	proj, view := w.glUpdateCameraUniforms(w.vi)

	// 1. GEOMETRY PASS (G-Buffer)
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.compiler.gBufferFbo)
	// Sfondo lontanissimo per evitare che il cielo occluda la geometria
	gl.ClearColor(0.0, 0.0, -100000.0, 1.0)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.ClearColor(0.0, 0.0, 0.0, 1.0) // Ripristina per eventuali pass successivi

	programGeometry := w.compiler.GetShaderProgram(shaderGeometry)
	gl.UseProgram(programGeometry)
	gl.Uniform1i(gl.GetUniformLocation(programGeometry, gl.Str("u_texture\x00")), 0)
	gl.UniformMatrix4fv(gl.GetUniformLocation(programGeometry, gl.Str("u_view\x00")), 1, false, &view[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(programGeometry, gl.Str("u_projection\x00")), 1, false, &proj[0])
	w.renderScene(programGeometry)

	// 2. SSAO PASS
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.compiler.ssaoFbo)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	programSSAO := w.compiler.GetShaderProgram(shaderSSAO)
	gl.UseProgram(programSSAO)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.compiler.gPositionDepth)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, w.compiler.gNormal)
	gl.ActiveTexture(gl.TEXTURE2)
	noiseTex, kernel := w.compiler.GetSSAOResources()
	gl.BindTexture(gl.TEXTURE_2D, noiseTex)

	gl.Uniform3fv(gl.GetUniformLocation(shaderSSAO, gl.Str("samples\x00")), 64, &kernel[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(shaderSSAO, gl.Str("projection\x00")), 1, false, &proj[0])
	w.drawScreenQuad()

	// 3. BLUR PASS
	gl.BindFramebuffer(gl.FRAMEBUFFER, w.compiler.ssaoBlurFbo)
	gl.Clear(gl.COLOR_BUFFER_BIT)

	programBlur := w.compiler.GetShaderProgram(shaderBlur)
	gl.UseProgram(programBlur)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, w.compiler.ssaoColorBuffer)
	w.drawScreenQuad()

	// 4. FINAL LIGHTING PASS
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	programMain := w.compiler.GetShaderProgram(shaderMain)
	gl.UseProgram(programMain)

	ssaoBlurTex := w.compiler.GetSSAOBlurTexture()
	gl.BindVertexArray(w.mainVao)

	for _, cmd := range w.frameCommands.Get() {
		if cmd.vertexCount > 0 {
			gl.ActiveTexture(gl.TEXTURE0)
			gl.BindTexture(gl.TEXTURE_2D, cmd.texId)

			gl.ActiveTexture(gl.TEXTURE1)
			gl.BindTexture(gl.TEXTURE_2D, cmd.normTexId)

			gl.ActiveTexture(gl.TEXTURE2)
			gl.BindTexture(gl.TEXTURE_2D, ssaoBlurTex)

			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}

// renderScene renders the current scene by iterating over draw commands and issuing OpenGL draw calls.
func (w *RenderOpenGL) renderScene(program uint32) {
	gl.BindVertexArray(w.mainVao)
	for _, cmd := range w.frameCommands.Get() {
		if cmd.vertexCount > 0 {
			// Vincolo alla TEXTURE0 richiesto per l'alpha discard nel geometry.frag
			gl.ActiveTexture(gl.TEXTURE0)
			gl.BindTexture(gl.TEXTURE_2D, cmd.texId)
			gl.DrawArrays(gl.TRIANGLES, cmd.firstVertex, cmd.vertexCount)
		}
	}
}

// drawScreenQuad renders a full-screen quad using the sky vertex array and disables depth testing during the draw operation.
func (w *RenderOpenGL) drawScreenQuad() {
	gl.BindVertexArray(w.skyVao)
	gl.Disable(gl.DEPTH_TEST)
	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)
	gl.Enable(gl.DEPTH_TEST)
}

// glUpdateCameraUniforms updates the camera view and projection matrices, along with related shader uniforms.
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
	fbW, fbH := w.win.GetFramebufferSize()
	gl.Uniform2f(gl.GetUniformLocation(shaderProgram, gl.Str("u_screenResolution\x00")), float32(fbW), float32(fbH))

	// --- NUOVI UNIFORM PER LA LUCE/TORCIA ---
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("u_cameraPos\x00")), ex, ey, ez)
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("u_cameraFront\x00")), fX, 0.0, fZ)
	// Calcolo della direzione della torcia compensando l'Y-Shear
	flashDirY := pitchShear / scaleY
	gl.Uniform3f(gl.GetUniformLocation(shaderProgram, gl.Str("u_flashDir\x00")), 0.0, flashDirY, -1.0)
	return proj, view
}

// glRenderSky renders the skybox using the provided projection and view matrices, and binds the given sky texture.
func (w *RenderOpenGL) glRenderSky(proj [16]float32, view [16]float32, cSky *textures.Texture) {
	skyProg := w.compiler.GetShaderProgram(shaderSky)
	gl.UseProgram(skyProg)

	// Z_Quad == Z_Clear: viene eseguito SOLO dove non c'è geometria solida!
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false) // Read-only

	gl.UniformMatrix4fv(gl.GetUniformLocation(skyProg, gl.Str("u_projection\x00")), 1, false, &proj[0])
	gl.UniformMatrix4fv(gl.GetUniformLocation(skyProg, gl.Str("u_view\x00")), 1, false, &view[0])

	gl.BindVertexArray(w.skyVao)

	if texId, normTextId, ok := w.compiler.GetTexture(cSky); ok {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texId)
		gl.ActiveTexture(gl.TEXTURE1)

		gl.BindTexture(gl.TEXTURE_2D, normTextId)
		gl.ActiveTexture(gl.TEXTURE2)

		gl.Uniform1i(gl.GetUniformLocation(skyProg, gl.Str("u_sky\x00")), 0)
	}

	gl.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	// Ripristina lo stato standard per il frame successivo
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

// doInitialize initializes the OpenGL rendering environment and compiles shaders and textures for the renderer.
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

// Start initializes and starts the OpenGL rendering loop by invoking the provided rendering function.
func (w *RenderOpenGL) Start() {
	pixels.GLRun(w.doRun)
}

// doRun executes the main rendering and input handling loop for the RenderOpenGL instance.
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

		if w.win.JustPressed(pixels.KeyF) {
			w.doPlayerFire()
		}
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

// doRender performs the rendering process by computing the scene, creating rendering batches, and issuing draw commands.
func (w *RenderOpenGL) doRender() {
	cs, count, things := w.engine.Compute(w.player, w.vi)
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

// doPlayerFire triggers the player's fire action by retrieving position, angle, and sector, and invoking the engine's fire logic.
func (w *RenderOpenGL) doPlayerFire() {
	x, y := w.player.GetPosition()
	w.engine.Fire(w.player.GetSector(), x, y, w.player.GetRealAngle())
}

// doPlayerDuckingToggle toggles the player's ducking state by invoking the SetDucking method on the player instance.
func (w *RenderOpenGL) doPlayerDuckingToggle() { w.player.SetDucking() }

// doPlayerJump triggers the player's jump action by invoking the SetJump method on the player instance.
func (w *RenderOpenGL) doPlayerJump() { w.player.SetJump() }

// doPlayerMoves moves the player based on the provided impulse and directional flags (up, down, left, right).
func (w *RenderOpenGL) doPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, left, right)
}

// doPlayerMouseMove adjusts the player's angle and yaw based on mouse movement, clamping the values within a defined offset range.
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

// doDebug toggles debugging or sets the debug mode to a specific sector depending on the provided next parameter.
func (w *RenderOpenGL) doDebug(next int) {
	if next == 0 {
		w.debug = !w.debug
		return
	}
	w.debug = true
	idx := w.debugIdx + next
	if idx < 0 || idx >= w.engine.Len() {
		return
	}
	w.debugIdx = idx
	sector := w.engine.SectorAt(idx)
	const offset = 5
	x := sector.Segments[0].Start.X + offset
	y := sector.Segments[0].Start.Y + offset
	w.player.SetSector(sector)
	w.player.SetXY(x, y)
}

// doDebugMoveSectorToggle toggles the `targetEnabled` state, enabling or disabling the debug move sector functionality.
func (w *RenderOpenGL) doDebugMoveSectorToggle() { w.targetEnabled = !w.targetEnabled }

// doDebugMoveSector adjusts the target sector index and updates the target sectors map based on the direction provided.
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

// createLight calculates and returns the light intensity and transformed light position in the OpenGL rendering context.
func (w *RenderOpenGL) createLight(mLight *model.Light) (float32, float32, float32, float32) {
	light := (1.0 - mLight.GetIntensity()) * 5.0
	lightPos := mLight.GetPos()
	_, _, liX, liZ := w.vi.TranslateXY(lightPos.X, lightPos.Y)
	liY := lightPos.Z - w.vi.GetZ()
	return float32(light), float32(liX), float32(liY), float32(-liZ)
}
