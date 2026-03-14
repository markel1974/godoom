package physics

import (
	"container/list"
	"math"
)

// IAABB defines an interface for objects that can return their associated Axis-Aligned Bounding Box (AABB).
type IAABB interface {
	GetAABB() *AABB
}

// AABB represents an axis-aligned bounding box defined by its minimum and maximum coordinates in 3D space.
type AABB struct {
	minX        float64
	minY        float64
	minZ        float64
	maxX        float64
	maxY        float64
	maxZ        float64
	surfaceArea float64
}

// calculateSurfaceArea computes and returns the surface area of the axis-aligned bounding box (AABB).
func (a *AABB) calculateSurfaceArea() float64 {
	s := 2.0 * (a.getWidth()*a.getHeight() + a.getWidth()*a.getDepth() + a.getHeight()*a.getDepth())
	return s
}

// NewAABB creates a new axis-aligned bounding box (AABB) with the specified minimum and maximum coordinates.
func NewAABB(minX float64, minY float64, minZ float64, maxX float64, maxY float64, maxZ float64) *AABB {
	a := &AABB{
		minX: minX,
		minY: minY,
		minZ: minZ,
		maxX: maxX,
		maxY: maxY,
		maxZ: maxZ,
	}

	a.surfaceArea = a.calculateSurfaceArea()

	return a
}

// overlaps checks if the current AABB intersects with another AABB in 3D space.
func (a *AABB) overlaps(other *AABB) bool {
	// y is deliberately first in the list of checks below as it is seen as more likely than things
	// collide on x,z but not on y than they do on y thus we drop out sooner on a y fail
	return a.maxX > other.minX &&
		a.minX < other.maxX &&
		a.maxY > other.minY &&
		a.minY < other.maxY &&
		a.maxZ > other.minZ &&
		a.minZ < other.maxZ
}

// contains determines if the current AABB fully encloses the specified AABB `other`.
func (a *AABB) contains(other *AABB) bool {
	return other.minX >= a.minX &&
		other.maxX <= a.maxX &&
		other.minY >= a.minY &&
		other.maxY <= a.maxY &&
		other.minZ >= a.minZ &&
		other.maxZ <= a.maxZ
}

// merge combines two AABBs into a new AABB that encapsulates both, preserving the smallest possible bounding volume.
func (a *AABB) merge(other *AABB) *AABB {
	b := NewAABB(math.Min(a.minX, other.minX), math.Min(a.minY, other.minY), math.Min(a.minZ, other.minZ),
		math.Max(a.maxX, other.maxX), math.Max(a.maxY, other.maxY), math.Max(a.maxZ, other.maxZ))
	return b
}

// intersection calculates and returns the overlapping region between the current AABB and another AABB, as a new AABB.
func (a *AABB) intersection(other *AABB) *AABB {
	b := NewAABB(math.Max(a.minX, other.minX), math.Max(a.minY, other.minY), math.Max(a.minZ, other.minZ),
		math.Min(a.maxX, other.maxX), math.Min(a.maxY, other.maxY), math.Min(a.maxZ, other.maxZ))
	return b
}

// getWidth computes and returns the width of the AABB as the difference between maxX and minX.
func (a *AABB) getWidth() float64 {
	return a.maxX - a.minX
}

// getHeight computes the height of the AABB as the difference between maxY and minY.
func (a *AABB) getHeight() float64 {
	return a.maxY - a.minY
}

// getDepth calculates and returns the depth of the AABB as the difference between maxZ and minZ.
func (a *AABB) getDepth() float64 {
	return a.maxZ - a.minZ
}

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

// isLeaf checks if the current AABBNode is a leaf node by verifying if its leftNodeIndex is equal to AABBNullNode.
func (a *AABBNode) isLeaf() bool {
	return a.leftNodeIndex == AABBNullNode
}

// AABBTree is a spatial partitioning data structure that organizes objects using an Axis-Aligned Bounding Box hierarchy.
type AABBTree struct {
	objectNodeIndexMap map[IAABB]uint
	nodes              []*AABBNode
	rootNodeIndex      uint
	allocatedNodeCount uint
	nextFreeNodeIndex  uint
	nodeCapacity       uint
	growthSize         uint
}

