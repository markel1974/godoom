package aabb

import (
	"math"
	"testing"
)

type testTreeObj struct {
	box *AABB
	id  int
}

func (o *testTreeObj) GetAABB() *AABB {
	return o.box
}

func makeOrthoVP(minX, maxX, minY, maxY, minZ, maxZ float32) [16]float32 {
	rl := maxX - minX
	tb := maxY - minY
	fn := maxZ - minZ
	return [16]float32{
		2 / rl, 0, 0, 0,
		0, 2 / tb, 0, 0,
		0, 0, -2 / fn, 0,
		-(maxX + minX) / rl, -(maxY + minY) / tb, -(maxZ + minZ) / fn, 1,
	}
}

func makePerspVP(fovRad, aspect, nearZ, farZ, eyeX, eyeY, eyeZ float64) [16]float32 {
	// Camera at (eyeX, eyeY, eyeZ) looking down +Z with up = +Y.
	f := float32(1.0 / math.Tan(fovRad*0.5))
	a := float32(aspect)
	n := float32(nearZ)
	far := float32(farZ)
	tx := -float32(eyeX)
	ty := -float32(eyeY)
	tz := -float32(eyeZ)
	// View: flip +Z to forward (-Z in GL clip): z_view = -(z - eyeZ)
	// Column-major V: col0=(1,0,0,0), col1=(0,1,0,0), col2=(0,0,-1,0), col3=(tx,ty,-tz,1) -> z_view = -(z - eyeZ) = eyeZ - z.
	p00 := f / a
	p11 := f
	p22 := -(far + n) / (far - n)
	p32 := -(2 * far * n) / (far - n)
	return [16]float32{
		p00, 0, 0, 0,
		0, p11, 0, 0,
		0, 0, -p22, 1,
		p00 * tx, p11 * ty, p22*tz + p32, -tz,
	}
}

func TestFrustumBoundsAndQueryFrustumsMaskEquivalence(t *testing.T) {
	tree := NewTree(32, 0.5)
	objs := make([]*testTreeObj, 0, 125)
	id := 1
	for ix := -2; ix <= 2; ix++ {
		for iy := -2; iy <= 2; iy++ {
			for iz := -2; iz <= 2; iz++ {
				cx := float64(ix) * 40
				cy := float64(iy) * 40
				cz := float64(iz) * 40
				box := NewAABB()
				box.Rebuild(cx-6, cy-6, cz-6, cx+6, cy+6, cz+6)
				obj := &testTreeObj{box: box, id: id}
				id++
				objs = append(objs, obj)
				tree.InsertObject(obj)
			}
		}
	}

	frustums := NewFrustums(4)
	frustums.Container[0].Rebuild(makeOrthoVP(-95, 35, -35, 95, -95, 95))
	frustums.Container[1].Rebuild(makePerspVP(math.Pi/3, 1.0, 10, 140, 0, 0, -50))
	frustums.Container[2].Rebuild(makeOrthoVP(25, 110, 25, 110, -50, 50))
	frustums.Container[3].Rebuild(makeOrthoVP(-100, 100, -100, 100, -20, 20))
	frustums.Container[3].DisableNearPlane()

	if !frustums.Container[0].hasBounds || !frustums.Container[1].hasBounds || !frustums.Container[2].hasBounds {
		t.Fatalf("expected valid world bounds for ortho and perspective frustums: %v %v %v", frustums.Container[0].hasBounds, frustums.Container[1].hasBounds, frustums.Container[2].hasBounds)
	}
	if frustums.Container[3].hasBounds {
		t.Fatalf("expected DisableNearPlane to clear finite frustum bounds")
	}

	// Nil Frustums / nil slot safety check.
	tree.QueryFrustumsMask(nil, 0b1111, func(object IObject, leafMask uint32) bool {
		t.Fatalf("unexpected callback on nil Frustums")
		return false
	})

	for _, mask := range []uint32{0b0001, 0b0010, 0b1000, 0b1111, 0b0101} {
		expected := make(map[int]uint32)
		for bit := 0; bit < 4; bit++ {
			if (mask & (uint32(1) << bit)) == 0 {
				continue
			}
			tree.QueryFrustum(frustums.Container[bit], func(object IObject) bool {
				expected[object.(*testTreeObj).id] |= uint32(1) << bit
				return false
			})
		}
		got := make(map[int]uint32)
		tree.QueryFrustumsMask(frustums, mask, func(object IObject, leafMask uint32) bool {
			obj := object.(*testTreeObj)
			if _, dup := got[obj.id]; dup {
				t.Fatalf("duplicate callback invocation for object %d under mask %b", obj.id, mask)
			}
			got[obj.id] = leafMask
			return false
		})
		if len(got) != len(expected) {
			t.Fatalf("mask %b object count mismatch: got %d want %d", mask, len(got), len(expected))
		}
		for objID, wantMask := range expected {
			if got[objID] != wantMask {
				t.Fatalf("mask %b object %d leafMask=%b want %b", mask, objID, got[objID], wantMask)
			}
		}
	}

	allocs := testing.AllocsPerRun(50, func() {
		tree.QueryFrustumsMask(frustums, 0b1111, func(object IObject, leafMask uint32) bool {
			return false
		})
	})
	if allocs > 0 {
		t.Fatalf("expected 0 allocs on QueryFrustumsMask, got %.2f", allocs)
	}
}

