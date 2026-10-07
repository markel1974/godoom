package open_gl

import (
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// ANY_SAMPLES_PASSED is a constant representing a query target for checking if any samples passed the test.
// QUERY_RESULT_AVAILABLE is a constant representing the status of whether the query result is available.
// QUERY_RESULT is a constant representing the result value of a query.
const (
	ANY_SAMPLES_PASSED     = 0x8C2F
	QUERY_RESULT_AVAILABLE = 0x8867
	QUERY_RESULT           = 0x8866
)

// OcclusionState represents the state of a hardware occlusion query, including its ID, activation status, and visibility.
type OcclusionState struct {
	QueryID     uint32
	QueryActive bool
	IsVisible   bool
}

// AABBTest represents a test case for occlusion queries involving an axis-aligned bounding box and its state.
type AABBTest struct {
	AABB  *physics.AABB
	State *OcclusionState
}

// OcclusionHW manages hardware-based occlusion queries using OpenGL-like graphics operations.
type OcclusionHW struct {
	ctx      api.IContext
	states   map[uint64]*OcclusionState
	queryIDs []uint32
	queryIdx int
	tests    []*AABBTest
	testsLen int
	cubeVAO  uint32
	cubeVBO  uint32
}

// NewOcclusionHW initializes a new hardware occlusion object with context and maximum entity support.
func NewOcclusionHW(ctx api.IContext, maxEntities int) *OcclusionHW {
	hw := &OcclusionHW{
		ctx:      ctx,
		states:   make(map[uint64]*OcclusionState, maxEntities),
		tests:    make([]*AABBTest, 1024),
		testsLen: 0,
		queryIDs: make([]uint32, maxEntities),
		queryIdx: 0,
	}

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
		state = &OcclusionState{
			QueryID:     hw.queryIDs[hw.queryIdx],
			QueryActive: false,
			IsVisible:   true,
		}
		hw.states[id] = state
		hw.queryIdx++
	}
	return state
}

// Reset clears all AABB tests stored in the OcclusionHW instance.
func (hw *OcclusionHW) Reset() {
	hw.testsLen = 0
}

// Add appends a new AABBTest containing the given AABB and OcclusionState to the tests slice if state is not nil.
func (hw *OcclusionHW) Add(aabb *physics.AABB, state *OcclusionState) {
	if state == nil {
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

// RenderQueries performs occlusion queries by rendering AABBs and updating their visibility status asynchronously.
// RenderQueries performs occlusion queries by rendering AABBs and updating their visibility status asynchronously.
func (hw *OcclusionHW) RenderQueries() {
	if hw.testsLen == 0 {
		return
	}

	// Regenerate the VBO buffer with all valid AABBs
	var allVertices []float32
	validCount := 0

	for i := 0; i < hw.testsLen; i++ {
		test := hw.tests[i]
		if !test.State.QueryActive {
			mX, mY, mZ := float32(test.AABB.GetMinX()), float32(test.AABB.GetMinY()), float32(test.AABB.GetMinZ())
			xX, xY, xZ := float32(test.AABB.GetMaxX()), float32(test.AABB.GetMaxY()), float32(test.AABB.GetMaxZ())

			// AABB vertices in Engine space
			p0 := [3]float32{mX, mY, mZ}
			p1 := [3]float32{xX, mY, mZ}
			p2 := [3]float32{xX, xY, mZ}
			p3 := [3]float32{mX, xY, mZ}
			p4 := [3]float32{mX, mY, xZ}
			p5 := [3]float32{xX, mY, xZ}
			p6 := [3]float32{xX, xY, xZ}
			p7 := [3]float32{mX, xY, xZ}

			// SWAP for the OpenGL shader: (X, Z, -Y)
			sw := func(p [3]float32) [3]float32 {
				return [3]float32{p[0], p[2], -p[1]}
			}
			v0, v1, v2, v3 := sw(p0), sw(p1), sw(p2), sw(p3)
			v4, v5, v6, v7 := sw(p4), sw(p5), sw(p6), sw(p7)

			// 36 vertices (12 triangles)
			vertices := []float32{
				v0[0], v0[1], v0[2], v1[0], v1[1], v1[2], v2[0], v2[1], v2[2], v2[0], v2[1], v2[2], v3[0], v3[1], v3[2], v0[0], v0[1], v0[2], // Back
				v1[0], v1[1], v1[2], v5[0], v5[1], v5[2], v6[0], v6[1], v6[2], v6[0], v6[1], v6[2], v2[0], v2[1], v2[2], v1[0], v1[1], v1[2], // Right
				v5[0], v5[1], v5[2], v4[0], v4[1], v4[2], v7[0], v7[1], v7[2], v7[0], v7[1], v7[2], v6[0], v6[1], v6[2], v5[0], v5[1], v5[2], // Front
				v4[0], v4[1], v4[2], v0[0], v0[1], v0[2], v3[0], v3[1], v3[2], v3[0], v3[1], v3[2], v7[0], v7[1], v7[2], v4[0], v4[1], v4[2], // Left
				v3[0], v3[1], v3[2], v2[0], v2[1], v2[2], v6[0], v6[1], v6[2], v6[0], v6[1], v6[2], v7[0], v7[1], v7[2], v3[0], v3[1], v3[2], // Top
				v4[0], v4[1], v4[2], v5[0], v5[1], v5[2], v1[0], v1[1], v1[2], v1[0], v1[1], v1[2], v0[0], v0[1], v0[2], v4[0], v4[1], v4[2], // Bottom
			}
			allVertices = append(allVertices, vertices...)
			validCount++
		}
	}

	if validCount > 0 {
		hw.ctx.BindVertexArray(hw.cubeVAO)
		hw.ctx.BindBuffer(api.ARRAY_BUFFER, hw.cubeVBO)
		hw.ctx.BufferData(api.ARRAY_BUFFER, len(allVertices)*4, hw.ctx.Ptr(allVertices), api.DYNAMIC_DRAW)

		idx := int32(0)
		for i := 0; i < hw.testsLen; i++ {
			test := hw.tests[i]
			if !test.State.QueryActive {
				hw.ctx.BeginQuery(ANY_SAMPLES_PASSED, test.State.QueryID)
				hw.ctx.DrawArrays(api.TRIANGLES, idx*36, 36)
				hw.ctx.EndQuery(ANY_SAMPLES_PASSED)
				test.State.QueryActive = true
				idx++
			}
		}
		hw.ctx.BindVertexArray(0)
	}

	// DEFERRED READ-BACK (Non-blocking)
	for i := 0; i < hw.testsLen; i++ {
		test := hw.tests[i]
		if test.State.QueryActive {
			var available uint32
			hw.ctx.GetQueryObjectuiv(test.State.QueryID, QUERY_RESULT_AVAILABLE, &available)
			if available == 1 {
				var passed uint32
				hw.ctx.GetQueryObjectuiv(test.State.QueryID, QUERY_RESULT, &passed)

				test.State.IsVisible = passed > 0
				test.State.QueryActive = false
			}
		}
	}
}
