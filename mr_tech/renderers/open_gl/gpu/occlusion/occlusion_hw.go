package occlusion

import (
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/physics/aabb"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

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
	debug          bool
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
		debug:        false,
	}
	hw.allVerticesPtr = ctx.Ptr(hw.allVertices)

	for idx := range hw.tests {
		hw.tests[idx] = NewAABBTest()
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

// Reset reads back completed queries from previous frames, clears all AABB tests, and advances the frame counter.
func (hw *OcclusionHW) Reset() {
	hw.Readback()
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

// SetDebug enables or disables continuous rendering of bounding box meshes for debugging.
func (hw *OcclusionHW) SetDebug(debug bool) {
	hw.debug = debug
}

// GetDebug reports whether debug bounding box rendering is active.
func (hw *OcclusionHW) GetDebug() bool {
	return hw.debug
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
	if state == nil {
		return
	}
	if !hw.debug && (!hw.CanSchedule() || state.QueryActive) {
		return
	}
	if hw.testsLen >= len(hw.tests) {
		t := hw.tests
		oldLen := len(hw.tests)
		newLen := len(hw.tests) + 1024
		hw.tests = make([]*AABBTest, newLen)
		copy(hw.tests, t)
		for i := oldLen; i < newLen; i++ {
			hw.tests[i] = NewAABBTest()
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
		if available != 0 {
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
	if hw.testsLen == 0 {
		return
	}
	vc := 0
	tc := 0
	for i := 0; i < hw.testsLen; i++ {
		test := hw.tests[i]
		if hw.debug || !test.State.QueryActive {
			if tc+108 >= len(hw.allVertices) {
				hw.growAllVertices()
			}
			writeOcclusionBoxVertices(hw.allVertices[tc:tc+108], test.AABB)
			tc += 108
			vc++
		}
	}
	if vc == 0 {
		return
	}

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
		} else if hw.debug {
			hw.ctx.DrawArrays(api.TRIANGLES, idx*36, 36)
			idx++
		}
	}
	hw.ctx.BindVertexArray(0)
	hw.ctx.BindBuffer(api.ARRAY_BUFFER, 0)
}

func writeOcclusionBoxVertices(dst []float32, box *aabb.AABB) {
	const occMargin = float32(0.05)
	mX, mY, mZ := float32(box.GetMinX())-occMargin, float32(box.GetMinY())-occMargin, float32(box.GetMinZ())-occMargin
	xX, xY, xZ := float32(box.GetMaxX())+occMargin, float32(box.GetMaxY())+occMargin, float32(box.GetMaxZ())+occMargin
	nmY, nxY := -mY, -xY

	// Bottom (Y = mZ, outward -Y)
	dst[0], dst[1], dst[2] = mX, mZ, nxY
	dst[3], dst[4], dst[5] = xX, mZ, nxY
	dst[6], dst[7], dst[8] = xX, mZ, nmY
	dst[9], dst[10], dst[11] = mX, mZ, nxY
	dst[12], dst[13], dst[14] = xX, mZ, nmY
	dst[15], dst[16], dst[17] = mX, mZ, nmY
	// Right (X = xX, outward +X)
	dst[18], dst[19], dst[20] = xX, mZ, nmY
	dst[21], dst[22], dst[23] = xX, mZ, nxY
	dst[24], dst[25], dst[26] = xX, xZ, nxY
	dst[27], dst[28], dst[29] = xX, mZ, nmY
	dst[30], dst[31], dst[32] = xX, xZ, nxY
	dst[33], dst[34], dst[35] = xX, xZ, nmY
	// Top (Y = xZ, outward +Y)
	dst[36], dst[37], dst[38] = mX, xZ, nmY
	dst[39], dst[40], dst[41] = xX, xZ, nmY
	dst[42], dst[43], dst[44] = xX, xZ, nxY
	dst[45], dst[46], dst[47] = mX, xZ, nmY
	dst[48], dst[49], dst[50] = xX, xZ, nxY
	dst[51], dst[52], dst[53] = mX, xZ, nxY
	// Left (X = mX, outward -X)
	dst[54], dst[55], dst[56] = mX, mZ, nxY
	dst[57], dst[58], dst[59] = mX, mZ, nmY
	dst[60], dst[61], dst[62] = mX, xZ, nmY
	dst[63], dst[64], dst[65] = mX, mZ, nxY
	dst[66], dst[67], dst[68] = mX, xZ, nmY
	dst[69], dst[70], dst[71] = mX, xZ, nxY
	// Back (Z = nxY, outward -Z)
	dst[72], dst[73], dst[74] = xX, mZ, nxY
	dst[75], dst[76], dst[77] = mX, mZ, nxY
	dst[78], dst[79], dst[80] = mX, xZ, nxY
	dst[81], dst[82], dst[83] = xX, mZ, nxY
	dst[84], dst[85], dst[86] = mX, xZ, nxY
	dst[87], dst[88], dst[89] = xX, xZ, nxY
	// Front (Z = nmY, outward +Z)
	dst[90], dst[91], dst[92] = mX, mZ, nmY
	dst[93], dst[94], dst[95] = xX, mZ, nmY
	dst[96], dst[97], dst[98] = xX, xZ, nmY
	dst[99], dst[100], dst[101] = mX, mZ, nmY
	dst[102], dst[103], dst[104] = xX, xZ, nmY
	dst[105], dst[106], dst[107] = mX, xZ, nmY
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
