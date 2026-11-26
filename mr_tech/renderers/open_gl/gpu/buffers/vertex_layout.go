package buffers

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const (
	// StaticVertexWords is the 32-bit word stride for compressed static/instanced meshes (16 words = 64 bytes).
	StaticVertexWords = 16
	// StaticVertexBytes is the byte stride of a compressed static/instanced vertex.
	StaticVertexBytes = StaticVertexWords * 4
	// StaticVertexFloats aliases StaticVertexWords for legacy stride callers.
	StaticVertexFloats = StaticVertexWords

	// StreamingVertexWords is the 32-bit word stride for CPU-streamed morphing/billboard/UI meshes (24 words = 96 bytes).
	StreamingVertexWords = 24
	// StreamingVertexBytes is the byte stride of a streaming vertex.
	StreamingVertexBytes = StreamingVertexWords * 4
	// StreamingVertexFloats aliases StreamingVertexWords for legacy stride callers.
	StreamingVertexFloats = StreamingVertexWords

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
	// Location 0: aPos (vec3 float32, offset 0)
	ctx.VertexAttribPointer(0, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(0))
	ctx.EnableVertexAttribArray(0)
	// Location 1: aTexCoords (u, v_gl, texLayer) (vec3 float32, offset 12)
	ctx.VertexAttribPointer(1, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(3*4))
	ctx.EnableVertexAttribArray(1)
	// Location 2: aNormal (packed GL_INT_2_10_10_10_REV normalized, offset 24)
	ctx.VertexAttribPointer(2, 4, api.INT_2_10_10_10_REV, true, strideBytes, ctx.PtrOffset(6*4))
	ctx.EnableVertexAttribArray(2)
	// Location 3: aTangent (packed GL_INT_2_10_10_10_REV normalized, w = sign, offset 28)
	ctx.VertexAttribPointer(3, 4, api.INT_2_10_10_10_REV, true, strideBytes, ctx.PtrOffset(7*4))
	ctx.EnableVertexAttribArray(3)
	// Location 4: aLightmap (lmU, lmV, lmLayer, 0) (HALF_FLOAT x4, offset 32)
	ctx.VertexAttribPointer(4, 4, api.HALF_FLOAT, false, strideBytes, ctx.PtrOffset(8*4))
	ctx.EnableVertexAttribArray(4)
	// Location 5: aMaterial (specBoost, shininess, detailScale, detailFactor) (HALF_FLOAT x4, offset 40)
	ctx.VertexAttribPointer(5, 4, api.HALF_FLOAT, false, strideBytes, ctx.PtrOffset(10*4))
	ctx.EnableVertexAttribArray(5)
	// Location 6: aExtra (packed, envStrength, renderMode, envTintPacked) (float32 x4, offset 48)
	ctx.VertexAttribPointer(6, 4, api.FLOAT, false, strideBytes, ctx.PtrOffset(12*4))
	ctx.EnableVertexAttribArray(6)

	if includeDynamic {
		// Location 7: aOrigin (vec3 float32, offset 64)
		ctx.VertexAttribPointer(7, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(16*4))
		ctx.EnableVertexAttribArray(7)
		// Location 8: aPosNext (vec3 float32, offset 76)
		ctx.VertexAttribPointer(8, 3, api.FLOAT, false, strideBytes, ctx.PtrOffset(19*4))
		ctx.EnableVertexAttribArray(8)
		// Location 9: aAnimData (lerp, yaw) (vec2 float32, offset 88)
		ctx.VertexAttribPointer(9, 2, api.FLOAT, false, strideBytes, ctx.PtrOffset(22*4))
		ctx.EnableVertexAttribArray(9)
	}
}

// Float32ToFloat16 converts an IEEE-754 float32 into a finite IEEE-754 binary16 (round-to-nearest-even).
func Float32ToFloat16(f float32) uint16 {
	bits := math.Float32bits(f)
	sign := uint16((bits >> 16) & 0x8000)
	exp := int((bits >> 23) & 0xFF)
	mant := bits & 0x7FFFFF

	if exp == 0xFF {
		if mant != 0 {
			return 0
		}
		return sign | 0x7BFF
	}
	if exp == 0 {
		return sign
	}

	unbiased := exp - 127
	if unbiased > 15 {
		return sign | 0x7BFF
	}
	if unbiased < -24 {
		return sign
	}
	if unbiased < -14 {
		// Subnormal half-float
		mantFull := mant | 0x800000
		shift := uint32(-14 - unbiased + 13)
		roundBit := uint32(1) << (shift - 1)
		sub := mantFull >> shift
		rem := mantFull & ((uint32(1) << shift) - 1)
		if rem > roundBit || (rem == roundBit && (sub&1) != 0) {
			sub++
		}
		return sign | uint16(sub)
	}

	halfExp := uint32(unbiased + 15)
	halfMant := mant >> 13
	rem := mant & 0x1FFF
	if rem > 0x1000 || (rem == 0x1000 && (halfMant&1) != 0) {
		halfMant++
		if halfMant == 0x400 {
			halfMant = 0
			halfExp++
			if halfExp >= 31 {
				return sign | 0x7BFF
			}
		}
	}
	return sign | uint16((halfExp<<10)|halfMant)
}

