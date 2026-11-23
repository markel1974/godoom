package queries

import (
	"testing"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

type occlusionRecordingCtx struct {
	api.IContext

	nextID           uint32
	beginCalls       int
	endCalls         int
	drawCalls        int
	queryAvailCalls  int
	queryResultCalls int
	queryAvailable   uint32
	queryPassed      uint32
}

func newOcclusionRecordingCtx() *occlusionRecordingCtx {
	return &occlusionRecordingCtx{nextID: 1}
}

func (c *occlusionRecordingCtx) allocIDs(n int32, out *uint32) {
	if out == nil || n <= 0 {
		return
	}
	s := unsafe.Slice(out, int(n))
	for i := range s {
		s[i] = c.nextID
		c.nextID++
	}
}

func (c *occlusionRecordingCtx) BindBuffer(_ uint32, _ uint32)                          {}
func (c *occlusionRecordingCtx) BindVertexArray(_ uint32)                               {}
func (c *occlusionRecordingCtx) BufferData(_ uint32, _ int, _ unsafe.Pointer, _ uint32) {}
func (c *occlusionRecordingCtx) BufferSubData(_ uint32, _ int, _ int, _ unsafe.Pointer) {}
func (c *occlusionRecordingCtx) EnableVertexAttribArray(_ uint32)                       {}
func (c *occlusionRecordingCtx) GenBuffers(n int32, b *uint32)                          { c.allocIDs(n, b) }
func (c *occlusionRecordingCtx) GenQueries(n int32, q *uint32)                          { c.allocIDs(n, q) }
func (c *occlusionRecordingCtx) GenVertexArrays(n int32, a *uint32)                     { c.allocIDs(n, a) }
func (c *occlusionRecordingCtx) Ptr(data interface{}) unsafe.Pointer {
	if s, ok := data.([]float32); ok && len(s) > 0 {
		return unsafe.Pointer(&s[0])
	}
	return nil
}
func (c *occlusionRecordingCtx) PtrOffset(offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(offset))
}
func (c *occlusionRecordingCtx) VertexAttribPointer(_ uint32, _ int32, _ uint32, _ bool, _ int32, _ unsafe.Pointer) {
}

func (c *occlusionRecordingCtx) BeginQuery(_ uint32, _ uint32) { c.beginCalls++ }
func (c *occlusionRecordingCtx) EndQuery(_ uint32)             { c.endCalls++ }
func (c *occlusionRecordingCtx) DrawArrays(_ uint32, _ int32, _ int32) {
	c.drawCalls++
}
func (c *occlusionRecordingCtx) GetQueryObjectuiv(_ uint32, pname uint32, params *uint32) {
	if pname == api.QUERY_RESULT_AVAILABLE {
		c.queryAvailCalls++
		*params = c.queryAvailable
		return
	}
	if pname == api.QUERY_RESULT {
		c.queryResultCalls++
		*params = c.queryPassed
	}
}

func TestOcclusionHWDefaultsToThrottledNonDebugAndDefersReadback(t *testing.T) {
	ctx := newOcclusionRecordingCtx()
	hw := NewOcclusionHW(ctx, 16)

	if hw.GetDebug() {
		t.Fatalf("OcclusionHW must initialize with debug=false for production throttling")
	}
	if hw.GetInterval() != defaultOcclusionInterval {
		t.Fatalf("expected default occlusion interval %d, got %d", defaultOcclusionInterval, hw.GetInterval())
	}

	state := hw.GetState(42)
	if state == nil {
		t.Fatalf("expected valid OcclusionState")
	}
	box := aabb.NewAABB()
	box.Rebuild(0, 0, 0, 10, 10, 10)

	// Frame counter 0 is schedulable (% 4 == 0).
	hw.Add(box, state)
	if !hw.HasDrawTests() {
		t.Fatalf("expected scheduled AABB test on frameCounter 0")
	}

	// Render issues BeginQuery + EndQuery WITHOUT polling GetQueryObjectuiv synchronously in the same frame.
	hw.Render()
	if ctx.beginCalls != 1 || ctx.endCalls != 1 || ctx.drawCalls != 1 {
		t.Fatalf("unexpected query calls on Render: begin=%d end=%d draw=%d", ctx.beginCalls, ctx.endCalls, ctx.drawCalls)
	}
	if ctx.queryAvailCalls != 0 || ctx.queryResultCalls != 0 {
		t.Fatalf("Render must not call Readback synchronously right after EndQuery (avail=%d res=%d)",
			ctx.queryAvailCalls, ctx.queryResultCalls)
	}
	if !state.QueryActive {
		t.Fatalf("expected state.QueryActive == true while GPU query is in flight")
	}

	// Simulate GPU finishing the query by next frame: Reset() must read back previous-frame result BEFORE new scheduling.
	ctx.queryAvailable = 1
	ctx.queryPassed = 0 // occluded!
	hw.Reset()
	if ctx.queryAvailCalls != 1 || ctx.queryResultCalls != 1 {
		t.Fatalf("expected Reset() to read back pending query (avail=%d res=%d)", ctx.queryAvailCalls, ctx.queryResultCalls)
	}
	if state.QueryActive {
		t.Fatalf("expected state.QueryActive == false after completed Readback")
	}
	if state.IsVisible {
		t.Fatalf("expected state.IsVisible == false after passed=0 readback")
	}

	// Non-scheduled interval frames (frameCounter == 1) must not emit queries when debug == false.
	hw.Add(box, state)
	if hw.HasDrawTests() {
		t.Fatalf("must not schedule query off-interval when debug == false")
	}

	// Enabling debug mode forces every-frame box submission; disabling restores throttling.
	hw.SetDebug(true)
	hw.Add(box, state)
	if !hw.HasDrawTests() {
		t.Fatalf("expected debug mode to submit bounding box test regardless of interval")
	}
	hw.SetDebug(false)
	hw.Reset() // frameCounter == 2
	hw.Add(box, state)
	if hw.HasDrawTests() {
		t.Fatalf("expected throttling to resume after SetDebug(false)")
	}
}
