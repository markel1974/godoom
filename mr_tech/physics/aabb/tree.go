package aabb

import "math/bits"

// treeLocalStackCap fits typical BVH traversal depths cleanly on the caller stack without heap allocations.
const treeLocalStackCap = 128

// Tree is a spatial data structure optimized for efficient querying and management of Axes-Aligned Bounding Boxes (AABBs).
type Tree struct {
	nodes              []*Node
	rootNodeIndex      uint
	allocatedNodeCount uint
	nextFreeNodeIndex  uint
	nodeCapacity       uint
	growthSize         uint
	margin             float64
	version            uint64
}

// Version returns a monotonic counter incremented on spatial mutations (insert, remove, update, clear).
func (a *Tree) Version() uint64 {
	return a.version
}

// NewTree initializes and returns a new instance of Tree with a specified initial node capacity.
func NewTree(initialSize uint, margin float64) *Tree {
	if initialSize == 0 {
		initialSize = 1
	}
	t := &Tree{
		rootNodeIndex:      NullNode,
		allocatedNodeCount: 0,
		nextFreeNodeIndex:  0,
		nodeCapacity:       initialSize,
		growthSize:         initialSize,
		nodes:              make([]*Node, initialSize),
		margin:             margin,
	}

	nodeSlab := make([]Node, initialSize)
	aabbSlab := make([]AABB, initialSize)
	for nodeIndex := uint(0); nodeIndex < initialSize; nodeIndex++ {
		node := &nodeSlab[nodeIndex]
		aabbSlab[nodeIndex] = MakeAABB()
		node.aabb = &aabbSlab[nodeIndex]
		node.parentNodeIndex = NullNode
		node.leftNodeIndex = NullNode
		node.rightNodeIndex = NullNode
		node.nextNodeIndex = nodeIndex + 1
		t.nodes[nodeIndex] = node
	}
	nodeSlab[initialSize-1].nextNodeIndex = NullNode

	return t
}

// Capacity returns the maximum number of nodes the tree can hold.
func (a *Tree) Capacity() uint {
	return a.nodeCapacity
}

// EnsureCapacity expands the internal node pool so at least minCap nodes fit without incremental reallocations.
func (a *Tree) EnsureCapacity(minCap uint) {
	if minCap <= a.nodeCapacity {
		return
	}
	newCap := a.nodeCapacity * 2
	if newCap < minCap {
		newCap = minCap
	}
	a.growTo(newCap)
}

// GetRoot returns the root AABB of the tree along with a boolean indicating its existence.
func (a *Tree) GetRoot() (*AABB, bool) {
	if a.rootNodeIndex == NullNode {
		return nil, false
	}
	return a.nodes[a.rootNodeIndex].aabb, true
}

// Clear resets the Tree to its initial empty state without deallocating the underlying structures.
// It rebuilds the internal free-node list and clears the object map, making it ready for reuse.
func (a *Tree) Clear() {
	a.rootNodeIndex = NullNode
	a.allocatedNodeCount = 0
	a.nextFreeNodeIndex = 0
	var nodeIndex uint
	for nodeIndex = 0; nodeIndex < a.nodeCapacity; nodeIndex++ {
		if obj := a.nodes[nodeIndex].object; obj != nil {
			if ab := obj.GetAABB(); ab != nil {
				ab.SetIndex(-1)
			}
		}
		a.nodes[nodeIndex].object = nil
		a.nodes[nodeIndex].parentNodeIndex = NullNode
		a.nodes[nodeIndex].leftNodeIndex = NullNode
		a.nodes[nodeIndex].rightNodeIndex = NullNode
		a.nodes[nodeIndex].nextNodeIndex = nodeIndex + 1
	}
	if a.nodeCapacity > 0 {
		a.nodes[a.nodeCapacity-1].nextNodeIndex = NullNode
	}
	a.version++
	//a.objectNodeIndexMap = make(map[IObject]uint)
}

