package aabb

// AABBNullNode represents an invalid or uninitialized node in an AABBTree, commonly used as a sentinel value.
const AABBNullNode = 0xffffffff

// Node represents a node in an AABB tree, used for spatial partitioning of objects in a 3D space.
type Node struct {
	aabb            *AABB
	object          IAABB
	parentNodeIndex uint
	leftNodeIndex   uint
	rightNodeIndex  uint
	nextNodeIndex   uint
}

// NewNode creates and returns a new instance of an Node with default uninitialized values.
func NewNode() *Node {
	node := &Node{
		aabb:            &AABB{},
		object:          nil,
		parentNodeIndex: AABBNullNode,
		leftNodeIndex:   AABBNullNode,
		rightNodeIndex:  AABBNullNode,
		nextNodeIndex:   AABBNullNode,
	}
	return node
}

// IsLeaf checks if the current Node is a leaf node by verifying if its leftNodeIndex is equal to AABBNullNode.
func (a *Node) IsLeaf() bool {
	return a.leftNodeIndex == AABBNullNode
}
