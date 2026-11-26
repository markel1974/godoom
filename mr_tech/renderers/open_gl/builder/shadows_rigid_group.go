package builder

import "github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"

// ShadowsRigidGroup manages shadow-casting rigid mesh entries backed by a resident mesh.
type ShadowsRigidGroup struct {
	mesh       *buffers.ResidentMesh
	entries    []*ShadowRigidEntry
	entriesLen int
}

// Assign sets the resident mesh for the group and clears its active shadow entries.
func (g *ShadowsRigidGroup) Assign(rm *buffers.ResidentMesh) {
	g.mesh = rm
	g.entriesLen = 0
}

// Add appends a rigid shadow entry to the group, growing the backing storage as needed.
func (g *ShadowsRigidGroup) Add(v *ShadowRigidEntry) {
	if g.entriesLen >= len(g.entries) {
		if newLen := len(g.entries) * 2; newLen == 0 {
			g.entries = make([]*ShadowRigidEntry, 64)
		} else {
			z := make([]*ShadowRigidEntry, newLen)
			copy(z, g.entries)
			g.entries = z
		}
	}
	g.entries[g.entriesLen] = v
	g.entriesLen++
}

// Len returns the number of active shadow entries in the group.
func (g *ShadowsRigidGroup) Len() int {
	return g.entriesLen
}

// At returns the shadow transform matrix for the entry at idx.
func (g *ShadowsRigidGroup) At(idx int) *[16]float32 {
	return &g.entries[idx].mat
}
