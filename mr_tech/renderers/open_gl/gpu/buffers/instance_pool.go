package buffers

import (
	"bytes"
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

const instancePoolBuffers = 4

// InstancePool accumulates per-instance 4x4 transform matrices for the active frame
// and uploads them into a ring-buffered GPU VBO, skipping redundant uploads when unchanged.
type InstancePool struct {
	ctx        api.IContext
	data       []float32
	uploaded   []float32
	ptr        unsafe.Pointer
	count      int32
	lastVBO    uint32
	lastFloats int
	vbo        [instancePoolBuffers]uint32
	vboCap     [instancePoolBuffers]int
	frameIdx   int
}

// NewInstancePool allocates an InstancePool with initial instance matrix capacity.
func NewInstancePool(ctx api.IContext, initInstances int) *InstancePool {
	if initInstances < 64 {
		initInstances = 64
	}
	floats := initInstances * InstanceMatrixFloats
	data := make([]float32, floats)
	return &InstancePool{
		ctx:      ctx,
		data:     data,
		uploaded: make([]float32, floats),
		ptr:      unsafe.Pointer(&data[0]),
	}
}

// Reset clears the staging instance count at the start of a frame while preserving the last uploaded VBO until Upload sees changes.
func (p *InstancePool) Reset() {
	p.count = 0
}

// Reserve allocates consecutive matrix slots and returns the base byte offset and base float index.
func (p *InstancePool) Reserve(numInstances int) (baseByteOffset int32, baseFloat int) {
	needed := (int(p.count) + numInstances) * InstanceMatrixFloats
	if needed > len(p.data) {
		newLen := len(p.data) * 2
		for newLen < needed {
			newLen *= 2
		}
		grown := make([]float32, newLen)
		copy(grown, p.data[:int(p.count)*InstanceMatrixFloats])
		p.data = grown
		p.ptr = unsafe.Pointer(&p.data[0])
	}
	baseByteOffset = p.count * (InstanceMatrixFloats * 4)
	baseFloat = int(p.count) * InstanceMatrixFloats
	p.count += int32(numInstances)
	return baseByteOffset, baseFloat
}

// BuildTransformMatrix converts world position (cx, cy, cz) and physics quaternion (qx, qy, qz, qw)
// into a column-major GL mat4 stored in dst (length >= InstanceMatrixFloats).
func BuildTransformMatrix(dst []float32, cx, cy, cz, qx, qy, qz, qw float64) {
	// Normalize quaternion defensively
	n2 := qx*qx + qy*qy + qz*qz + qw*qw
	if n2 > 1e-12 && math.Abs(n2-1.0) > 1e-6 {
		inv := 1.0 / math.Sqrt(n2)
		qx, qy, qz, qw = qx*inv, qy*inv, qz*inv, qw*inv
	} else if n2 <= 1e-12 {
		qx, qy, qz, qw = 0, 0, 0, 1
	}

	xx, yy, zz := qx*qx, qy*qy, qz*qz
	xy, xz, yz := qx*qy, qx*qz, qy*qz
	wx, wy, wz := qw*qx, qw*qy, qw*qz

	r00 := 1.0 - 2.0*(yy+zz)
	r01 := 2.0 * (xy - wz)
	r02 := 2.0 * (xz + wy)

	r10 := 2.0 * (xy + wz)
	r11 := 1.0 - 2.0*(xx+zz)
	r12 := 2.0 * (yz - wx)

	r20 := 2.0 * (xz - wy)
	r21 := 2.0 * (yz + wx)
	r22 := 1.0 - 2.0*(xx+yy)

	_ = dst[15]
	// Col 0
	dst[0] = float32(r00)
	dst[1] = float32(r20)
	dst[2] = float32(-r10)
	dst[3] = 0
	// Col 1
	dst[4] = float32(r02)
	dst[5] = float32(r22)
	dst[6] = float32(-r12)
	dst[7] = 0
	// Col 2
	dst[8] = float32(-r01)
	dst[9] = float32(-r21)
	dst[10] = float32(r11)
	dst[11] = 0
	// Col 3
	dst[12] = float32(cx)
	dst[13] = float32(cz)
	dst[14] = float32(-cy)
	dst[15] = 1
}

// WriteTransformAt converts world position (cx, cy, cz) and physics quaternion (qx, qy, qz, qw)
// into a column-major GL mat4 at float offset baseFloat.
func (p *InstancePool) WriteTransformAt(baseFloat int, cx, cy, cz, qx, qy, qz, qw float64) {
	BuildTransformMatrix(p.data[baseFloat:baseFloat+InstanceMatrixFloats], cx, cy, cz, qx, qy, qz, qw)
}

// WriteMatrixAt copies a precomputed column-major GL mat4 into the staging buffer at float offset baseFloat.
func (p *InstancePool) WriteMatrixAt(baseFloat int, mat *[InstanceMatrixFloats]float32) {
	copy(p.data[baseFloat:baseFloat+InstanceMatrixFloats], mat[:])
}

// ActiveVBO returns the latest uploaded GPU buffer ID containing active instance matrices.
func (p *InstancePool) ActiveVBO() uint32 {
	return p.lastVBO
}

// Upload transfers accumulated instance matrices to an unsynchronized ring-buffered GPU VBO,
// returning the existing GPU VBO with zero driver calls when contents match the previous upload.
func (p *InstancePool) Upload() uint32 {
	if p.count == 0 {
		return 0
	}
	floatsNeeded := int(p.count) * InstanceMatrixFloats
	bytesNeeded := floatsNeeded * 4
	if p.lastVBO != 0 && p.lastFloats == floatsNeeded && len(p.uploaded) >= floatsNeeded {
		cur := unsafe.Slice((*byte)(p.ptr), bytesNeeded)
		prev := unsafe.Slice((*byte)(unsafe.Pointer(&p.uploaded[0])), bytesNeeded)
		if bytes.Equal(cur, prev) {
			return p.lastVBO
		}
	}
	if p.lastVBO != 0 {
		p.frameIdx = (p.frameIdx + 1) & (instancePoolBuffers - 1)
	}
	idx := p.frameIdx
	if p.vbo[idx] == 0 {
		p.ctx.GenBuffers(1, &p.vbo[idx])
	}
	p.ctx.BindBuffer(api.ARRAY_BUFFER, p.vbo[idx])
	if bytesNeeded > p.vboCap[idx] {
		newCap := max(bytesNeeded*2, 64*InstanceMatrixFloats*4)
		p.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		p.vboCap[idx] = newCap
	} else {
		// Orphan existing storage so glBufferSubData never stalls on in-flight GPU draws.
		p.ctx.BufferData(api.ARRAY_BUFFER, p.vboCap[idx], nil, api.DYNAMIC_DRAW)
	}
	p.ctx.BufferSubData(api.ARRAY_BUFFER, 0, bytesNeeded, p.ptr)
	p.ctx.BindBuffer(api.ARRAY_BUFFER, 0)

	if len(p.uploaded) < len(p.data) {
		p.uploaded = make([]float32, len(p.data))
	}
	copy(p.uploaded[:floatsNeeded], p.data[:floatsNeeded])
	p.lastFloats = floatsNeeded
	p.lastVBO = p.vbo[idx]
	return p.vbo[idx]
}
