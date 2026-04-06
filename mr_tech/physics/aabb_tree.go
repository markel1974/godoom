package physics

const aabbMargin = 4.0 // aabbMargin defines the expansion margin for Axis-Aligned Bounding Boxes (AABBs) to accommodate movement and avoid frequent updates.

// AABBTree is a spatial data structure optimized for efficient querying and management of Axis-Aligned Bounding Boxes (AABBs).
type AABBTree struct {
	objectNodeIndexMap map[IAABB]uint
	nodes              []*AABBNode
	rootNodeIndex      uint
	allocatedNodeCount uint
	nextFreeNodeIndex  uint
	nodeCapacity       uint
	growthSize         uint
	stack              []uint
}

// NewAABBTree initializes and returns a new instance of AABBTree with a specified initial node capacity.
func NewAABBTree(initialSize uint) *AABBTree {
	if initialSize == 0 {
		initialSize = 1
	}
	t := &AABBTree{
		rootNodeIndex:      AABBNullNode,
		allocatedNodeCount: 0,
		nextFreeNodeIndex:  0,
		nodeCapacity:       initialSize,
		growthSize:         initialSize,
		nodes:              make([]*AABBNode, initialSize),
		objectNodeIndexMap: make(map[IAABB]uint),
		stack:              make([]uint, 0, 256),
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

// GetRoot returns the root AABB of the tree along with a boolean indicating its existence.
func (a *AABBTree) GetRoot() (*AABB, bool) {
	if a.rootNodeIndex == AABBNullNode {
		return nil, false
	}
	return a.nodes[a.rootNodeIndex].aabb, true
}

// Nodes returns a map linking objects implementing IAABB to their corresponding node indices in the AABB tree.
func (a *AABBTree) Nodes() map[IAABB]uint {
	return a.objectNodeIndexMap
}

// InsertObject inserts a new object into the AABBTree, updates its AABB, and associates it with a node index.
func (a *AABBTree) InsertObject(object IAABB) {
	nodeIndex, node := a.allocateNode()
	node.object = object
	node.aabb = object.GetAABB().Expand(aabbMargin)

	a.insertLeaf(nodeIndex)
	a.objectNodeIndexMap[object] = nodeIndex
}

// QueryFrustum traverses the tree and invokes the callback for each object whose AABB intersects with the specified frustum.
func (a *AABBTree) QueryFrustum(frustum *Frustum, callback func(object IAABB) bool) {
	a.stack = a.stack[:0]
	a.stack = append(a.stack, a.rootNodeIndex)

	for len(a.stack) > 0 {
		lastIdx := len(a.stack) - 1
		nodeIndex := a.stack[lastIdx]
		a.stack = a.stack[:lastIdx]

		if nodeIndex == AABBNullNode {
			continue
		}

		node := a.nodes[nodeIndex]

		// Test di intersezione tra l'AABB del nodo e il Frustum
		if node.aabb.IntersectFrustum(frustum) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				// Il nodo è visibile o intersecato: continuiamo a scendere nei rami
				a.stack = append(a.stack, node.leftNodeIndex)
				a.stack = append(a.stack, node.rightNodeIndex)
			}
		}
		// Altrimenti, se il nodo è fuori dal Frustum, l'intero ramo viene scartato! (Frustum Culling)
	}
}

// RemoveObject removes the specified object from the AABBTree if it exists.
func (a *AABBTree) RemoveObject(object IAABB) {
	if nodeIndex, ok := a.objectNodeIndexMap[object]; ok {
		a.removeLeaf(nodeIndex)
		a.deallocateNode(nodeIndex)
		delete(a.objectNodeIndexMap, object)
	}
}

// UpdateObject updates the position of an object in the tree by modifying its AABB and repositioning it if necessary.
func (a *AABBTree) UpdateObject(object IAABB) {
	if nodeIndex, ok := a.objectNodeIndexMap[object]; ok {
		a.updateLeaf(nodeIndex, object.GetAABB())
	}
}

// QueryOverlaps identifies and returns all objects in the tree whose AABBs overlap with the given object's AABB.
func (a *AABBTree) QueryOverlaps(object IAABB, callback func(object IAABB) bool) {
	testAabb := object.GetAABB()
	a.stack = a.stack[:0]
	a.stack = append(a.stack, a.rootNodeIndex)

	for len(a.stack) > 0 {
		// Pop (LIFO) dall'ultima posizione della slice
		lastIdx := len(a.stack) - 1
		nodeIndex := a.stack[lastIdx]
		a.stack = a.stack[:lastIdx]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		if node.aabb.Overlaps(testAabb) {
			if node.IsLeaf() {
				if node.object != object {
					if callback(node.object) {
						break
					}
				}
			} else {
				// Push dei nodi figli
				a.stack = append(a.stack, node.leftNodeIndex)
				a.stack = append(a.stack, node.rightNodeIndex)
			}
		}
	}
}

