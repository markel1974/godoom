package draws

import (
	"cmp"
	"slices"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/gpu/buffers"
)

func (w *DrawCommandsRender) acquireStaticBatch(pool *[]*DrawBatchStaticMulti, count *int) *DrawBatchStaticMulti {
	idx := *count
	if idx >= len(*pool) {
		oldLen := len(*pool)
		newLen := max(oldLen*2, 16)
		grown := make([]*DrawBatchStaticMulti, newLen)
		copy(grown, *pool)
		for i := oldLen; i < newLen; i++ {
			grown[i] = NewDrawBatchStaticMulti()
		}
		*pool = grown
	}
	b := (*pool)[idx]
	*count = idx + 1
	return b
}

func (w *DrawCommandsRender) classifySubMesh(sub *buffers.ResidentSubMesh) {
	if sub.HasCutout {
		w.hasCutout = true
		if sub.DepthWrite && !sub.PolygonOffset {
			w.hasCutoutPrepass = true
		}
		return
	}
	w.hasSolid = true
	if sub.DepthWrite && !sub.PolygonOffset {
		w.hasSolidPrepass = true
	}
}

func (w *DrawCommandsRender) compileShadowStaticBatches(cutout bool, dst *[]*DrawBatchStaticMulti, dstLen *int) {
	*dstLen = 0
	if w.staticLen == 0 {
		return
	}
	if len(w.staticScratch) < w.staticLen {
		w.staticScratch = make([]*DrawItemStatic, max(len(w.staticScratch)*2, w.staticLen))
	}
	n := 0
	for i := 0; i < w.staticLen; i++ {
		item := w.staticItems[i]
		sub := item.Sub
		if sub.IndexCount > 0 && sub.HasCutout == cutout {
			w.staticScratch[n] = item
			n++
		}
	}
	if n == 0 {
		return
	}
	scratch := w.staticScratch[:n]
	if n > 1 {
		slices.SortFunc(scratch, func(a, b *DrawItemStatic) int {
			if a.Mesh.VAO != b.Mesh.VAO {
				return cmp.Compare(a.Mesh.VAO, b.Mesh.VAO)
			}
			dwa := 0
			if !w.isAdditive && a.Sub.DepthWrite {
				dwa = 1
			}
			dwb := 0
			if !w.isAdditive && b.Sub.DepthWrite {
				dwb = 1
			}
			if dwa != dwb {
				// Draw DepthWrite=1 first so Depth.Render's initial DepthMask(true) avoids redundant toggles.
				return cmp.Compare(dwb, dwa)
			}
			return cmp.Compare(a.Sub.FirstIndex, b.Sub.FirstIndex)
		})
	}
	var curBatch *DrawBatchStaticMulti
	curVAO := uint32(0)
	curDW := -1
	for i := 0; i < n; i++ {
		item := scratch[i]
		sub := item.Sub
		dw := 0
		if !w.isAdditive && sub.DepthWrite {
			dw = 1
		}
		if curBatch == nil || item.Mesh.VAO != curVAO || dw != curDW {
			curVAO = item.Mesh.VAO
			curDW = dw
			curBatch = w.acquireStaticBatch(dst, dstLen)
			curBatch.Reset(curVAO, dw, 0, config.CullNone, cutout)
		}
		curBatch.AppendRange(w.ctx, sub.FirstIndex, sub.IndexCount)
	}
}