// IsEmpty reports whether the tree currently holds zero allocated nodes.
func (a *Tree) IsEmpty() bool {
	return a.allocatedNodeCount == 0
}

// ContainsObject reports in O(1) whether object currently maps to an active node in this tree.
func (a *Tree) ContainsObject(object IObject) bool {
	if object == nil {
		return false
	}
	ab := object.GetAABB()
	if ab == nil {
		return false
	}
	nodeIdx := ab.GetIndex()
	if nodeIdx < 0 || nodeIdx >= int64(len(a.nodes)) {
		return false
	}
	return a.nodes[uint(nodeIdx)].object == object
}

// InsertObject inserts a new object into the Tree, updates its AABB, and associates it with a node index.
func (a *Tree) InsertObject(object IObject) {
	if object == nil {
		return
	}
	nodeIndex, node := a.allocateNode()
	node.object = object
	node.aabb.ExpandInPlace(object.GetAABB(), a.margin)
	a.insertLeaf(nodeIndex)
	object.GetAABB().SetIndex(int64(nodeIndex))
	a.version++
}

// RemoveObject removes the specified object from the Tree if it exists.
func (a *Tree) RemoveObject(object IObject) {
	if object == nil {
		return
	}
	aabb := object.GetAABB()
	if aabb == nil {
		return
	}
	nodeIdx := aabb.GetIndex()
	aabb.SetIndex(-1)
	if nodeIdx < 0 || nodeIdx >= int64(len(a.nodes)) {
		return
	}
	nodeIndex := uint(nodeIdx)
	node := a.nodes[nodeIndex]
	if node.object != object {
		return
	}
	a.removeLeaf(nodeIndex)
	a.deallocateNode(nodeIndex)
	a.version++
}

// UpdateObject updates the position of an object in the tree by modifying its AABB and repositioning it if necessary.
func (a *Tree) UpdateObject(object IObject) {
	if object == nil {
		return
	}
	aabb := object.GetAABB()
	if aabb == nil {
		return
	}
	nodeIdx := aabb.GetIndex()
	if nodeIdx < 0 || nodeIdx >= int64(len(a.nodes)) {
		return
	}
	nodeIndex := uint(nodeIdx)
	node := a.nodes[nodeIndex]
	if node.object != object {
		return
	}
	a.version++
	// Branch Prediction amichevole: se è nel Fat Margin e non si è ristretto eccessivamente, non facciamo nulla
	if node.aabb.Contains(aabb) {
		maxExcess := a.margin * 4.0
		if (node.aabb.maxX-node.aabb.minX)-(aabb.maxX-aabb.minX) <= maxExcess &&
			(node.aabb.maxY-node.aabb.minY)-(aabb.maxY-aabb.minY) <= maxExcess &&
			(node.aabb.maxZ-node.aabb.minZ)-(aabb.maxZ-aabb.minZ) <= maxExcess {
			return
		}
	}
	// Riaggiorna la foglia direttamente senza deallocare/riallocare il nodo
	a.updateLeaf(nodeIndex, aabb)
}

// growLocalTreeStack expands a query traversal stack beyond its stack-allocated capacity.
func growLocalTreeStack(stack []uint) []uint {
	next := make([]uint, len(stack)*2)
	copy(next, stack)
	return next
}

// growLocalTreeStackMask expands both traversal stack and per-entry frustum mask beyond stack capacity.
func growLocalTreeStackMask(stack []uint, mask []uint32) ([]uint, []uint32) {
	newCap := len(stack) * 2
	nextStack := make([]uint, newCap)
	copy(nextStack, stack)
	nextMask := make([]uint32, newCap)
	copy(nextMask, mask)
	return nextStack, nextMask
}

