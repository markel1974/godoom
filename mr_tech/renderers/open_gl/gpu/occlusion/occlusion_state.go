package occlusion

import "github.com/markel1974/godoom/mr_tech/physics/aabb"

// OcclusionState represents the state of a hardware occlusion query, including its ID, activation status, and visibility.
type OcclusionState struct {
	QueryID     uint32
	QueryActive bool
	IsVisible   bool
	LastFrame   int
}

// NewOcclusionState creates and returns a new OcclusionState with the given query ID and initial visibility status.
func NewOcclusionState(id uint32, isVisible bool) *OcclusionState {
	return &OcclusionState{
		QueryID:     id,
		QueryActive: false,
		IsVisible:   isVisible,
	}
}

// AABBTest represents a test case for occlusion queries involving an axis-aligned bounding box and its state.
type AABBTest struct {
	AABB  *aabb.AABB
	State *OcclusionState
}

// NewAABBTest creates and returns an empty AABBTest entry for the occlusion test pool.
func NewAABBTest() *AABBTest {
	return &AABBTest{}
}
