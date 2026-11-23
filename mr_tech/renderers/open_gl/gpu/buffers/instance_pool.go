package buffers

import (
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// InstancePool accumulates per-instance 4x4 transform matrices for the active frame
// and uploads them once per frame into a double-buffered GPU VBO.
type InstancePool struct {
	ctx      api.IContext
	data     []float32
	ptr      unsafe.Pointer
	count    int32
	vbo      [2]uint32
	vboCap   [2]int
	frameIdx int
}

// NewInstancePool allocates an InstancePool with initial instance matrix capacity.
func NewInstancePool(ctx api.IContext, initInstances int) *InstancePool {
	if initInstances < 64 {
		initInstances = 64
	}
	data := make([]float32, initInstances*InstanceMatrixFloats)
	return &InstancePool{
		ctx:  ctx,
		data: data,
		ptr:  unsafe.Pointer(&data[0]),
	}
}

// Reset clears the instance list at the start of a frame while advancing the double-buffer slot.
func (p *InstancePool) Reset() {
	p.count = 0
	p.frameIdx = (p.frameIdx + 1) & 1
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

// WriteTransformAt converts world position (cx, cy, cz) and physics quaternion (qx, qy, qz, qw)
// into a column-major GL mat4 at float offset baseFloat.
func (p *InstancePool) WriteTransformAt(baseFloat int, cx, cy, cz, qx, qy, qz, qw float64) {
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

	d := p.data[baseFloat : baseFloat+16]
	// Col 0
	d[0] = float32(r00)
	d[1] = float32(r20)
	d[2] = float32(-r10)
	d[3] = 0
	// Col 1
	d[4] = float32(r02)
	d[5] = float32(r22)
	d[6] = float32(-r12)
	d[7] = 0
	// Col 2
	d[8] = float32(-r01)
	d[9] = float32(-r21)
	d[10] = float32(r11)
	d[11] = 0
	// Col 3
	d[12] = float32(cx)
	d[13] = float32(cz)
	d[14] = float32(-cy)
	d[15] = 1
}

// ActiveVBO returns the current double-buffered GPU buffer ID containing this frame's instance matrices.
func (p *InstancePool) ActiveVBO() uint32 {
	return p.vbo[p.frameIdx]
}

// Upload transfers accumulated instance matrices to the active GPU VBO in one buffer call.
func (p *InstancePool) Upload() uint32 {
	if p.count == 0 {
		return 0
	}
	idx := p.frameIdx
	if p.vbo[idx] == 0 {
		p.ctx.GenBuffers(1, &p.vbo[idx])
	}
	bytesNeeded := int(p.count) * (InstanceMatrixFloats * 4)
	p.ctx.BindBuffer(api.ARRAY_BUFFER, p.vbo[idx])
	if bytesNeeded > p.vboCap[idx] {
		newCap := max(bytesNeeded*2, 64*InstanceMatrixFloats*4)
		p.ctx.BufferData(api.ARRAY_BUFFER, newCap, nil, api.DYNAMIC_DRAW)
		p.vboCap[idx] = newCap
	}
	p.ctx.BufferSubData(api.ARRAY_BUFFER, 0, bytesNeeded, p.ptr)
	p.ctx.BindBuffer(api.ARRAY_BUFFER, 0)
	return p.vbo[idx]
}