// Float16ToFloat32 decodes an IEEE-754 binary16 half-float to float32.
func Float16ToFloat32(h uint16) float32 {
	sign := uint32(h&0x8000) << 16
	exp := uint32((h >> 10) & 0x1F)
	mant := uint32(h & 0x3FF)

	if exp == 0 {
		if mant == 0 {
			return math.Float32frombits(sign)
		}
		shift := uint32(1)
		for (mant & 0x400) == 0 {
			mant <<= 1
			shift++
		}
		mant &= 0x3FF
		exp32 := uint32(127 - 15 - shift + 1)
		return math.Float32frombits(sign | (exp32 << 23) | (mant << 13))
	}
	if exp == 0x1F {
		if mant == 0 {
			return math.Float32frombits(sign | 0x7F800000)
		}
		return math.Float32frombits(sign | 0x7FC00000 | (mant << 13))
	}
	exp32 := exp + (127 - 15)
	return math.Float32frombits(sign | (exp32 << 23) | (mant << 13))
}

// PackHalf2 packs two float32 values into two little-endian IEEE binary16 lanes inside a uint32.
func PackHalf2(a, b float32) uint32 {
	return uint32(Float32ToFloat16(a)) | (uint32(Float32ToFloat16(b)) << 16)
}

// UnpackHalf2 decodes two little-endian IEEE binary16 values from a uint32.
func UnpackHalf2(w uint32) (float32, float32) {
	return Float16ToFloat32(uint16(w)), Float16ToFloat32(uint16(w >> 16))
}

func quantizeSnorm10(v float32) uint32 {
	if v > 1.0 {
		v = 1.0
	} else if v < -1.0 {
		v = -1.0
	}
	var q int32
	if v >= 0 {
		q = int32(v*511.0 + 0.5)
	} else {
		q = int32(v*511.0 - 0.5)
	}
	return uint32(q) & 0x3FF
}

// PackSnorm10x3_2 packs (x, y, z, w) into GL_INT_2_10_10_10_REV (10-10-10-2 signed normalized bits).
func PackSnorm10x3_2(x, y, z, w float32, normalizeXYZ bool) uint32 {
	if normalizeXYZ {
		lenSq := float64(x)*float64(x) + float64(y)*float64(y) + float64(z)*float64(z)
		if lenSq > 1e-12 && math.Abs(lenSq-1.0) > 1e-6 {
			inv := float32(1.0 / math.Sqrt(lenSq))
			x *= inv
			y *= inv
			z *= inv
		}
	}
	var qw int32
	if w > 0 {
		qw = 1
	} else if w < 0 {
		qw = -1
	}
	return quantizeSnorm10(x) |
		(quantizeSnorm10(y) << 10) |
		(quantizeSnorm10(z) << 20) |
		((uint32(qw) & 0x3) << 30)
}

// UnpackSnorm10x3_2 decodes a GL_INT_2_10_10_10_REV signed normalized 4-vector back to float32 components.
func UnpackSnorm10x3_2(p uint32) (x, y, z, w float32) {
	signExt10 := func(v uint32) float32 {
		s := int32(v<<22) >> 22
		if s < -511 {
			s = -511
		}
		return float32(s) / 511.0
	}
	x = signExt10(p & 0x3FF)
	y = signExt10((p >> 10) & 0x3FF)
	z = signExt10((p >> 20) & 0x3FF)
	sw := int32(p) >> 30
	if sw < -1 {
		sw = -1
	}
	w = float32(sw)
	return x, y, z, w
}

// EncodeVertexBaseGL writes the 16 compressed 32-bit words (64 bytes, locations 0..6) in OpenGL coordinate space.
func EncodeVertexBaseGL(dst []uint32, x, y, z, u, v, texLayer, nx, ny, nz, tx, ty, tz, tSign, lmu, lmv, lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint float32) {
	w := dst[:StaticVertexWords:StaticVertexWords]
	// Location 0: aPos
	w[0] = math.Float32bits(x)
	w[1] = math.Float32bits(y)
	w[2] = math.Float32bits(z)
	// Location 1: aTexCoords
	w[3] = math.Float32bits(u)
	w[4] = math.Float32bits(v)
	w[5] = math.Float32bits(texLayer)
	// Location 2 & 3: aNormal + aTangent packed INT_2_10_10_10_REV
	w[6] = PackSnorm10x3_2(nx, ny, nz, 0, true)
	signW := float32(1.0)
	if tSign < 0 {
		signW = -1.0
	}
	w[7] = PackSnorm10x3_2(tx, ty, tz, signW, true)
	// Location 4: aLightmap (HALF_FLOAT x4)
	w[8] = PackHalf2(lmu, lmv)
	w[9] = PackHalf2(lmIDFloat, 0)
	// Location 5: aMaterial (HALF_FLOAT x4)
	w[10] = PackHalf2(spec, shininess)
	w[11] = PackHalf2(dScale, dFactor)
	// Location 6: aExtra (float32 x4)
	w[12] = math.Float32bits(packed)
	w[13] = math.Float32bits(envStr)
	w[14] = math.Float32bits(renderMode)
	w[15] = math.Float32bits(envTint)
}

// WriteStaticVertex encodes a static vertex from engine coordinates into the 16-word (64-byte) OpenGL representation.
func WriteStaticVertex(dst []uint32, px, py, pz, u, v, texLayer, nx, ny, nz, tx, ty, tz, tSign, lmu, lmv, lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint float32) {
	EncodeVertexBaseGL(dst, px, pz, -py, u, -v, texLayer, nx, nz, -ny, tx, tz, -ty, tSign, lmu, lmv, lmIDFloat, packed, spec, shininess, dScale, dFactor, envStr, renderMode, envTint)
}
