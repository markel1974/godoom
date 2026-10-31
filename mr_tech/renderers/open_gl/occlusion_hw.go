package open_gl

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// OcclusionState represents the state of a hardware occlusion query, including its ID, activation status, and visibility.
type OcclusionState struct {
	QueryID     uint32
	QueryActive bool
	IsVisible   bool
	LastFrame   int
}

// NewOcclusionState creates and returns a new OcclusionState with the given query ID and initial visibility status.
func NewOcclusionState(id uint32, isVisible bool) *OcclusionState {
	state := &OcclusionState{
		QueryID:     id,
		QueryActive: false,
		IsVisible:   isVisible,
	}
	return state
}

// AABBTest represents a test case for occlusion queries involving an axis-aligned bounding box and its state.
type AABBTest struct {
	AABB  *aabb.AABB
	State *OcclusionState
}

// OcclusionHW manages hardware-based occlusion queries using OpenGL-like graphics operations.
type OcclusionHW struct {
	ctx            api.IContext
	states         map[uint64]*OcclusionState
	queryIDs       []uint32
	queryIdx       int
	tests          []*AABBTest
	testsLen       int
	cubeVAO        uint32
	cubeVBO        uint32
	allVertices    []float32
	allVerticesPtr unsafe.Pointer
	interval       int
	frameCounter   int
	activeStates   []*OcclusionState
}

const defaultOcclusionInterval = 4

// NewOcclusionHW initializes a new hardware occlusion object with context and maximum entity support.
func NewOcclusionHW(ctx api.IContext, maxEntities int) *OcclusionHW {
	hw := &OcclusionHW{
		ctx:          ctx,
		states:       make(map[uint64]*OcclusionState, maxEntities),
		tests:        make([]*AABBTest, 8192),
		allVertices:  make([]float32, 1024*1024),
		testsLen:     0,
		queryIDs:     make([]uint32, maxEntities),
		queryIdx:     0,
		interval:     defaultOcclusionInterval,
		frameCounter: 0,
		activeStates: make([]*OcclusionState, 0, 1024),
	}
	hw.allVerticesPtr = ctx.Ptr(hw.allVertices)

	for idx := range hw.tests {
		hw.tests[idx] = &AABBTest{}
	}

	// Allocate HW queries in batch
	ctx.GenQueries(int32(maxEntities), &hw.queryIDs[0])

	// Create the bounding box geometry (1x1x1 cube from 0.0 to 1.0)
	// This way, by multiplying by (Max-Min) and translating to (Min),
	// the cube will perfectly cover any AABB.
	vertices := []float32{
		0, 0, 0, 1, 0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 0, 0, 0, 0, // Back
		1, 0, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 0, 0, // Right
		1, 0, 1, 0, 0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 1, // Front
		0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, // Left
		0, 1, 0, 1, 1, 0, 1, 1, 1, 1, 1, 1, 0, 1, 1, 0, 1, 0, // Top
		0, 0, 1, 1, 0, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, // Bottom
	}

	ctx.GenVertexArrays(1, &hw.cubeVAO)
	ctx.BindVertexArray(hw.cubeVAO)

	ctx.GenBuffers(1, &hw.cubeVBO)
	ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
	ctx.BufferData(api.ARRAY_BUFFER, len(vertices)*4, ctx.Ptr(vertices), api.STATIC_DRAW)

	// aPos (location = 0)
	ctx.VertexAttribPointer(0, 3, api.FLOAT, false, 3*4, ctx.PtrOffset(0))
	ctx.EnableVertexAttribArray(0)
	ctx.BindVertexArray(0)

	return hw
}

// GetState retrieves the OcclusionState at the specified index if the index is within bounds; otherwise, it returns nil.
func (hw *OcclusionHW) GetState(id uint64) *OcclusionState {
	state, exists := hw.states[id]
	if !exists {
		// Alloca una nuova query dalla pool pre-allocata
		if hw.queryIdx >= len(hw.queryIDs) {
			return nil // Pool esaurito!
		}
		state = NewOcclusionState(hw.queryIDs[hw.queryIdx], true)
		state.LastFrame = hw.frameCounter
		hw.states[id] = state
		hw.queryIdx++
		return state
	}
	if hw.frameCounter-state.LastFrame > hw.interval+1 {
		state.IsVisible = true
	}
	state.LastFrame = hw.frameCounter
	return state
}

// Reset clears all AABB tests stored in the OcclusionHW instance and advances the frame counter.
func (hw *OcclusionHW) Reset() {
	hw.testsLen = 0
	hw.frameCounter++
}

// CanSchedule checks if the current frame is scheduled to emit occlusion queries.
func (hw *OcclusionHW) CanSchedule() bool {
	return hw.frameCounter%hw.interval == 0
}

// HasDrawTests checks if there are AABB queries scheduled to be drawn this frame.
func (hw *OcclusionHW) HasDrawTests() bool {
	return hw.testsLen > 0
}

// SetInterval sets the frame interval between occlusion query draw passes.
func (hw *OcclusionHW) SetInterval(interval int) {
	if interval < 1 {
		interval = 1
	}
	hw.interval = interval
}

// GetInterval returns the frame interval between occlusion query draw passes.
func (hw *OcclusionHW) GetInterval() int {
	return hw.interval
}

