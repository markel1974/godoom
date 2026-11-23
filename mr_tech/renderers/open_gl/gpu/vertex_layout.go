package gpu

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const (
	// StaticVertexFloats is the float32 stride for GPU-resident static/instanced meshes (24 floats = 96 bytes).
	StaticVertexFloats = 24
	// StreamingVertexFloats is the float32 stride for CPU-streamed morphing/billboard/UI meshes (32 floats = 128 bytes).
	StreamingVertexFloats = 32
	// InstanceMatrixFloats is the float32 stride per rigid instance mat4 transform (16 floats = 64 bytes).
	InstanceMatrixFloats = 16
)

const (
	// InstanceModeStatic selects direct resident static geometry (locations 0..6, u_instanceMode = 0).
	InstanceModeStatic = int32(0)
	// InstanceModeInstanced selects GPU instanced geometry (locations 0..6 + mat4 at 10..13, u_instanceMode = 1).
	InstanceModeInstanced = int32(1)
	// InstanceModeStreaming selects dynamic CPU-streamed morphing/billboard/UI geometry (locations 0..9, u_instanceMode = 2).
	InstanceModeStreaming = int32(2)
)

// ConfigureVertexAttribs configures vertex attribute pointers 0..6 (and optionally 7..9 for streaming) on the active VAO.
func ConfigureVertexAttribs(ctx api.IContext, strideBytes int32, includeDynamic bool) {
	// Location 0: aPos (vec3)
	ctx.VertexAttribPointer(0, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(0))
	ctx.EnableVertexAttribArray(0)
	// Location 1: aTexCoords (u, v_gl, texLayer) (vec3)
	ctx.VertexAttribPointer(1, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(3*4))
	ctx.EnableVertexAttribArray(1)
	// Location 2: aNormal (vec3 GL space)
	ctx.VertexAttribPointer(2, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(6*4))
	ctx.EnableVertexAttribArray(2)
	// Location 3: aTangent (tx, ty, tz, sign) (vec4 GL space)
	ctx.VertexAttribPointer(3, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(9*4))
	ctx.EnableVertexAttribArray(3)
	// Location 4: aLightmap (lmU, lmV, lmLayer) (vec3)
	ctx.VertexAttribPointer(4, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(13*4))
	ctx.EnableVertexAttribArray(4)
	// Location 5: aMaterial (packed, specBoost, shininess, detailScale) (vec4)
	ctx.VertexAttribPointer(5, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(16*4))
	ctx.EnableVertexAttribArray(5)
	// Location 6: aExtra (detailFactor, envStrength, renderMode, envTintPacked) (vec4)
	ctx.VertexAttribPointer(6, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(20*4))
	ctx.EnableVertexAttribArray(6)

	if includeDynamic {
		// Location 7: aOrigin (vec3)
		ctx.VertexAttribPointer(7, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(24*4))
		ctx.EnableVertexAttribArray(7)
		// Location 8: aPosNext (vec3)
		ctx.VertexAttribPointer(8, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(27*4))
		ctx.EnableVertexAttribArray(8)
		// Location 9: aAnimData (lerp, yaw) (vec2)
		ctx.VertexAttribPointer(9, 2, api.FLOAT, false, strideBytes, ctx.PtrOffset(30*4))
		ctx.EnableVertexAttribArray(9)
	}
}

// WriteStaticVertex encodes a static vertex in OpenGL coordinate space into the 24-float dst slice.
func WriteStaticVertex(dst []float32, px, py, pz, u, v float64, texLayer float32, nx, ny, nz, tx, ty, tz, tSign, lmu, lmv float64, lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint float32) {
	dst[0] = float32(px)
	dst[1] = float32(pz)
	dst[2] = float32(-py)
	dst[3] = float32(u)
	dst[4] = float32(-v)
	dst[5] = texLayer
	dst[6] = float32(nx)
	dst[7] = float32(nz)
	dst[8] = float32(-ny)
	dst[9] = float32(tx)
	dst[10] = float32(tz)
	dst[11] = float32(-ty)
	dst[12] = float32(tSign)
	dst[13] = float32(lmu)
	dst[14] = float32(lmv)
	dst[15] = lmIDFloat
	dst[16] = packed
	dst[17] = spec
	dst[18] = shininess
	dst[19] = dScale
	dst[20] = dFactor
	dst[21] = envStr
	dst[22] = renderMode
	dst[23] = envTint
}
