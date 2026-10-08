package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// IBuilder provides methods for constructing and managing rendering data, such as vertices, lights, textures, and draw commands.
type IBuilder interface {
	Compute(fbw, fbh int32, vi *model.ViewMatrix, engine *engine.Engine)

	GetSkyTexture() *textures.Texture

	GetSkyUV() (float64, float64)

	GetVerticesStride() int32

	GetLightsStride() int32

	GetDrawCommands() *DrawCommandsRender

	GetDrawCommandsAdditive() *DrawCommandsRender

	GetDrawCommandsLiquid() *DrawCommandsRender

	GetHWOcclusion() *OcclusionHW

	GetVertices() ([]float32, int32, []uint32, int32)

	GetLights() ([]float32, int32)

	GetShadowLights() ([8]*Light, int32)
}

// SpriteNode represents a node in a sprite rendering system, associating an entity with its distance-squared value.
type SpriteNode struct {
	Thing  model.IThing
	DistSq float64
}

// RenderOpenGL encapsulates rendering logic using the OpenGL API.
type RenderOpenGL struct {
	ctx             api.IContext
	engine          *engine.Engine
	vi              *model.ViewMatrix
	player          *model.ThingPlayer
	shaders         *Shaders
	tex             *Textures
	builder         IBuilder
	enableClear     bool
	builders        []IBuilder
	startWidth      int32
	startHeight     int32
	buildersCounter int
}

// NewRender initializes a new OpenGL rendering instance with the given context and dimensions.
func NewRender(ctx api.IContext, w, h int32) *RenderOpenGL {
	r := &RenderOpenGL{
		ctx:         ctx,
		engine:      nil,
		vi:          model.NewViewMatrix(),
		player:      nil,
		enableClear: false,
		shaders:     nil,
		startWidth:  w,
		startHeight: h,
	}
	return r
}

// Setup initializes the RenderOpenGL instance by linking it to the provided engine and setting up the rendering context.
func (w *RenderOpenGL) Setup(en *engine.Engine) error {
	w.engine = en
	w.player = en.GetPlayer()
	if err := w.ctx.Setup(w); err != nil {
		return err
	}
	return nil
}

// Start initializes the rendering process by invoking the context's Start method with RenderStart.
func (w *RenderOpenGL) Start() {
	w.ctx.Start()
}

// RenderPrepare initializes shaders, textures, and builder components required for rendering and configures global settings.
func (w *RenderOpenGL) RenderPrepare() error {
	cal := w.engine.GetCalibration()
	w.tex = NewTextures(w.ctx)
	w.buildersCounter = 0
	w.player.SetPitchOptions(-1.5, 1.5, 0.01)
	w.builder = NewBuilderVolume(w.ctx, w.tex, cal)
	w.builders = append(w.builders, w.builder)
	vStride := w.builder.GetVerticesStride()
	lStride := w.builder.GetLightsStride()
	dcOpaque := w.builder.GetDrawCommands()
	dcAdditive := w.builder.GetDrawCommandsAdditive()
	dcLiquid := w.builder.GetDrawCommandsLiquid()
	hwOcc := w.builder.GetHWOcclusion()
	w.shaders = NewShaders(w.ctx)
	if err := w.shaders.Setup(vStride, lStride, w.player, cal, w.tex, dcOpaque, dcAdditive, dcLiquid, hwOcc); err != nil {
		return err
	}
	if err := w.tex.Setup(w.engine.GetTextures()); err != nil {
		return err
	}

	return nil
}

// RenderStart initializes and executes the rendering process for the current frame with specified framebuffer dimensions.
func (w *RenderOpenGL) RenderStart(fbW int, fbH int, winW int, winH int) {
	w.engine.Compute(w.player, w.vi)
	w.builder.Compute(int32(fbW), int32(fbH), w.vi, w.engine)
	cSky := w.builder.GetSkyTexture()
	vert, vertLen, indices, indicesLen := w.builder.GetVertices()
	light, lightsCount := w.builder.GetLights()

	shadowLights, shadowLightsCount := w.builder.GetShadowLights()
	skyLayer := float32(-1.0)
	skyEnabled := false
	skyU, skyV := float32(0), float32(0)
	if cSky != nil {
		skyLayer, skyEnabled = w.tex.Get(cSky)
		u, v := w.builder.GetSkyUV()
		tick := float32(textures.GlobalTick()) / 60.0
		skyU = float32(u) * tick
		skyV = float32(v) * tick
		// DEBUG
		//if textures.GlobalTick()%60 == 0 {
		//	fmt.Println("SKY DEBUG: Name=", cSky.GetName(), " Layer=", skyLayer, " Enabled=", skyEnabled, " U=", skyU, " V=", skyV)
		//}
	}
	w.shaders.Render(w.vi, int32(fbW), int32(fbH), int32(winW), int32(winH), vert, vertLen, indices, indicesLen, skyEnabled, skyLayer, skyU, skyV, light, lightsCount, shadowLights, shadowLightsCount)
}

// BuilderUpdate increments the buildersCounter and updates the current builder using a round-robin approach.
//func (w *RenderOpenGL) BuilderUpdate() {0
//	w.buildersCounter++
//	index := w.buildersCounter % (len(w.builders))
//	w.builder = w.builders[index]
//}

// RenderToggleShadows toggles the shadow rendering state in the shaders, enabling or disabling shadows dynamically.
func (w *RenderOpenGL) RenderToggleShadows() {
	w.shaders.ToggleShadows()
}

// RenderEnableClear sets the enableClear flag to true, indicating that the rendering system should clear buffers.
func (w *RenderOpenGL) RenderEnableClear() {
	w.enableClear = true
}

// RenderPlayerThrow triggers the player to throw a predefined object with a set velocity and default throwable index.
func (w *RenderOpenGL) RenderPlayerThrow() {
	const throwableIndex = 2
	w.player.Throw(throwableIndex, 300)
}

// RenderPlayerFire triggers the player to fire their weapon, using the "gun" identifier as the weapon type.
func (w *RenderOpenGL) RenderPlayerFire() {
	w.player.Fire("gun")
}

// RenderPlayerDuckingToggle toggles the player's ducking state through the player's SetDucking method.
func (w *RenderOpenGL) RenderPlayerDuckingToggle() { w.player.SetDucking() }

// RenderPlayerJump triggers the player's jump action with the option for a multi-jump when the multi flag is true.
func (w *RenderOpenGL) RenderPlayerJump(multi bool) { w.player.SetJump(multi) }

// RenderPlayerMoves updates the player's movement based on impulse and directional input flags (up, down, left, right).
func (w *RenderOpenGL) RenderPlayerMoves(impulse float64, up bool, down bool, left bool, right bool) {
	w.player.Move(impulse, up, down, right, left)
}

// RenderPlayerMouseMove adjusts the player's view angle and pitch based on mouse movement values within defined thresholds.
func (w *RenderOpenGL) RenderPlayerMouseMove(mouseX float64, mouseY float64) {
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
	w.player.AddAngle(-mouseX * 0.03)
	w.player.SetPitch(mouseY)
	//w.player.MoveApply(0, 0, 0)
}

func (w *RenderOpenGL) RenderIncreaseFlashFactor() {
	w.player.GetFlash().IncreaseFlashFactor()
}
func (w *RenderOpenGL) RenderDecreaseFlashFactor() {
	w.player.GetFlash().DecreaseFlashFactor()
}
