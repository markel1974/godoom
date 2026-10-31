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
	activeLen      int
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
		activeStates: make([]*OcclusionState, maxEntities),
		activeLen:    0,
	}
	hw.allVerticesPtr = ctx.Ptr(hw.allVertices)

	for idx := range hw.tests {
		hw.tests[idx] = &AABBTest{}
	}

	// Allocate HW queries in batch
	ctx.GenQueries(int32(maxEntities), &hw.queryIDs[0])

	ctx.GenVertexArrays(1, &hw.cubeVAO)
	ctx.BindVertexArray(hw.cubeVAO)

	ctx.GenBuffers(1, &hw.cubeVBO)
	ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
	ctx.BufferData(api.ARRAY_BUFFER, len(hw.allVertices)*4, nil, api.DYNAMIC_DRAW)

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
	if state == nil || !hw.CanSchedule() || state.QueryActive {
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
	if hw.activeLen == 0 {
		return
	}
	writeIdx := 0
	for readIdx := 0; readIdx < hw.activeLen; readIdx++ {
		state := hw.activeStates[readIdx]
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
			hw.activeStates[writeIdx] = state
			writeIdx++
		}
	}
	for i := writeIdx; i < hw.activeLen; i++ {
		hw.activeStates[i] = nil
	}
	hw.activeLen = writeIdx
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
				if tc+108 >= len(hw.allVertices) {
					hw.growAllVertices()
				}
				mX, mY, mZ := float32(test.AABB.GetMinX()), float32(test.AABB.GetMinY()), float32(test.AABB.GetMinZ())
				xX, xY, xZ := float32(test.AABB.GetMaxX()), float32(test.AABB.GetMaxY()), float32(test.AABB.GetMaxZ())
				nmY, nxY := -mY, -xY

				dst := hw.allVertices[tc : tc+108]
				// Back
				dst[0] = mX
				dst[1] = mZ
				dst[2] = nmY
				dst[3] = xX
				dst[4] = mZ
				dst[5] = nmY
				dst[6] = xX
				dst[7] = mZ
				dst[8] = nxY
				dst[9] = xX
				dst[10] = mZ
				dst[11] = nxY
				dst[12] = mX
				dst[13] = mZ
				dst[14] = nxY
				dst[15] = mX
				dst[16] = mZ
				dst[17] = nmY
				// Right
				dst[18] = xX
				dst[19] = mZ
				dst[20] = nmY
				dst[21] = xX
				dst[22] = xZ
				dst[23] = nmY
				dst[24] = xX
				dst[25] = xZ
				dst[26] = nxY
				dst[27] = xX
				dst[28] = xZ
				dst[29] = nxY
				dst[30] = xX
				dst[31] = mZ
				dst[32] = nxY
				dst[33] = xX
				dst[34] = mZ
				dst[35] = nmY
				// Front
				dst[36] = xX
				dst[37] = xZ
				dst[38] = nmY
				dst[39] = mX
				dst[40] = xZ
				dst[41] = nmY
				dst[42] = mX
				dst[43] = xZ
				dst[44] = nxY
				dst[45] = mX
				dst[46] = xZ
				dst[47] = nxY
				dst[48] = xX
				dst[49] = xZ
				dst[50] = nxY
				dst[51] = xX
				dst[52] = xZ
				dst[53] = nmY
				// Left
				dst[54] = mX
				dst[55] = xZ
				dst[56] = nmY
				dst[57] = mX
				dst[58] = mZ
				dst[59] = nmY
				dst[60] = mX
				dst[61] = mZ
				dst[62] = nxY
				dst[63] = mX
				dst[64] = mZ
				dst[65] = nxY
				dst[66] = mX
				dst[67] = xZ
				dst[68] = nxY
				dst[69] = mX
				dst[70] = xZ
				dst[71] = nmY
				// Top
				dst[72] = mX
				dst[73] = mZ
				dst[74] = nxY
				dst[75] = xX
				dst[76] = mZ
				dst[77] = nxY
				dst[78] = xX
				dst[79] = xZ
				dst[80] = nxY
				dst[81] = xX
				dst[82] = xZ
				dst[83] = nxY
				dst[84] = mX
				dst[85] = xZ
				dst[86] = nxY
				dst[87] = mX
				dst[88] = mZ
				dst[89] = nxY
				// Bottom
				dst[90] = mX
				dst[91] = xZ
				dst[92] = nmY
				dst[93] = xX
				dst[94] = xZ
				dst[95] = nmY
				dst[96] = xX
				dst[97] = mZ
				dst[98] = nmY
				dst[99] = xX
				dst[100] = mZ
				dst[101] = nmY
				dst[102] = mX
				dst[103] = mZ
				dst[104] = nmY
				dst[105] = mX
				dst[106] = xZ
				dst[107] = nmY

				tc += 108
				vc++
			}
		}

		if vc > 0 {
			hw.ctx.BindVertexArray(hw.cubeVAO)
			hw.ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
			hw.ctx.BufferSubData(api.ARRAY_BUFFER, 0, tc*4, hw.allVerticesPtr)

			idx := int32(0)
			for i := 0; i < hw.testsLen; i++ {
				test := hw.tests[i]
				if !test.State.QueryActive {
					hw.ctx.BeginQuery(api.ANY_SAMPLES_PASSED, test.State.QueryID)
					hw.ctx.DrawArrays(api.TRIANGLES, idx*36, 36)
					hw.ctx.EndQuery(api.ANY_SAMPLES_PASSED)
					test.State.QueryActive = true
					if hw.activeLen >= len(hw.activeStates) {
						hw.growActiveStates()
					}
					hw.activeStates[hw.activeLen] = test.State
					hw.activeLen++
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

// growAllVertices doubles the capacity of the allVertices slice and updates the associated pointer in the GPU context.
func (hw *OcclusionHW) growAllVertices() {
	newSize := len(hw.allVertices) * 2
	newData := make([]float32, newSize)
	copy(newData, hw.allVertices)
	hw.allVertices = newData
	hw.allVerticesPtr = hw.ctx.Ptr(hw.allVertices)

	hw.ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
	hw.ctx.BufferData(api.ARRAY_BUFFER, len(hw.allVertices)*4, nil, api.DYNAMIC_DRAW)
	hw.ctx.BindBuffer(api.ARRAY_BUFFER, 0)
}

// growActiveStates doubles the capacity of the activeStates slice, initializing it to 1024 if empty, and preserves existing data.
func (hw *OcclusionHW) growActiveStates() {
	newSize := len(hw.activeStates) * 2
	if newSize == 0 {
		newSize = 1024
	}
	newData := make([]*OcclusionState, newSize)
	copy(newData, hw.activeStates)
	hw.activeStates = newData
}
