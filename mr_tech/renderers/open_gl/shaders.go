package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/shaders"
)

// IShader defines an interface for shader operations, including setup, sampler configuration, and compilation logic.
type IShader interface {
	Init() error
	SetupSamplers() error
	Compile(a shaders.IAssets) error
}

// EnableAdditiveLights configures OpenGL to use additive blending for rendering by adjusting depth and blend settings.
func EnableAdditiveLights(ctx api.IContext) {
	ctx.DepthMask(false)
	ctx.Enable(api.BLEND)
	ctx.BlendFunc(api.ONE, api.ONE)
	ctx.DepthFunc(api.LEQUAL)
}

// DisableAdditiveLights disables blend-based additive light rendering and reconfigures the depth test to default behavior.
func DisableAdditiveLights(ctx api.IContext) {
	ctx.Disable(api.BLEND)
	ctx.DepthMask(true)
	ctx.DepthFunc(api.LESS)
}
