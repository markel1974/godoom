package draws

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const (
	CullStateUnset = -1
	CullStateOff   = 0
	CullStateBack  = 1
	CullStateFront = 2
)

// ApplyMaterialCullState updates GL CULL_FACE and CullFace only when the desired state changes.
// Returns the updated current state token.
func ApplyMaterialCullState(ctx api.IContext, curState int, cullMode int, hasCutout bool, forceDisable bool) int {
	want := CullStateOff
	if !forceDisable && !hasCutout {
		switch cullMode {
		case config.CullFront:
			want = CullStateBack
		case config.CullBack:
			want = CullStateFront
		case config.CullNone:
			want = CullStateOff
		default:
			want = CullStateBack
		}
	}
	if want == curState {
		return curState
	}
	switch want {
	case CullStateOff:
		ctx.Disable(api.CULL_FACE)
	case CullStateBack:
		if curState <= CullStateOff {
			ctx.Enable(api.CULL_FACE)
		}
		ctx.CullFace(api.BACK)
	case CullStateFront:
		if curState <= CullStateOff {
			ctx.Enable(api.CULL_FACE)
		}
		ctx.CullFace(api.FRONT)
	}
	return want
}