func (w *DrawCommandsRender) compileOrderedStaticBatches(cutoutFilter int, prepassOnly bool, dst *[]*DrawBatchStaticMulti, dstLen *int) {
	*dstLen = 0
	if w.staticLen == 0 {
		return
	}
	var curBatch *DrawBatchStaticMulti
	curVAO := uint32(0)
	curDW := -1
	curPO := -1
	curCull := -1
	for i := 0; i < w.staticLen; i++ {
		item := w.staticItems[i]
		sub := item.Sub
		if sub.IndexCount == 0 {
			continue
		}
		if cutoutFilter >= 0 && ((cutoutFilter == 1) != sub.HasCutout) {
			continue
		}
		if prepassOnly && (!sub.DepthWrite || sub.PolygonOffset) {
			continue
		}
		dw := 0
		if !w.isAdditive && sub.DepthWrite {
			dw = 1
		}
		po := 0
		if sub.PolygonOffset {
			po = 1
		}
		if curBatch == nil || item.Mesh.VAO != curVAO || dw != curDW || po != curPO || sub.CullMode != curCull || curBatch.hasCutout != sub.HasCutout {
			curVAO = item.Mesh.VAO
			curDW = dw
			curPO = po
			curCull = sub.CullMode
			curBatch = w.acquireStaticBatch(dst, dstLen)
			curBatch.Reset(curVAO, dw, po, sub.CullMode, sub.HasCutout)
		}
		curBatch.AppendRange(w.ctx, sub.FirstIndex, sub.IndexCount)
	}
}

func (w *DrawCommandsRender) compileShadowInstanced(cutout bool, dst *[]InstShadowCommand, dstLen *int) {
	*dstLen = 0
	if w.instancedLen == 0 {
		return
	}
	if len(*dst) < w.instancedLen {
		*dst = make([]InstShadowCommand, max(len(*dst)*2, w.instancedLen))
	}
	out := *dst
	n := 0
	// Two passes put DepthWrite=1 before DepthWrite=0 without allocation.
	for passDW := 1; passDW >= 0; passDW-- {
		for i := 0; i < w.instancedLen; i++ {
			item := w.instancedItems[i]
			sub := item.Sub
			if sub.IndexCount == 0 || sub.HasCutout != cutout {
				continue
			}
			dw := 0
			if !w.isAdditive && sub.DepthWrite {
				dw = 1
			}
			if dw != passDW {
				continue
			}
			po := 0
			if sub.PolygonOffset {
				po = 1
			}
			out[n] = InstShadowCommand{
				mesh:           item.Mesh,
				instanceVBO:    item.InstanceVBO,
				baseByteOffset: item.BaseByteOffset,
				instanceCount:  item.InstanceCount,
				indexCount:     sub.IndexCount,
				indexOffsetPtr: w.ctx.PtrOffset(int(sub.FirstIndex) * 4),
				depthWrite:     dw,
				polygonOffset:  po,
				cullMode:       sub.CullMode,
				hasCutout:      sub.HasCutout,
			}
			n++
		}
	}
	*dstLen = n
}

// SetResidentDraws attaches GPU-resident static and instanced draw slices built for the current frame.
func (w *DrawCommandsRender) SetResidentDraws(staticItems []*DrawItemStatic, staticLen int, instancedItems []*DrawItemInstanced, instancedLen int) {
	w.staticItems = staticItems
	w.staticLen = staticLen
	w.instancedItems = instancedItems
	w.instancedLen = instancedLen
	for i := range staticLen {
		w.classifySubMesh(staticItems[i].Sub)
	}
	for i := range instancedLen {
		w.classifySubMesh(instancedItems[i].Sub)
	}

	w.compileShadowStaticBatches(false, &w.shadowSolidStatic, &w.shadowSolidStaticLen)
	w.compileShadowStaticBatches(true, &w.shadowCutoutStatic, &w.shadowCutoutStaticLen)
	w.compileShadowInstanced(false, &w.shadowSolidInst, &w.shadowSolidInstLen)
	w.compileShadowInstanced(true, &w.shadowCutoutInst, &w.shadowCutoutInstLen)

	if w.isAdditive {
		w.prepassSolidStaticLen = 0
		w.prepassCutoutStaticLen = 0
		w.shadeSolidStaticLen = 0
		w.shadeCutoutStaticLen = 0
		w.compileOrderedStaticBatches(-1, false, &w.renderAllStatic, &w.renderAllStaticLen)
		return
	}
	w.renderAllStaticLen = 0
	w.compileOrderedStaticBatches(0, true, &w.prepassSolidStatic, &w.prepassSolidStaticLen)
	w.compileOrderedStaticBatches(1, true, &w.prepassCutoutStatic, &w.prepassCutoutStaticLen)
	w.compileOrderedStaticBatches(0, false, &w.shadeSolidStatic, &w.shadeSolidStaticLen)
	w.compileOrderedStaticBatches(1, false, &w.shadeCutoutStatic, &w.shadeCutoutStaticLen)
}

