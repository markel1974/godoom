package buffers

import (
	"math"
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

type poolBufferDataCall struct {
	vbo    uint32
	size   int
	hasPtr bool
	usage  uint32
}

type poolBufferSubCall struct {
	vbo    uint32
	offset int
	size   int
}

type poolTestCtx struct {
	api.IContext
	nextID       uint32
	boundArray   uint32
	dataCalls    []poolBufferDataCall
	subDataCalls []poolBufferSubCall
}

func (c *poolTestCtx) resetLogs() {
	c.dataCalls = c.dataCalls[:0]
	c.subDataCalls = c.subDataCalls[:0]
}

func (c *poolTestCtx) GenBuffers(n int32, buffers *uint32) {
	s := unsafe.Slice(buffers, int(n))
	for i := range s {
		c.nextID++
		s[i] = c.nextID
	}
}

func (c *poolTestCtx) BindBuffer(target uint32, buffer uint32) {
	if target == api.ARRAY_BUFFER {
		c.boundArray = buffer
	}
}

func (c *poolTestCtx) BufferData(target uint32, size int, data unsafe.Pointer, usage uint32) {
	if target != api.ARRAY_BUFFER {
		return
	}
	c.dataCalls = append(c.dataCalls, poolBufferDataCall{
		vbo:    c.boundArray,
		size:   size,
		hasPtr: data != nil,
		usage:  usage,
	})
}

func (c *poolTestCtx) BufferSubData(target uint32, offset int, size int, _ unsafe.Pointer) {
	if target != api.ARRAY_BUFFER {
		return
	}
	c.subDataCalls = append(c.subDataCalls, poolBufferSubCall{
		vbo:    c.boundArray,
		offset: offset,
		size:   size,
	})
}

func TestInstancePoolOrphaningRingAndUnchangedSkip(t *testing.T) {
	ctx := &poolTestCtx{}
	pool := NewInstancePool(ctx, 64)

	writeSample := func(offset float64) {
		pool.Reset()
		_, bf := pool.Reserve(2)
		pool.WriteTransformAt(bf, 10+offset, 20, 30, 0, 0, 0, 1)
		var mat [InstanceMatrixFloats]float32
		BuildTransformMatrix(mat[:], 40, 50, 60, 0, 0, math.Sin(0.3), math.Cos(0.3))
		pool.WriteMatrixAt(bf+InstanceMatrixFloats, &mat)
	}

	// Frame 1: initial upload allocates VBO #1 and uploads once.
	writeSample(0)
	vbo1 := pool.Upload()
	if vbo1 == 0 || pool.ActiveVBO() != vbo1 {
		t.Fatalf("expected non-zero initial VBO, got vbo=%d active=%d", vbo1, pool.ActiveVBO())
	}
	if len(ctx.dataCalls) != 1 || ctx.dataCalls[0].hasPtr || len(ctx.subDataCalls) != 1 {
		t.Fatalf("expected 1 BufferData(nil) + 1 BufferSubData on initial upload, got data=%+v sub=%+v", ctx.dataCalls, ctx.subDataCalls)
	}

	// Frame 2: identical instance matrices skip GPU upload completely and keep vbo1.
	ctx.resetLogs()
	writeSample(0)
	if got := pool.Upload(); got != vbo1 {
		t.Fatalf("unchanged upload returned VBO %d, want %d", got, vbo1)
	}
	if len(ctx.dataCalls) != 0 || len(ctx.subDataCalls) != 0 {
		t.Fatalf("expected 0 GL calls on unchanged InstancePool upload, got data=%+v sub=%+v", ctx.dataCalls, ctx.subDataCalls)
	}

	allocs := testing.AllocsPerRun(50, func() {
		writeSample(0)
		_ = pool.Upload()
	})
	if allocs != 0 {
		t.Fatalf("expected 0 allocs on steady-state InstancePool.Upload, got %.2f", allocs)
	}

	// Changing transforms rotates cleanly across all 4 ring VBOs and orphans before BufferSubData.
	seenVBOs := map[uint32]bool{vbo1: true}
	var prevVBO uint32
	for step := 1; step <= 5; step++ {
		ctx.resetLogs()
		writeSample(float64(step))
		vbo := pool.Upload()
		if vbo == 0 {
			t.Fatalf("step %d returned zero VBO", step)
		}
		seenVBOs[vbo] = true
		if len(ctx.dataCalls) != 1 || ctx.dataCalls[0].hasPtr || len(ctx.subDataCalls) != 1 {
			t.Fatalf("step %d expected orphan BufferData(nil) + BufferSubData, got data=%+v sub=%+v", step, ctx.dataCalls, ctx.subDataCalls)
		}
		if step <= 3 && vbo == prevVBO {
			t.Fatalf("step %d reused previous VBO %d instead of advancing 4-slot ring", step, vbo)
		}
		prevVBO = vbo
	}
	if len(seenVBOs) != instancePoolBuffers {
		t.Fatalf("expected %d distinct ring VBOs used, got %d (%v)", instancePoolBuffers, len(seenVBOs), seenVBOs)
	}
}

func TestInstancePoolWriteMatrixMatchesWriteTransformAt(t *testing.T) {
	ctx := &poolTestCtx{}
	pool := NewInstancePool(ctx, 64)
	pool.Reset()
	_, bf := pool.Reserve(2)

	qx, qy, qz, qw := 0.1, -0.2, 0.3, 0.9
	pool.WriteTransformAt(bf, 12.5, -34.25, 8.75, qx, qy, qz, qw)

	var mat [InstanceMatrixFloats]float32
	BuildTransformMatrix(mat[:], 12.5, -34.25, 8.75, qx, qy, qz, qw)
	pool.WriteMatrixAt(bf+InstanceMatrixFloats, &mat)

	for i := range InstanceMatrixFloats {
		a := pool.data[bf+i]
		b := pool.data[bf+InstanceMatrixFloats+i]
		if a != b {
			t.Fatalf("matrix lane %d mismatch: WriteTransformAt=%v WriteMatrixAt=%v", i, a, b)
		}
	}
}