// QueryPoint searches the tree for objects whose AABBs contain the given point, with tolerance defined by epsilon.
func (a *AABBTree) QueryPoint(px, py float64, callback func(object IAABB) bool) {
	a.stack = a.stack[:0]
	a.stack = append(a.stack, a.rootNodeIndex)

	for len(a.stack) > 0 {
		lastIdx := len(a.stack) - 1
		nodeIndex := a.stack[lastIdx]
		a.stack = a.stack[:lastIdx]

		if nodeIndex == AABBNullNode {
			continue
		}

		node := a.nodes[nodeIndex]

		if node.aabb.QueryPoint(px, py) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				a.stack = append(a.stack, node.leftNodeIndex)
				a.stack = append(a.stack, node.rightNodeIndex)
			}
		}
	}
}

// allocateNode manages the allocation of a new node in the tree, resizing the node array if capacity is exceeded.
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

// deallocateNode removes a node from the tree, marking it as free and linking it to the free list for reuse.
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

// insertLeaf inserts a new leaf node with the given index into the AABB tree, adjusting the tree structure as needed.
func (a *AABBTree) insertLeaf(leafNodeIndex uint) {
	// if the tree is empty, then we make the root the leaf
	if a.rootNodeIndex == AABBNullNode {
		a.rootNodeIndex = leafNodeIndex
		return
	}

	// search for the best place to put the new leaf in the tree;
	// we use surface area and depth as search heuristics
	treeNodeIndex := a.rootNodeIndex
	leafNode := a.nodes[leafNodeIndex]

	for !a.nodes[treeNodeIndex].IsLeaf() {
		// because of the test in the while loop above, we know we are never a leaf inside it
		treeNode := a.nodes[treeNodeIndex]
		leftNodeIndex := treeNode.leftNodeIndex
		rightNodeIndex := treeNode.rightNodeIndex
		leftNode := a.nodes[leftNodeIndex]
		rightNode := a.nodes[rightNodeIndex]

		combinedAabb := treeNode.aabb.Merge(leafNode.aabb)

		newParentNodeCost := 2.0 * combinedAabb.surfaceArea
		minimumPushDownCost := 2.0 * (combinedAabb.surfaceArea - treeNode.aabb.surfaceArea)

		// use the costs to figure out whether to create a new parent here or descend
		var costLeft float64
		var costRight float64
		if leftNode.IsLeaf() {
			costLeft = leafNode.aabb.Merge(leftNode.aabb).surfaceArea + minimumPushDownCost
		} else {
			newLeftAabb := leafNode.aabb.Merge(leftNode.aabb)
			costLeft = (newLeftAabb.surfaceArea - leftNode.aabb.surfaceArea) + minimumPushDownCost
		}
		if rightNode.IsLeaf() {
			costRight = leafNode.aabb.Merge(rightNode.aabb).surfaceArea + minimumPushDownCost
		} else {
			newRightAabb := leafNode.aabb.Merge(rightNode.aabb)
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
	newParent.aabb = leafNode.aabb.Merge(leafSibling.aabb) // the new parents aabb is the leaf aabb combined with it's siblings aabb
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

// removeLeaf removes a leaf node from the AABBTree by reassigning its sibling and restructuring the tree as necessary.
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
		// if we have no grandparent, then the parent is the root, and so our sibling becomes the root and has it's parent removed
		a.rootNodeIndex = siblingNodeIndex
		siblingNode.parentNodeIndex = AABBNullNode
		a.deallocateNode(parentNodeIndex)
	}
	leafNode.parentNodeIndex = AABBNullNode
}

// updateLeaf updates the AABB of a leaf node if the new AABB does not fit within the current margin.
// It removes the leaf, updates its AABB, and reinserts it into the tree.
func (a *AABBTree) updateLeaf(leafNodeIndex uint, newAABB *AABB) {
	node := a.nodes[leafNodeIndex]
	// Branch Prediction amichevole: nel 99% dei frame le entità restano nel loro Fat Margin
	if node.aabb.Contains(newAABB) {
		return
	}
	// Boundary violato: mutazione dell'albero necessaria
	a.removeLeaf(leafNodeIndex)
	node.aabb = newAABB.Expand(aabbMargin) // Rigenera il Fat Margin centrato sulla nuova posizione
	a.insertLeaf(leafNodeIndex)
}

// fixUpwardsTree recalculates the bounding volumes and heights of nodes moving upwards in the AABBTree from a given node index.
func (a *AABBTree) fixUpwardsTree(treeNodeIndex uint) {
	for treeNodeIndex != AABBNullNode {
		treeNode := a.nodes[treeNodeIndex]

		// fix height and area
		leftNode := a.nodes[treeNode.leftNodeIndex]
		rightNode := a.nodes[treeNode.rightNodeIndex]
		treeNode.aabb = leftNode.aabb.Merge(rightNode.aabb)

		treeNodeIndex = treeNode.parentNodeIndex
	}
}