// Prepare organizes and processes an array of DrawCommand instances into renderable batches for efficient rendering.
func (w *DrawCommandsRender) Prepare(dc []*DrawCommand) {
	w.batchesLen = 0
	w.streamSolid = 0
	w.streamCutout = 0
	w.hasSolidPrepass = false
	w.hasCutoutPrepass = false
	w.hasSolid = false
	w.hasCutout = false
	w.staticItems = nil
	w.staticLen = 0
	w.instancedItems = nil
	w.instancedLen = 0
	w.transparentItems = nil
	w.shadowSolidStaticLen = 0
	w.shadowCutoutStaticLen = 0
	w.shadowSolidInstLen = 0
	w.shadowCutoutInstLen = 0
	w.prepassSolidStaticLen = 0
	w.prepassCutoutStaticLen = 0
	w.shadeSolidStaticLen = 0
	w.shadeCutoutStaticLen = 0
	w.renderAllStaticLen = 0

	if len(dc) == 0 {
		return
	}

	currentDepthWrite := -1
	currentPolygonOffset := -1
	currentHasCutout := -1
	currentCullMode := -1

	var currentBatch *DrawBatch

	for _, cmd := range dc {
		if cmd.indexCount == 0 {
			continue
		}

		w.totalDraws++
		w.totalIndices += uint64(cmd.indexCount)

		depthWrite := 0
		polygonOffset := 0
		hasCutout := 0
		cullMode := config.CullFront

		if cmd.material != nil {
			if !w.isAdditive && cmd.material.GetDepthWrite() {
				depthWrite = 1
			}
			if cmd.material.GetPolygonOffset() {
				polygonOffset = 1
			}
			if !w.isAdditive && cmd.material.HasCutout() {
				hasCutout = 1
			}
			cullMode = cmd.material.GetCullMode()
		} else {
			if !w.isAdditive {
				depthWrite = 1
			}
		}

		if hasCutout == 1 {
			w.hasCutout = true
			if depthWrite == 1 && polygonOffset == 0 {
				w.hasCutoutPrepass = true
			}
		} else {
			w.hasSolid = true
			if depthWrite == 1 && polygonOffset == 0 {
				w.hasSolidPrepass = true
			}
		}

		if depthWrite != currentDepthWrite || polygonOffset != currentPolygonOffset || hasCutout != currentHasCutout || cullMode != currentCullMode || currentBatch == nil {
			if w.batchesLen >= len(w.batches) {
				w.grow()
			}
			currentBatch = w.batches[w.batchesLen]
			currentBatch.Reset(depthWrite, polygonOffset, hasCutout, cullMode)
			w.batchesLen++
			if hasCutout == 1 {
				w.streamCutout++
			} else {
				w.streamSolid++
			}
			currentDepthWrite = depthWrite
			currentPolygonOffset = polygonOffset
			currentHasCutout = hasCutout
			currentCullMode = cullMode
		}

		currentBatch.Add(cmd.indexCount, cmd.firstIndex*4)
	}
}

// grow doubles the capacity of the render batches slice and initializes new DrawBatch instances for added capacity.
func (w *DrawCommandsRender) grow() {
	batches := w.batches
	newLen := len(w.batches) * 2
	w.batches = make([]*DrawBatch, newLen)
	copy(w.batches, batches)
	for i := len(batches); i < newLen; i++ {
		w.batches[i] = NewDrawBatch()
	}
}
