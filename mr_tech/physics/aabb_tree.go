package physics

// AABBTree is a spatial data structure optimized for efficient querying and management of Axis-Aligned Bounding Boxes (AABBs).
type AABBTree struct {
	objectNodeIndexMap map[IAABB]uint
	nodes              []*AABBNode
	rootNodeIndex      uint
	allocatedNodeCount uint
	nextFreeNodeIndex  uint
	nodeCapacity       uint
	growthSize         uint
	margin             float64
	stack              []uint
}

// NewAABBTree initializes and returns a new instance of AABBTree with a specified initial node capacity.
func NewAABBTree(initialSize uint, margin float64) *AABBTree {
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
		stack:              make([]uint, initialSize),
		margin:             margin,
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

// Clear resets the AABBTree to its initial empty state without deallocating the underlying structures.
// It rebuilds the internal free-node list and clears the object map, making it ready for reuse.
func (a *AABBTree) Clear() {
	a.rootNodeIndex = AABBNullNode
	a.allocatedNodeCount = 0
	a.nextFreeNodeIndex = 0
	var nodeIndex uint
	for nodeIndex = 0; nodeIndex < a.nodeCapacity; nodeIndex++ {
		a.nodes[nodeIndex].object = nil
		a.nodes[nodeIndex].parentNodeIndex = AABBNullNode
		a.nodes[nodeIndex].leftNodeIndex = AABBNullNode
		a.nodes[nodeIndex].rightNodeIndex = AABBNullNode
		a.nodes[nodeIndex].nextNodeIndex = nodeIndex + 1
	}
	if a.nodeCapacity > 0 {
		a.nodes[a.nodeCapacity-1].nextNodeIndex = AABBNullNode
	}
	a.objectNodeIndexMap = make(map[IAABB]uint)
}

// InsertObject inserts a new object into the AABBTree, updates its AABB, and associates it with a node index.
func (a *AABBTree) InsertObject(object IAABB) {
	nodeIndex, node := a.allocateNode()
	node.object = object
	node.aabb.ExpandInPlace(object.GetAABB(), a.margin)
	a.insertLeaf(nodeIndex)
	a.objectNodeIndexMap[object] = nodeIndex
	//fmt.Println("INSERTING OBJECT", object.GetAABB().GetMinZ(), object.GetAABB().GetMaxZ())
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
		node := a.nodes[nodeIndex]
		// Branch Prediction amichevole: se è nel Fat Margin, non facciamo nulla
		if node.aabb.Contains(object.GetAABB()) {
			return
		}
		// Se esce dal margine, rimuoviamo e reinseriamo in modo pulito
		a.RemoveObject(object)
		a.InsertObject(object)
	}
}