// NewAABBTree creates and initializes a new AABBTree with a specified initial size for the node capacity.
func NewAABBTree(initialSize uint) *AABBTree {
	t := &AABBTree{
		rootNodeIndex:      AABBNullNode,
		allocatedNodeCount: 0,
		nextFreeNodeIndex:  0,
		nodeCapacity:       initialSize,
		growthSize:         initialSize,
		nodes:              make([]*AABBNode, initialSize),
		objectNodeIndexMap: make(map[IAABB]uint),
		///nodes.resize(initialSize)
	}
	var nodeIndex uint

	for nodeIndex = 0; nodeIndex < initialSize; nodeIndex++ {
		node := NewAABBNode()
		t.nodes[nodeIndex] = node
		node.nextNodeIndex = nodeIndex + 1
	}
	t.nodes[initialSize-1].nextNodeIndex = AABBNullNode

	return t
}

// allocateNode allocates a new node in the tree, expanding the node array if needed, and returns the node index and instance.
func (a *AABBTree) allocateNode() (uint, *AABBNode) {
	if a.nextFreeNodeIndex == AABBNullNode {
		//assert(a.allocatedNodeCount == a.nodeCapacity)
		a.nodeCapacity += a.growthSize

		nodes := make([]*AABBNode, a.nodeCapacity)
		copy(nodes, a.nodes)
		a.nodes = nodes

		for nodeIndex := a.allocatedNodeCount; nodeIndex < a.nodeCapacity; nodeIndex++ {
			node := NewAABBNode()
			a.nodes[nodeIndex] = node
			node.nextNodeIndex = nodeIndex + 1
		}
		a.nodes[a.nodeCapacity-1].nextNodeIndex = AABBNullNode
		a.nextFreeNodeIndex = a.allocatedNodeCount
	}

	nodeIndex := a.nextFreeNodeIndex
	allocatedNode := a.nodes[nodeIndex]
	allocatedNode.parentNodeIndex = AABBNullNode
	allocatedNode.leftNodeIndex = AABBNullNode
	allocatedNode.rightNodeIndex = AABBNullNode
	a.nextFreeNodeIndex = allocatedNode.nextNodeIndex
	a.allocatedNodeCount++

	return nodeIndex, allocatedNode
}

// deallocateNode releases a node by adding it back to the free list and updating the next free node index.
func (a *AABBTree) deallocateNode(nodeIndex uint) {
	if len(a.nodes) == 0 {
		return
	}

	if nodeIndex >= 0 && nodeIndex < uint(len(a.nodes)) {
		deallocatedNode := a.nodes[nodeIndex]
		deallocatedNode.nextNodeIndex = a.nextFreeNodeIndex
		a.nextFreeNodeIndex = nodeIndex
		a.allocatedNodeCount--
	}
}

// Nodes returns a map associating IAABB objects with their corresponding node indices in the tree.
func (a *AABBTree) Nodes() map[IAABB]uint {
	return a.objectNodeIndexMap
}

/*
func (a * AABBTree) NodeAt(at uint) IAABB {
	if at < 0 || at >= uint(len(a.nodes)) {
		return nil
	}
	return a.nodes[at]
}
*/

// InsertObject inserts the given object into the AABB tree and updates the internal mappings and structure.
func (a *AABBTree) InsertObject(object IAABB) {
	nodeIndex, node := a.allocateNode()
	node.object = object
	node.aabb = object.GetAABB()

	a.insertLeaf(nodeIndex)
	a.objectNodeIndexMap[object] = nodeIndex
}

// RemoveObject removes the specified object from the AABBTree by deallocating its node and updating internal structures.
func (a *AABBTree) RemoveObject(object IAABB) {
	if nodeIndex, ok := a.objectNodeIndexMap[object]; ok {
		a.removeLeaf(nodeIndex)
		a.deallocateNode(nodeIndex)
		delete(a.objectNodeIndexMap, object)
	}
}

// UpdateObject updates the AABBTree to reflect changes in the AABB of the given object, if it already exists in the tree.
func (a *AABBTree) UpdateObject(object IAABB) {
	if nodeIndex, ok := a.objectNodeIndexMap[object]; ok {
		a.updateLeaf(nodeIndex, object.GetAABB())
	}
}

