//go:build js && wasm

package web

import (
	"math"
	"syscall/js"
	"testing"
	"unsafe"
)

func newTestContext(t *testing.T) *Context {
	t.Helper()
	ctx := &Context{
		uniforms: []js.Value{js.Null()},
		ptrMap:   make(map[uintptr]interface{}),
	}
	ctx.initJSTypes()
	return ctx
}

func assertAttachedView(t *testing.T, view js.Value, wantLen int) {
	t.Helper()
	if view.IsNull() || view.IsUndefined() {
		t.Fatalf("expected non-null TypedArray view")
	}
	if view.Get("buffer").Get("byteLength").Int() == 0 {
		t.Fatalf("underlying ArrayBuffer is detached or empty")
	}
	if got := view.Length(); got != wantLen {
		t.Fatalf("unexpected view length: got %d, want %d", got, wantLen)
	}
}

func TestTexSubImageAndMemoryGrowNoDetach(t *testing.T) {
	ctx := newTestContext(t)

	var lastView js.Value
	ctx.fn_texSubImage2D = js.FuncOf(func(_ js.Value, args []js.Value) any {
		lastView = args[8]
		return nil
	}).Value

	// Matches Lights.PrepareClusters parameters: 128x132 RGBA16UI / UNSIGNED_SHORT.
	const rows = 132
	const width = 128
	clusterSlice := make([]uint16, width*rows*4)
	clusterSlice[0] = 0x1234
	clusterSlice[len(clusterSlice)-1] = 0xABCD

	ctx.TexSubImage2D(0x0DE1, 0, 0, 0, width, rows, 0x8D99, 0x1403, ctx.Ptr(clusterSlice))
	assertAttachedView(t, lastView, len(clusterSlice))
	if got := lastView.Index(0).Int(); got != 0x1234 {
		t.Fatalf("unexpected first uint16 element: got %#x, want 0x1234", got)
	}
	if got := lastView.Index(len(clusterSlice) - 1).Int(); got != 0xABCD {
		t.Fatalf("unexpected last uint16 element: got %#x, want 0xABCD", got)
	}

	// Verify V8 can still construct views over lastView.buffer after forcing wasm memory.grow().
	keepAlive := make([][]byte, 0, 6)
	for i := 0; i < 6; i++ {
		blk := make([]byte, 8*1024*1024)
		blk[0] = byte(i + 1)
		keepAlive = append(keepAlive, blk)
	}
	_ = keepAlive[0][0]

	clone := ctx.jsUint16Array.New(lastView.Get("buffer"), lastView.Get("byteOffset"), lastView.Length())
	if clone.Length() != len(clusterSlice) {
		t.Fatalf("view invalidated after wasm memory.grow(): got %d, want %d", clone.Length(), len(clusterSlice))
	}

	// Subsequent larger TexImage2D + TexImage3D + TexSubImage3D must reuse/grow scratch cleanly.
	var lastTexImage2D, lastTexImage3D, lastTexSub3D js.Value
	ctx.fn_texImage2D = js.FuncOf(func(_ js.Value, args []js.Value) any {
		lastTexImage2D = args[8]
		return nil
	}).Value
	ctx.fn_texImage3D = js.FuncOf(func(_ js.Value, args []js.Value) any {
		lastTexImage3D = args[9]
		return nil
	}).Value
	ctx.fn_texSubImage3D = js.FuncOf(func(_ js.Value, args []js.Value) any {
		lastTexSub3D = args[10]
		return nil
	}).Value

	fullBuf := make([]uint16, 128*1200*4)
	fullBuf[5] = 777
	ctx.TexImage2D(0x0DE1, 0, 0x8D76, 128, 1200, 0, 0x8D99, 0x1403, ctx.Ptr(fullBuf))
	assertAttachedView(t, lastTexImage2D, len(fullBuf))
	if lastTexImage2D.Index(5).Int() != 777 {
		t.Fatalf("unexpected TexImage2D payload element: got %d, want 777", lastTexImage2D.Index(5).Int())
	}

	ctx.TexImage3D(0x8C1A, 0, 0x8058, 4, 4, 1, 0, 0x1908, 0x1401, nil)
	if !lastTexImage3D.IsNull() {
		t.Fatalf("expected null pixels for nil TexImage3D")
	}

	pix3D := []byte{11, 22, 33, 44}
	ctx.TexSubImage3D(0x8C1A, 0, 0, 0, 0, 1, 1, 1, 0x1908, 0x1401, ctx.Ptr(pix3D))
	assertAttachedView(t, lastTexSub3D, 4)
	if lastTexSub3D.Index(2).Int() != 33 {
		t.Fatalf("unexpected TexSubImage3D element: got %d, want 33", lastTexSub3D.Index(2).Int())
	}
}