// QueryOverlaps identifies and returns all objects in the tree whose AABBs overlap with the given object's AABB.
func (a *AABBTree) QueryOverlaps(object IAABB, callback func(object IAABB) bool) {
	testAabb := object.GetAABB()
	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
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
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				// Push dei nodi figli
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryPoint2d searches the tree for objects whose AABBs contain the given point, with tolerance defined by epsilon.
func (a *AABBTree) QueryPoint2d(px, py float64, callback func(object IAABB) bool) {
	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		if node.aabb.ContainsPoint2d(px, py) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryPoint3d searches the tree for objects whose AABBs contain the given 3D point.
func (a *AABBTree) QueryPoint3d(px, py, pz float64, callback func(object IAABB) bool) {
	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		if node.aabb.ContainsPoint3d(px, py, pz) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryFrustum traverses the tree and invokes the callback for each object whose AABB intersects with the specified frustum.
func (a *AABBTree) QueryFrustum(frustum *Frustum, callback func(object IAABB) bool) {
	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		if node.aabb.IntersectFrustum(frustum) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
		// Altrimenti, se il nodo è fuori dal Frustum, l'intero ramo viene scartato! (Frustum Culling)
	}
}

// QueryMultiFrustum traverses the AABB tree to find objects intersecting any of the two given frustums, invoking the callback per match.
func (a *AABBTree) QueryMultiFrustum(f1, f2 *Frustum, callback func(object IAABB) bool) {
	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		// Short-circuit evaluation: previene test doppi e invocazioni duplicate del callback
		if node.aabb.IntersectFrustum(f1) || node.aabb.IntersectFrustum(f2) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryRay performs a raycast query against the AABB tree, calling the callback for each intersected object within range.
// oX, oY, oZ represent the origin of the ray.
// dirX, dirY, dirZ represent the direction vector of the ray.
// maxDistance specifies the maximum distance for the ray to query.
// callback is invoked with each intersected object and its distance, and may update the maxDistance dynamically.
func (a *AABBTree) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object IAABB, distance float64) (float64, bool)) {
	// Pre-calcolo delle inverse per la vettorizzazione dello Slab Method
	invDirX := 1.0 / dirX
	invDirY := 1.0 / dirY
	invDirZ := 1.0 / dirZ

	stackIndex := 0
	a.stack[stackIndex] = a.rootNodeIndex
	stackIndex++

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := a.stack[stackIndex]
		if nodeIndex == AABBNullNode {
			continue
		}
		node := a.nodes[nodeIndex]
		tMin, hit := node.aabb.IntersectRay(oX, oY, oZ, invDirX, invDirY, invDirZ)
		// Ray Culling: scartiamo l'intero ramo se è più lontano del nostro limite attuale
		if hit && tMin <= maxDistance {
			if node.IsLeaf() {
				// Segnaliamo l'oggetto e permettiamo alla logica di restringere il raggio
				if newMax, ok := callback(node.object, tMin); ok {
					if newMax < maxDistance {
						maxDistance = newMax
					}
				}
			} else {
				if (stackIndex + 2) > len(a.stack) {
					a.stackGrow()
				}
				a.stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				a.stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// allocateNode manages the allocation of a new node in the tree, resizing the node array if capacity is exceeded.
func (a *AABBTree) allocateNode() (uint, *AABBNode) {
	if a.nextFreeNodeIndex == AABBNullNode {
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
		deallocatedNode.object = nil
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

		combinedAabbSurfaceArea := treeNode.aabb.GetSurfaceAreaMerged(leafNode.aabb)

		newParentNodeCost := 2.0 * combinedAabbSurfaceArea
		minimumPushDownCost := 2.0 * (combinedAabbSurfaceArea - treeNode.aabb.GetSurfaceArea())

		// use the costs to figure out whether to create a new parent here or descend
		var costLeft float64
		var costRight float64
		leftMergeSurfaceArea := leafNode.aabb.GetSurfaceAreaMerged(leftNode.aabb)
		rightMergeSurfaceArea := leafNode.aabb.GetSurfaceAreaMerged(rightNode.aabb)
		if leftNode.IsLeaf() {
			costLeft = leftMergeSurfaceArea + minimumPushDownCost
		} else {
			costLeft = (leftMergeSurfaceArea - leftNode.aabb.GetSurfaceArea()) + minimumPushDownCost
		}
		if rightNode.IsLeaf() {
			costRight = rightMergeSurfaceArea + minimumPushDownCost
		} else {
			costRight = (rightMergeSurfaceArea - rightNode.aabb.GetSurfaceArea()) + minimumPushDownCost
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
	//the new parents aabb is the leaf aabb combined with it's siblings aabb
	//newParent.aabb = NewAABBMerge(leafNode.aabb, leafSibling.aabb)
	newParent.aabb.MergeInPlace(leafNode.aabb, leafSibling.aabb)
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
	//node.aabb = NewAABBExpand(newAABB, a.margin)
	node.aabb.ExpandInPlace(newAABB, a.margin)
	a.insertLeaf(leafNodeIndex)
}

// fixUpwardsTree recalculates the bounding volumes and heights of nodes moving upwards in the AABBTree from a given node index.
func (a *AABBTree) fixUpwardsTree(treeNodeIndex uint) {
	for treeNodeIndex != AABBNullNode {
		treeNode := a.nodes[treeNodeIndex]

		// fix height and area
		leftNode := a.nodes[treeNode.leftNodeIndex]
		rightNode := a.nodes[treeNode.rightNodeIndex]
		treeNode.aabb.MergeInPlace(leftNode.aabb, rightNode.aabb)

		treeNodeIndex = treeNode.parentNodeIndex
	}
}

// stackGrow doubles the size of the internal stack and copies existing elements into the expanded stack.
func (a *AABBTree) stackGrow() {
	newStack := make([]uint, len(a.stack)*2)
	copy(newStack, a.stack)
	a.stack = newStack
}