// QueryOverlaps returns a slice of IAABB objects that overlap with the given object in the AABBTree.
func (a *AABBTree) QueryOverlaps(object IAABB) []IAABB {
	var overlaps []IAABB

	stack := list.New()
	testAabb := object.GetAABB()

	stack.PushBack(a.rootNodeIndex)

	for stack.Len() > 0 {
		nodeIndex := stack.Back().Value.(uint)
		stack.Remove(stack.Back())
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		if node.aabb.overlaps(testAabb) {
			if node.isLeaf() && node.object != object {
				overlaps = append(overlaps, node.object)
			} else {
				stack.PushBack(node.leftNodeIndex)
				stack.PushBack(node.rightNodeIndex)
			}
		}
	}

	return overlaps
}

// insertLeaf inserts a new leaf node into the AABB tree at the optimal position based on surface area and depth heuristics.
func (a *AABBTree) insertLeaf(leafNodeIndex uint) {
	// make sure we're inserting a new leaf
	//assert(a.nodes[leafNodeIndex].parentNodeIndex == AABBNullNode)
	//assert(a.nodes[leafNodeIndex].leftNodeIndex == AABBNullNode)
	//assert(a.nodes[leafNodeIndex].rightNodeIndex == AABBNullNode)

	// if the tree is empty then we make the root the leaf
	if a.rootNodeIndex == AABBNullNode {
		a.rootNodeIndex = leafNodeIndex
		return
	}

	// search for the best place to put the new leaf in the tree
	// we use surface area and depth as search heuristics
	treeNodeIndex := a.rootNodeIndex
	leafNode := a.nodes[leafNodeIndex]

	for !a.nodes[treeNodeIndex].isLeaf() {
		//while !a.nodes[treeNodeIndex].isLeaf() {
		// because of the test in the while loop above we know we are never a leaf inside it
		treeNode := a.nodes[treeNodeIndex]
		leftNodeIndex := treeNode.leftNodeIndex
		rightNodeIndex := treeNode.rightNodeIndex
		leftNode := a.nodes[leftNodeIndex]
		rightNode := a.nodes[rightNodeIndex]

		combinedAabb := treeNode.aabb.merge(leafNode.aabb)

		newParentNodeCost := 2.0 * combinedAabb.surfaceArea
		minimumPushDownCost := 2.0 * (combinedAabb.surfaceArea - treeNode.aabb.surfaceArea)

		// use the costs to figure out whether to create a new parent here or descend
		var costLeft float64
		var costRight float64
		if leftNode.isLeaf() {
			costLeft = leafNode.aabb.merge(leftNode.aabb).surfaceArea + minimumPushDownCost
		} else {
			newLeftAabb := leafNode.aabb.merge(leftNode.aabb)
			costLeft = (newLeftAabb.surfaceArea - leftNode.aabb.surfaceArea) + minimumPushDownCost
		}
		if rightNode.isLeaf() {
			costRight = leafNode.aabb.merge(rightNode.aabb).surfaceArea + minimumPushDownCost
		} else {
			newRightAabb := leafNode.aabb.merge(rightNode.aabb)
			costRight = (newRightAabb.surfaceArea - rightNode.aabb.surfaceArea) + minimumPushDownCost
		}

		// if the cost of creating a new parent node here is less than descending in either direction then
		// we know we need to create a new parent node, here and attach the leaf to that
		if newParentNodeCost < costLeft && newParentNodeCost < costRight {
			break
		}

		// otherwise descend in the cheapest direction
		if costLeft < costRight {
			treeNodeIndex = leftNodeIndex
		} else {
			treeNodeIndex = rightNodeIndex
		}
	}

	// the leafs sibling is going to be the node we found above and we are going to create a new
	// parent node and attach the leaf and this item
	leafSiblingIndex := treeNodeIndex
	leafSibling := a.nodes[leafSiblingIndex]
	oldParentIndex := leafSibling.parentNodeIndex

	newParentIndex, newParent := a.allocateNode()
	newParent.parentNodeIndex = oldParentIndex
	newParent.aabb = leafNode.aabb.merge(leafSibling.aabb) // the new parents aabb is the leaf aabb combined with it's siblings aabb
	newParent.leftNodeIndex = leafSiblingIndex
	newParent.rightNodeIndex = leafNodeIndex

	leafNode.parentNodeIndex = newParentIndex
	leafSibling.parentNodeIndex = newParentIndex

	if oldParentIndex == AABBNullNode {
		// the old parent was the root and so this is now the root
		a.rootNodeIndex = newParentIndex
	} else {
		// the old parent was not the root and so we need to patch the left or right index to
		// point to the new node
		oldParent := a.nodes[oldParentIndex]
		if oldParent.leftNodeIndex == leafSiblingIndex {
			oldParent.leftNodeIndex = newParentIndex
		} else {
			oldParent.rightNodeIndex = newParentIndex
		}
	}

	// finally we need to walk back up the tree fixing heights and areas
	treeNodeIndex = leafNode.parentNodeIndex
	a.fixUpwardsTree(treeNodeIndex)
}