func TestBufferAndMultiDrawAndUniformUploads(t *testing.T) {
	ctx := newTestContext(t)

	var bufView, subBufView js.Value
	ctx.fn_bufferData = js.FuncOf(func(_ js.Value, args []js.Value) any {
		bufView = args[1]
		return nil
	}).Value
	ctx.fn_bufferSubData = js.FuncOf(func(_ js.Value, args []js.Value) any {
		subBufView = args[2]
		return nil
	}).Value

	// Test slice via ptrMap and sub-range direct pointer not in ptrMap.
	floats := []float32{1.5, 2.5, 3.5, 4.5}
	ctx.BufferData(0x8892, 16, ctx.Ptr(floats), 0x88E4)
	assertAttachedView(t, bufView, 16)

	ctx.BufferSubData(0x8892, 4, 8, unsafe.Pointer(&floats[1]))
	assertAttachedView(t, subBufView, 8)
	recovered := math.Float32frombits(uint32(subBufView.Index(0).Int()) |
		uint32(subBufView.Index(1).Int())<<8 |
		uint32(subBufView.Index(2).Int())<<16 |
		uint32(subBufView.Index(3).Int())<<24)
	if recovered != 2.5 {
		t.Fatalf("unexpected BufferSubData raw pointer value: got %v, want 2.5", recovered)
	}

	// Test MultiDrawElements passes two non-overlapping attached views simultaneously.
	var gotCounts, gotOffsets js.Value
	var gotDrawCount int
	ctx.fn_multiDrawElements = js.FuncOf(func(_ js.Value, args []js.Value) any {
		gotCounts = args[2]
		gotOffsets = args[4]
		gotDrawCount = args[5].Int()
		return nil
	}).Value

	counts := []int32{6, 12, 18}
	ptrs := []unsafe.Pointer{
		ctx.PtrOffset(0),
		ctx.PtrOffset(24),
		ctx.PtrOffset(72),
	}
	ctx.MultiDrawElements(0x0004, &counts[0], 0x1405, &ptrs[0], 3)
	if gotDrawCount != 3 {
		t.Fatalf("unexpected drawcount: got %d, want 3", gotDrawCount)
	}
	assertAttachedView(t, gotCounts, 3)
	assertAttachedView(t, gotOffsets, 3)
	if gotCounts.Index(1).Int() != 12 || gotOffsets.Index(1).Int() != 24 {
		t.Fatalf("MultiDrawElements scratch buffers collided or corrupted: count=%d offset=%d", gotCounts.Index(1).Int(), gotOffsets.Index(1).Int())
	}

	// Test Uniform1iv, UniformMatrix4fv (multi), Uniform3fv (multi).
	ctx.uniforms = append(ctx.uniforms, js.ValueOf("loc1"))
	var u1ivView, matView, v3View js.Value
	ctx.fn_uniform1iv = js.FuncOf(func(_ js.Value, args []js.Value) any {
		u1ivView = args[1]
		return nil
	}).Value
	ctx.fn_uniformMatrix4fv = js.FuncOf(func(_ js.Value, args []js.Value) any {
		matView = args[2]
		return nil
	}).Value
	ctx.fn_uniform3fv = js.FuncOf(func(_ js.Value, args []js.Value) any {
		v3View = args[1]
		return nil
	}).Value

	ints := []int32{10, 20, 30}
	ctx.Uniform1iv(1, 3, &ints[0])
	assertAttachedView(t, u1ivView, 3)
	if u1ivView.Index(2).Int() != 30 {
		t.Fatalf("unexpected Uniform1iv[2]: got %d, want 30", u1ivView.Index(2).Int())
	}

	mats := make([]float32, 32)
	mats[17] = 9.25
	ctx.UniformMatrix4fv(1, 2, false, &mats[0])
	assertAttachedView(t, matView, 32)
	if matView.Index(17).Float() != 9.25 {
		t.Fatalf("unexpected UniformMatrix4fv[17]: got %v, want 9.25", matView.Index(17).Float())
	}

	vecs := []float32{1, 2, 3, 4, 5, 6}
	ctx.Uniform3fv(1, 2, &vecs[0])
	assertAttachedView(t, v3View, 6)
	if v3View.Index(4).Float() != 5 {
		t.Fatalf("unexpected Uniform3fv[4]: got %v, want 5", v3View.Index(4).Float())
	}
}