// QueryOverlaps identifies and returns all objects in the tree whose AABBs overlap with the given object's AABB.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryOverlaps(object IObject, callback func(object IObject) bool) {
	if a.rootNodeIndex == NullNode {
		return
	}
	testAabb := object.GetAABB()
	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	stack[0] = a.rootNodeIndex
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		if nodeIndex == NullNode {
			continue
		}
		node := nodes[nodeIndex]
		if node.aabb.Overlaps(testAabb) {
			if node.IsLeaf() {
				if node.object != object {
					if callback(node.object) {
						break
					}
				}
			} else {
				if (stackIndex + 2) > len(stack) {
					stack = growLocalTreeStack(stack)
				}
				// Push dei nodi figli
				stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryPoint2d searches the tree for objects whose AABBs contain the given point, with tolerance defined by epsilon.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryPoint2d(px, py float64, callback func(object IObject) bool) {
	if a.rootNodeIndex == NullNode {
		return
	}
	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	stack[0] = a.rootNodeIndex
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		if nodeIndex == NullNode {
			continue
		}
		node := nodes[nodeIndex]
		if node.aabb.ContainsPoint2d(px, py) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(stack) {
					stack = growLocalTreeStack(stack)
				}
				stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryPoint3d searches the tree for objects whose AABBs contain the given 3D point.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryPoint3d(px, py, pz float64, callback func(object IObject) bool) {
	if a.rootNodeIndex == NullNode {
		return
	}
	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	stack[0] = a.rootNodeIndex
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		if nodeIndex == NullNode {
			continue
		}
		node := nodes[nodeIndex]
		if node.aabb.ContainsPoint3d(px, py, pz) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(stack) {
					stack = growLocalTreeStack(stack)
				}
				stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

// QueryFrustum traverses the tree and invokes the callback for each object whose AABB intersects with the specified frustum.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryFrustum(frustum *Frustum, callback func(object IObject) bool) {
	if a.rootNodeIndex == NullNode || frustum == nil {
		return
	}
	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	stack[0] = a.rootNodeIndex
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		if nodeIndex == NullNode {
			continue
		}
		node := nodes[nodeIndex]
		box := node.aabb
		if intersectFrustumBounds(box.minX, box.minY, box.minZ, box.maxX, box.maxY, box.maxZ, frustum) {
			if node.IsLeaf() {
				if callback(node.object) {
					break
				}
			} else {
				if (stackIndex + 2) > len(stack) {
					stack = growLocalTreeStack(stack)
				}
				stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
		// Altrimenti, se il nodo è fuori dal Frustum, l'intero ramo viene scartato! (Frustum Culling)
	}
}

// QueryFrustumsMask traverses the tree once against all frustums enabled in activeMask,
// pruning subtrees per frustum bit and invoking callback once per leaf with its matching bitmask.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryFrustumsMask(frustums *Frustums, activeMask uint32, callback func(object IObject, leafMask uint32) bool) {
	if a.rootNodeIndex == NullNode || activeMask == 0 {
		return
	}
	fLen := len(frustums.Container)
	if fLen == 0 {
		return
	}
	if fLen < 32 {
		activeMask &= (uint32(1) << fLen) - 1
		if activeMask == 0 {
			return
		}
	}
	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	if activeMask&(activeMask-1) == 0 {
		f := frustums.Container[bits.TrailingZeros32(activeMask)]
		stack[0] = a.rootNodeIndex
		stackIndex := 1
		for stackIndex > 0 {
			stackIndex--
			nodeIndex := stack[stackIndex]
			if nodeIndex == NullNode {
				continue
			}
			node := nodes[nodeIndex]
			box := node.aabb
			if intersectFrustumBounds(box.minX, box.minY, box.minZ, box.maxX, box.maxY, box.maxZ, f) {
				if node.IsLeaf() {
					if callback(node.object, activeMask) {
						break
					}
				} else {
					if (stackIndex + 2) > len(stack) {
						stack = growLocalTreeStack(stack)
					}
					stack[stackIndex] = node.leftNodeIndex
					stackIndex++
					stack[stackIndex] = node.rightNodeIndex
					stackIndex++
				}
			}
		}
		return
	}
	var localMask [treeLocalStackCap]uint32
	stackMask := localMask[:]

	stack[0] = a.rootNodeIndex
	stackMask[0] = activeMask
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		mask := stackMask[stackIndex]
		if nodeIndex == NullNode || mask == 0 {
			continue
		}
		node := nodes[nodeIndex]
		box := node.aabb
		minX, minY, minZ := box.minX, box.minY, box.minZ
		maxX, maxY, maxZ := box.maxX, box.maxY, box.maxZ

		var hitMask uint32
		rem := mask
		for rem != 0 {
			bit := bits.TrailingZeros32(rem)
			rem &^= 1 << bit
			if f := frustums.Container[bit]; f != nil && intersectFrustumBounds(minX, minY, minZ, maxX, maxY, maxZ, f) {
				hitMask |= 1 << bit
			}
		}
		if hitMask == 0 {
			continue
		}
		if node.IsLeaf() {
			if callback(node.object, hitMask) {
				break
			}
		} else {
			if (stackIndex + 2) > len(stack) {
				stack, stackMask = growLocalTreeStackMask(stack, stackMask)
			}
			stack[stackIndex] = node.leftNodeIndex
			stackMask[stackIndex] = hitMask
			stackIndex++
			stack[stackIndex] = node.rightNodeIndex
			stackMask[stackIndex] = hitMask
			stackIndex++
		}
	}
}

// QueryRay performs a raycast query against the AABB tree, calling the callback for each intersected object within range.
// oX, oY, oZ represent the origin of the ray.
// dirX, dirY, dirZ represent the direction vector of the ray.
// maxDistance specifies the maximum distance for the ray to query.
// callback is invoked with each intersected object and its distance, and may update the maxDistance dynamically.
// It is read-only and safe for concurrent calls on the same Tree.
func (a *Tree) QueryRay(oX, oY, oZ, dirX, dirY, dirZ float64, maxDistance float64, callback func(object IObject, distance float64) (float64, bool)) {
	if a.rootNodeIndex == NullNode {
		return
	}
	// Pre-calcolo delle inverse per la vettorizzazione dello Slab Method
	invDirX := 1.0 / dirX
	invDirY := 1.0 / dirY
	invDirZ := 1.0 / dirZ

	nodes := a.nodes
	var localStack [treeLocalStackCap]uint
	stack := localStack[:]
	stack[0] = a.rootNodeIndex
	stackIndex := 1

	for stackIndex > 0 {
		stackIndex--
		nodeIndex := stack[stackIndex]
		if nodeIndex == NullNode {
			continue
		}
		node := nodes[nodeIndex]
		tMin, hit := node.aabb.IntersectRay(oX, oY, oZ, invDirX, invDirY, invDirZ)
		// Ray Culling: scartiamo l'intero ramo se è più lontano del nostro limite attuale
		if hit && tMin <= maxDistance {
			if node.IsLeaf() {
				// Narrow-phase test: verifichiamo l'intersezione esatta contro l'AABB tight dell'oggetto
				tightAABB := node.object.GetAABB()
				if tightAABB != nil {
					tMinTight, hitTight := tightAABB.IntersectRay(oX, oY, oZ, invDirX, invDirY, invDirZ)
					if hitTight && tMinTight <= maxDistance {
						if newMax, ok := callback(node.object, tMinTight); ok {
							if newMax < maxDistance {
								maxDistance = newMax
							}
						}
					}
				}
			} else {
				if (stackIndex + 2) > len(stack) {
					stack = growLocalTreeStack(stack)
				}
				stack[stackIndex] = node.leftNodeIndex
				stackIndex++
				stack[stackIndex] = node.rightNodeIndex
				stackIndex++
			}
		}
	}
}

func (a *Tree) growTo(newCap uint) {
	if newCap <= a.nodeCapacity {
		return
	}
	prevCap := a.nodeCapacity
	added := newCap - prevCap
	nodes := make([]*Node, newCap)
	copy(nodes, a.nodes)

	nodeSlab := make([]Node, added)
	aabbSlab := make([]AABB, added)
	for i := uint(0); i < added; i++ {
		nodeIdx := prevCap + i
		node := &nodeSlab[i]
		aabbSlab[i] = MakeAABB()
		node.aabb = &aabbSlab[i]
		node.parentNodeIndex = NullNode
		node.leftNodeIndex = NullNode
		node.rightNodeIndex = NullNode
		node.nextNodeIndex = nodeIdx + 1
		nodes[nodeIdx] = node
	}
	nodeSlab[added-1].nextNodeIndex = a.nextFreeNodeIndex
	a.nodes = nodes
	a.nodeCapacity = newCap
	a.nextFreeNodeIndex = prevCap
}

// allocateNode manages the allocation of a new node in the tree, resizing the node array if capacity is exceeded.
func (a *Tree) allocateNode() (uint, *Node) {
	if a.nextFreeNodeIndex == NullNode {
		grow := a.nodeCapacity
		if grow < a.growthSize {
			grow = a.growthSize
		}
		if grow < 16 {
			grow = 16
		}
		a.growTo(a.nodeCapacity + grow)
	}

	nodeIndex := a.nextFreeNodeIndex
	allocatedNode := a.nodes[nodeIndex]
	allocatedNode.parentNodeIndex = NullNode
	allocatedNode.leftNodeIndex = NullNode
	allocatedNode.rightNodeIndex = NullNode
	a.nextFreeNodeIndex = allocatedNode.nextNodeIndex
	a.allocatedNodeCount++

	return nodeIndex, allocatedNode
}

// deallocateNode removes a node from the tree, marking it as free and linking it to the free list for reuse.
func (a *Tree) deallocateNode(nodeIndex uint) {
	if len(a.nodes) == 0 {
		return
	}

	if nodeIndex < uint(len(a.nodes)) {
		deallocatedNode := a.nodes[nodeIndex]
		deallocatedNode.object = nil
		deallocatedNode.nextNodeIndex = a.nextFreeNodeIndex
		a.nextFreeNodeIndex = nodeIndex
		a.allocatedNodeCount--
	}
}

// insertLeaf inserts a new leaf node with the given index into the AABB tree, adjusting the tree structure as needed.
func (a *Tree) insertLeaf(leafNodeIndex uint) {
	// if the tree is empty, then we make the root the leaf
	if a.rootNodeIndex == NullNode {
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

	if oldParentIndex == NullNode {
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

// removeLeaf removes a leaf node from the Tree by reassigning its sibling and restructuring the tree as necessary.
func (a *Tree) removeLeaf(leafNodeIndex uint) {
	// if the leaf is the root then we can just clear the root pointer and return
	if leafNodeIndex == a.rootNodeIndex {
		a.rootNodeIndex = NullNode
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
	if grandParentNodeIndex != NullNode {
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
		siblingNode.parentNodeIndex = NullNode
		a.deallocateNode(parentNodeIndex)
	}
	leafNode.parentNodeIndex = NullNode
}

// updateLeaf removes the leaf, updates its expanded AABB, and reinserts it into the tree.
func (a *Tree) updateLeaf(leafNodeIndex uint, newAABB *AABB) {
	node := a.nodes[leafNodeIndex]
	a.removeLeaf(leafNodeIndex)
	node.aabb.ExpandInPlace(newAABB, a.margin)
	a.insertLeaf(leafNodeIndex)
}

// fixUpwardsTree recalculates the bounding volumes and heights of nodes moving upwards in the Tree from a given node index.
func (a *Tree) fixUpwardsTree(treeNodeIndex uint) {
	for treeNodeIndex != NullNode {
		treeNode := a.nodes[treeNodeIndex]

		// fix height and area
		leftNode := a.nodes[treeNode.leftNodeIndex]
		rightNode := a.nodes[treeNode.rightNodeIndex]
		treeNode.aabb.MergeInPlace(leftNode.aabb, rightNode.aabb)

		treeNodeIndex = treeNode.parentNodeIndex
	}
}