func TestTreeVersionAndBoxPoseEpochTrackMutations(t *testing.T) {
	tree := NewTree(4, 2.0)
	v0 := tree.Version()

	box := NewBoundingBox()
	box.Rebuild(0, 0, 0, 10, 10, 10)
	pose0 := box.GetPoseGen()
	global0 := GlobalBoxPoseEpoch()

	obj := &testTreeObj{box: box.GetAABB(), id: 1}
	tree.InsertObject(obj)
	if tree.Version() <= v0 {
		t.Fatalf("expected tree version increment after InsertObject")
	}

	// Small move within fat margin still increments tree.Version, box poseGen, and global epoch.
	v1 := tree.Version()
	box.MoveCenterTo(5.2, 5.0, 5.0)
	if box.GetPoseGen() <= pose0 || GlobalBoxPoseEpoch() <= global0 {
		t.Fatalf("expected poseGen and global epoch to advance on MoveCenterTo")
	}
	tree.UpdateObject(obj)
	if tree.Version() <= v1 {
		t.Fatalf("expected tree version increment even when UpdateObject stays within fat margin")
	}

	// Rotation bumps poseGen.
	pose1 := box.GetPoseGen()
	box.SetRotationAngleZ(0.25)
	if box.GetPoseGen() <= pose1 {
		t.Fatalf("expected poseGen increment on SetRotationAngleZ")
	}

	v2 := tree.Version()
	tree.RemoveObject(obj)
	if tree.Version() <= v2 {
		t.Fatalf("expected tree version increment after RemoveObject")
	}

	v3 := tree.Version()
	tree.Clear()
	if tree.Version() <= v3 {
		t.Fatalf("expected tree version increment after Clear")
	}
	if !tree.IsEmpty() || tree.ContainsObject(obj) {
		t.Fatalf("expected Clear to reset empty state and invalidate cleared object membership")
	}
}

func TestTreeEnsureCapacityAndGeometricGrowth(t *testing.T) {
	// Starts with initialSize=1 (worst-case growthSize=1 in legacy linear mode) and inserts 2048 items.
	const count = 2048
	tree := NewTree(1, 0.0)
	if !tree.IsEmpty() {
		t.Fatalf("expected new tree to start empty")
	}

	objs := make([]*testTreeObj, count)
	for i := range count {
		ab := NewAABB()
		ab.Rebuild(float64(i)*10, 0, 0, float64(i)*10+5, 5, 5)
		objs[i] = &testTreeObj{box: ab, id: i}
	}

	// EnsureCapacity must pre-size cleanly in one jump, then preserve node membership after insertions and Clear.
	tree.EnsureCapacity(count * 2)
	capAfterEnsure := tree.Capacity()
	if capAfterEnsure < count*2 {
		t.Fatalf("expected capacity >= %d after EnsureCapacity, got %d", count*2, capAfterEnsure)
	}
	for _, obj := range objs {
		tree.InsertObject(obj)
	}
	if tree.Capacity() != capAfterEnsure {
		t.Fatalf("expected zero further reallocations after EnsureCapacity(%d), capacity grew to %d", capAfterEnsure, tree.Capacity())
	}
	for _, obj := range objs {
		if !tree.ContainsObject(obj) {
			t.Fatalf("expected inserted object %d to be in tree", obj.id)
		}
	}

	// Clear must clear leaf indices on all objects so stale indices do not match ContainsObject.
	tree.Clear()
	if !tree.IsEmpty() {
		t.Fatalf("expected tree to be empty after Clear")
	}
	for _, obj := range objs {
		if tree.ContainsObject(obj) || obj.GetAABB().GetIndex() != -1 {
			t.Fatalf("expected cleared object %d index=-1 and ContainsObject=false", obj.id)
		}
	}

	// Direct InsertObject without EnsureCapacity must grow geometrically rather than linearly (+1 per node).
	allocs := testing.AllocsPerRun(1, func() {
		growTree := NewTree(1, 0.0)
		for i := range count {
			ab := NewAABB()
			ab.Rebuild(float64(i)*10, 0, 0, float64(i)*10+5, 5, 5)
			growTree.InsertObject(&testTreeObj{box: ab, id: i})
		}
	})
	// With exponential growth, 4096 nodes take ~10 growth steps (~3 allocs/step + 1 AABB each) instead of 4000+ slice copies.
	if allocs > 6000 {
		t.Fatalf("unexpectedly high allocations during geometric tree growth: %.0f", allocs)
	}
}
