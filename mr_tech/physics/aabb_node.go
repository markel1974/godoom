package physics

// AABBNullNode represents an invalid or uninitialized node in an AABBTree, commonly used as a sentinel value.
const AABBNullNode = 0xffffffff

// AABBNode represents a node in an AABB tree, used for spatial partitioning of objects in a 3D space.
type AABBNode struct {
	aabb            *AABB
	object          IAABB
	parentNodeIndex uint
	leftNodeIndex   uint
	rightNodeIndex  uint
	nextNodeIndex   uint
}

// NewAABBNode creates and returns a new instance of an AABBNode with default uninitialized values.
func NewAABBNode() *AABBNode {
	node := &AABBNode{
		aabb:            &AABB{},
		object:          nil,
		parentNodeIndex: AABBNullNode,
		leftNodeIndex:   AABBNullNode,
		rightNodeIndex:  AABBNullNode,
		nextNodeIndex:   AABBNullNode,
	}
	return node
}

// IsLeaf checks if the current AABBNode is a leaf node by verifying if its leftNodeIndex is equal to AABBNullNode.
func (a *AABBNode) IsLeaf() bool {
	return a.leftNodeIndex == AABBNullNode
}