// removeLeaf removes a leaf node from the AABB tree and updates parent or sibling nodes as necessary.
func (a *AABBTree) removeLeaf(leafNodeIndex uint) {
	// if the leaf is the root then we can just clear the root pointer and return
	if leafNodeIndex == a.rootNodeIndex {
		a.rootNodeIndex = AABBNullNode
		a.deallocateNode(leafNodeIndex)
		return
	}

	leafNode := a.nodes[leafNodeIndex]
	parentNodeIndex := leafNode.parentNodeIndex
	parentNode := a.nodes[parentNodeIndex]
	grandParentNodeIndex := parentNode.parentNodeIndex
	var siblingNodeIndex uint
	if parentNode.leftNodeIndex == leafNodeIndex {
		siblingNodeIndex = parentNode.rightNodeIndex
	} else {
		siblingNodeIndex = parentNode.leftNodeIndex
	}
	//parentNode.leftNodeIndex == leafNodeIndex ? parentNode.rightNodeIndex : parentNode.leftNodeIndex
	//assert(siblingNodeIndex != AABBNullNode) // we must have a sibling
	siblingNode := a.nodes[siblingNodeIndex]

	if grandParentNodeIndex != AABBNullNode {
		// if we have a grand parent (i.e. the parent is not the root) then destroy the parent and connect the sibling to the grandparent in its
		// place
		grandParentNode := a.nodes[grandParentNodeIndex]
		if grandParentNode.leftNodeIndex == parentNodeIndex {
			grandParentNode.leftNodeIndex = siblingNodeIndex
		} else {
			grandParentNode.rightNodeIndex = siblingNodeIndex
		}
		siblingNode.parentNodeIndex = grandParentNodeIndex
		a.deallocateNode(parentNodeIndex)

		a.fixUpwardsTree(grandParentNodeIndex)
	} else {
		// if we have no grandparent then the parent is the root and so our sibling becomes the root and has it's parent removed
		a.rootNodeIndex = siblingNodeIndex
		siblingNode.parentNodeIndex = AABBNullNode
		a.deallocateNode(parentNodeIndex)
	}

	leafNode.parentNodeIndex = AABBNullNode
}

// updateLeaf updates the AABB of the specified leaf node and repositions it within the tree if necessary.
func (a *AABBTree) updateLeaf(leafNodeIndex uint, newAABB *AABB) {
	node := a.nodes[leafNodeIndex]

	// if the node contains the new aabb then we just leave things
	// TODO: when we add velocity this check should kick in as often an update will lie within the velocity fattened initial aabb
	// to support this we might need to differentiate between velocity fattened aabb and actual aabb

	//if node.aabb.contains(newAABB) {
	//	return
	//}

	a.removeLeaf(leafNodeIndex)
	node.aabb = newAABB
	a.insertLeaf(leafNodeIndex)
}

// fixUpwardsTree updates the AABB and height of all ancestor nodes, starting from a given tree node index.
func (a *AABBTree) fixUpwardsTree(treeNodeIndex uint) {
	for treeNodeIndex != AABBNullNode {
		treeNode := a.nodes[treeNodeIndex]

		// every node should be a parent
		//assert(treeNode.leftNodeIndex != AABBNullNode && treeNode.rightNodeIndex != AABBNullNode)

		// fix height and area
		leftNode := a.nodes[treeNode.leftNodeIndex]
		rightNode := a.nodes[treeNode.rightNodeIndex]
		treeNode.aabb = leftNode.aabb.merge(rightNode.aabb)

		treeNodeIndex = treeNode.parentNodeIndex
	}
}