// Add appends a new AABBTest containing the given AABB and OcclusionState to the tests slice if state is not nil and scheduling is active.
func (hw *OcclusionHW) Add(aabb *aabb.AABB, state *OcclusionState) {
	if state == nil || !hw.CanSchedule() {
		return
	}
	if hw.testsLen >= len(hw.tests) {
		t := hw.tests
		oldLen := len(hw.tests)
		newLen := len(hw.tests) + 1024
		hw.tests = make([]*AABBTest, newLen)
		copy(hw.tests, t)
		for i := oldLen; i < newLen; i++ {
			hw.tests[i] = &AABBTest{}
		}
	}
	test := hw.tests[hw.testsLen]
	test.AABB = aabb
	test.State = state
	hw.testsLen++
}

// Readback checks pending queries and updates their visibility status asynchronously without blocking.
func (hw *OcclusionHW) Readback() {
	if len(hw.activeStates) == 0 {
		return
	}
	remaining := hw.activeStates[:0]
	for _, state := range hw.activeStates {
		if !state.QueryActive {
			continue
		}
		var available uint32
		hw.ctx.GetQueryObjectuiv(state.QueryID, api.QUERY_RESULT_AVAILABLE, &available)
		if available == 1 {
			var passed uint32
			hw.ctx.GetQueryObjectuiv(state.QueryID, api.QUERY_RESULT, &passed)

			state.IsVisible = passed > 0
			state.QueryActive = false
		} else {
			remaining = append(remaining, state)
		}
	}
	hw.activeStates = remaining
}

// Render performs occlusion queries by rendering AABBs and updating their visibility status asynchronously.
func (hw *OcclusionHW) Render() {
	if hw.testsLen > 0 {
		// Regenerate the VBO buffer with all valid AABBs
		vc := 0
		tc := 0
		for i := 0; i < hw.testsLen; i++ {
			test := hw.tests[i]
			if !test.State.QueryActive {
				mX, mY, mZ := float32(test.AABB.GetMinX()), float32(test.AABB.GetMinY()), float32(test.AABB.GetMinZ())
				xX, xY, xZ := float32(test.AABB.GetMaxX()), float32(test.AABB.GetMaxY()), float32(test.AABB.GetMaxZ())

				// AABB vertices in Engine space SWAP for the OpenGL shader: (X, Z, -Y)
				v0 := [3]float32{mX, mZ, -mY}
				v1 := [3]float32{xX, mZ, -mY}
				v2 := [3]float32{xX, mZ, -xY}
				v3 := [3]float32{mX, mZ, -xY}
				v4 := [3]float32{mX, xZ, -mY}
				v5 := [3]float32{xX, xZ, -mY}
				v6 := [3]float32{xX, xZ, -xY}
				v7 := [3]float32{mX, xZ, -xY}

				// 36 vertices (12 triangles)
				vertices := [108]float32{
					v0[0], v0[1], v0[2], v1[0], v1[1], v1[2], v2[0], v2[1], v2[2], v2[0], v2[1], v2[2], v3[0], v3[1], v3[2], v0[0], v0[1], v0[2], // Back
					v1[0], v1[1], v1[2], v5[0], v5[1], v5[2], v6[0], v6[1], v6[2], v6[0], v6[1], v6[2], v2[0], v2[1], v2[2], v1[0], v1[1], v1[2], // Right
					v5[0], v5[1], v5[2], v4[0], v4[1], v4[2], v7[0], v7[1], v7[2], v7[0], v7[1], v7[2], v6[0], v6[1], v6[2], v5[0], v5[1], v5[2], // Front
					v4[0], v4[1], v4[2], v0[0], v0[1], v0[2], v3[0], v3[1], v3[2], v3[0], v3[1], v3[2], v7[0], v7[1], v7[2], v4[0], v4[1], v4[2], // Left
					v3[0], v3[1], v3[2], v2[0], v2[1], v2[2], v6[0], v6[1], v6[2], v6[0], v6[1], v6[2], v7[0], v7[1], v7[2], v3[0], v3[1], v3[2], // Top
					v4[0], v4[1], v4[2], v5[0], v5[1], v5[2], v1[0], v1[1], v1[2], v1[0], v1[1], v1[2], v0[0], v0[1], v0[2], v4[0], v4[1], v4[2], // Bottom
				}

				if tc+len(vertices) >= len(hw.allVertices) {
					hw.grow()
				}

				copy(hw.allVertices[tc:], vertices[:])

				tc += len(vertices)
				vc++
			}
		}

		if vc > 0 {
			hw.ctx.BindVertexArray(hw.cubeVAO)
			hw.ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
			hw.ctx.BufferData(api.ARRAY_BUFFER, tc*4, hw.allVerticesPtr, api.DYNAMIC_DRAW)

			idx := int32(0)
			for i := 0; i < hw.testsLen; i++ {
				test := hw.tests[i]
				if !test.State.QueryActive {
					hw.ctx.BeginQuery(api.ANY_SAMPLES_PASSED, test.State.QueryID)
					hw.ctx.DrawArrays(api.TRIANGLES, idx*36, 36)
					hw.ctx.EndQuery(api.ANY_SAMPLES_PASSED)
					test.State.QueryActive = true
					hw.activeStates = append(hw.activeStates, test.State)
					idx++
				}
			}
			hw.ctx.BindVertexArray(0)
			hw.ctx.BindBuffer(api.ARRAY_BUFFER, 0)
		}
	}

	// DEFERRED READ-BACK (Non-blocking)
	hw.Readback()
}

// grow doubles the size of the allVertices slice, copies existing data into the new slice, and updates its GPU pointer.
func (hw *OcclusionHW) grow() {
	newSize := len(hw.allVertices) * 2
	newData := make([]float32, newSize)
	copy(newData, hw.allVertices)
	hw.allVertices = newData
	hw.allVerticesPtr = hw.ctx.Ptr(hw.allVertices)
}
