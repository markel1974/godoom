package shaders

import (
	"math"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

const (
	lightsDoubleBuffer = 2

	// Clustered forward 3D grid dimensions (16x9x16 = 2304 clusters).
	ClusterGridX = 16
	ClusterGridY = 9
	ClusterGridZ = 16
	ClusterCount = ClusterGridX * ClusterGridY * ClusterGridZ // 2304

	// Single 128x1200 RGBA16UI lookup texture: rows 0..17 store [offsetLo16, count, offsetHi16, 0], rows 18..1199 store 4 uint16 indices/texel.
	ClusterTexWidth        = 128
	ClusterTexHeight       = 1200
	clusterHeaderRows      = 18
	clusterHeaderTexels    = ClusterCount // 18 * 128 = 2304
	clusterPoolBaseUint16  = clusterHeaderTexels * 4
	clusterPoolMaxIndices  = (ClusterTexHeight - clusterHeaderRows) * ClusterTexWidth * 4 // 605184 (> 2304 * 256)
	clusterMaxLightsPerBin = 256
	maxUBOLights           = 256
)

// LightLoc represents an identifier for light-related uniform locations in a shader program.
type LightLoc int

// LightLocProjection specifies the light's projection matrix location.
// LightLocView specifies the light's view matrix location.
// LightLocInvView specifies the light's inverse view matrix location.
// LightLocRoomSpaceMatrix specifies the light's room-space matrix location.
// LightLocTexture specifies the light's texture resource location.
// LightLocNormalMap specifies the light's normal map resource location.
// LightLocRoomShadowMap specifies the light's room shadow map resource location.
// LightLocScreenResolution specifies the screen resolution location.
// LightLocAmbientLight specifies the ambient light intensity location.
// LightLocEnableShadows specifies whether shadows are enabled location.
// LightLocVolumetricSteps specifies the number of volumetric lighting steps location.
// LightLocBeamRatioFactor specifies the beam-to-light ratio factor location.
// LightLocNumLights specifies the number of active lights location.
// LightLocLast specifies the marker for the last LightLoc value.
const (
	LightLocProjection = LightLoc(iota)
	LightLocView
	LightLocTexture
	LightLocNormalMap

	LightLocScreenResolution
	LightLocAmbientLight

	LightLocNumLights
	LightLocDebugLights
	LightLocLast
)

//LightLocInvView
//LightLocVolumetricSteps
//LightLocRoomSpaceMatrix
//LightLocRoomShadowMap
//LightLocEnableShadows
//LightLocBeamRatioFactor

// Lights represents a collection of light data and OpenGL resources for managing and rendering dynamic scene lighting.
type Lights struct {
	ctx             api.IContext
	prg             uint32
	table           [LightLocLast]int32
	uboLights       [lightsDoubleBuffer]uint32
	viewLights      []float32
	viewLightsPtr   unsafe.Pointer
	frameIdx        int
	activeLights    int32
	shadows         int32
	stride          int32
	handler         *Handler
	cal             *model.Calibration
	main            *Main
	depth           *Depth
	mapMetrics      *metrics.Map
	debugLights     int
	clusterTex      uint32
	clusterBuf      []uint16
	clusterCounts   [ClusterCount]uint16
	clusterCells    []uint16
	clusterNear     float32
	clusterLogScale float32
	globalCount     int32
	clustersReady   bool
	lightOrder      []int
	lightScores     []float32
}

const lightFloatsPerEntry = 16

// NewLights initializes and returns a new instance of Lights with default settings.
func NewLights(stride int32) *Lights {
	viewLights := make([]float32, 1024*lightFloatsPerEntry)
	return &Lights{
		stride:        stride,
		viewLights:    viewLights,
		viewLightsPtr: unsafe.Pointer(&viewLights[0]),
		frameIdx:      0,
		debugLights:   0,
		clusterBuf:    make([]uint16, ClusterTexWidth*ClusterTexHeight*4),
		clusterCells:  make([]uint16, ClusterCount*clusterMaxLightsPerBin),
		lightOrder:    make([]int, 1024),
		lightScores:   make([]float32, 1024),
	}
}

// Setup initializes the Lights instance by assigning the rendering context from the provided Handler.
func (s *Lights) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.cal = h.GetCalibration()
	s.main = h.GetMain()
	s.depth = h.GetDepth()
	s.mapMetrics = h.GetMapMetrics()
	return nil
}

// Init initializes the UBO and clustered forward lookup texture resources.
func (s *Lights) Init() error {
	size := 1024 * int(s.stride)

	s.ctx.GenBuffers(lightsDoubleBuffer, &s.uboLights[0])

	for i := 0; i < lightsDoubleBuffer; i++ {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, size, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)

	s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
	s.ctx.GenTextures(1, &s.clusterTex)
	s.ctx.BindTexture(api.TEXTURE_2D, s.clusterTex)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, int32(api.RGBA16UI), ClusterTexWidth, ClusterTexHeight, 0, api.RGBA_INTEGER, api.UNSIGNED_SHORT, s.ctx.Ptr(s.clusterBuf))
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	return nil
}

// SetupSamplers configures shader samplers for texture, normal map, and room shadow map locations.
func (s *Lights) SetupSamplers() error {
	if s.prg == 0 {
		return nil
	}
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	normalUnits := []int32{api.TexUnitNormalBase, api.TexUnitNormalBase + 1, api.TexUnitNormalBase + 2, api.TexUnitNormalBase + 3}
	s.ctx.Uniform1iv(s.GetUniform(LightLocTexture), 4, &diffuseUnits[0]) // FlashLocTexture in shadow_light.go
	s.ctx.Uniform1iv(s.GetUniform(LightLocNormalMap), 4, &normalUnits[0])
	//s.ctx.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 12) // gl_api.TEXTURE12 per Room, gl_api.TEXTURE13 per Flash
	return nil
}

// GetUniform retrieves the uniform location for a given LightLoc identifier from the internal table.
func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

// Compile initializes UBO and clustered lighting resources for the active forward pipeline.
// The legacy standalone lights.frag shader is not compiled at startup.
func (s *Lights) Compile(_ IAssets) error {
	return nil
}

// Prepare pre-transforms active lights into camera view space, packs squared range and
// inverse falloff into spot_params.zw, and uploads the inactive UBO buffer cleanly.
func (s *Lights) Prepare(frameLights unsafe.Pointer, lightSize int32, numLights int32) {
	s.activeLights = numLights
	s.frameIdx = (s.frameIdx + 1) % lightsDoubleBuffer
	s.clustersReady = false
	if numLights <= 0 || frameLights == nil {
		s.activeLights = 0
		s.globalCount = 0
		return
	}
	const floatsPerLight = 16
	srcCount := int(numLights)
	srcFloats := srcCount * floatsPerLight
	src := unsafe.Slice((*float32)(frameLights), srcFloats)

	outCount := srcCount
	uploadBytes := int(lightSize)
	if srcCount > maxUBOLights {
		outCount = maxUBOLights
		s.activeLights = maxUBOLights
		uploadBytes = maxUBOLights * floatsPerLight * 4
	}
	neededFloats := outCount * floatsPerLight
	if neededFloats > len(s.viewLights) {
		s.viewLights = make([]float32, neededFloats)
		s.viewLightsPtr = unsafe.Pointer(&s.viewLights[0])
	}
	dst := s.viewLights[:neededFloats]

	packLight := func(srcIdx, dstIdx int) {
		sOff := srcIdx * floatsPerLight
		dOff := dstIdx * floatsPerLight

		dst[dOff], dst[dOff+1], dst[dOff+2] = s.mapMetrics.TransformPointToView(src[sOff], src[sOff+1], src[sOff+2])
		dst[dOff+8], dst[dOff+9], dst[dOff+10] = s.mapMetrics.TransformDirectionNormalizedToView(src[sOff+8], src[sOff+9], src[sOff+10])

		dst[dOff+3] = src[sOff+3]
		dst[dOff+4] = src[sOff+4]
		dst[dOff+5] = src[sOff+5]
		dst[dOff+6] = src[sOff+6]
		dst[dOff+7] = src[sOff+7]
		dst[dOff+11] = src[sOff+11]
		dst[dOff+12] = src[sOff+12]
		dst[dOff+13] = src[sOff+13]
		dst[dOff+14], dst[dOff+15] = metrics.ComputeLightAttenuation(src[sOff+7], src[sOff+11])
	}

	if srcCount <= maxUBOLights {
		for i := range srcCount {
			packLight(i, i)
		}
	} else {
		if cap(s.lightOrder) < srcCount {
			s.lightOrder = make([]int, srcCount)
			s.lightScores = make([]float32, srcCount)
		}
		order := s.lightOrder[:srcCount]
		scores := s.lightScores[:srcCount]
		for i := range srcCount {
			order[i] = i
			sOff := i * floatsPerLight
			intensity := src[sOff+7]
			if intensity <= 0.001 {
				scores[i] = 1e30
				continue
			}
			lightType := int(src[sOff+3] + 0.5)
			if lightType == 2 {
				scores[i] = -1e30
				continue
			}
			vx, vy, vz := src[sOff], src[sOff+1], src[sOff+2]
			vx, vy, vz = s.mapMetrics.TransformPointToView(vx, vy, vz)

			effSq, _ := metrics.ComputeLightAttenuation(intensity, src[sOff+11])
			scores[i] = (vx*vx + vy*vy + vz*vz) - effSq
		}
		// Partial selection sort for top maxUBOLights nearest/relevant lights (0 allocs).
		for i := range maxUBOLights {
			minPos := i
			minScore := scores[order[i]]
			for j := i + 1; j < srcCount; j++ {
				sc := scores[order[j]]
				if sc < minScore {
					minScore = sc
					minPos = j
				}
			}
			order[i], order[minPos] = order[minPos], order[i]
			packLight(order[i], i)
		}
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
	// Scrittura asincrona garantita sull'UBO inattivo
	s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, uploadBytes, s.viewLightsPtr)
	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
}

// Render configures the shader program and draws geometry with lighting, shadows, and volumetric effects applied.
func (s *Lights) Render(renderGeometry func(), ambient float32) {
	vao := s.main.GetVAO()
	roomShadowTex := s.depth.GetRoomShadowTextures()
	proj := s.mapMetrics.GetProjPtr()
	view := s.mapMetrics.GetViewPtr()
	fbW, fbH := s.handler.GetSize()

	s.ctx.UseProgram(s.prg)
	s.ctx.BindVertexArray(vao)

	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, view)
	//s.ctx.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, invView)
	//s.ctx.UniformMatrix4fv(s.GetUniform(LightLocRoomSpaceMatrix), 1, false, roomSpace)
	s.ctx.Uniform1i(s.GetUniform(LightLocNumLights), s.activeLights)
	s.ctx.Uniform2f(s.GetUniform(LightLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)
	//s.ctx.Uniform1i(s.GetUniform(LightLocEnableShadows), s.shadows)
	//s.ctx.Uniform1i(s.GetUniform(LightLocVolumetricSteps), int32(s.cal.VolSteps))
	//s.ctx.Uniform1f(s.GetUniform(LightLocBeamRatioFactor), float32(s.cal.BeamRatio))
	s.ctx.Uniform1i(s.GetUniform(LightLocDebugLights), int32(s.debugLights))
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitRoomShadow)
		s.ctx.BindTexture(api.TEXTURE_2D, roomShadowTex)
	}
	renderGeometry()
}

// SetShadowsEnabled enables or disables shadow rendering based on the provided boolean value.
func (s *Lights) SetShadowsEnabled(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
}

// GetActiveUBO returns the currently active UBO buffer for dynamic lights.
func (s *Lights) GetActiveUBO() uint32 {
	return s.uboLights[s.frameIdx]
}

// GetActiveLightsCount returns the number of active dynamic lights in the current frame.
func (s *Lights) GetActiveLightsCount() int32 {
	return s.activeLights
}

// ClusterID computes the linearized cluster header texel index for 3D grid cell (cx, cy, cz).
func ClusterID(cx, cy, cz int) int {
	return (cz*ClusterGridY+cy)*ClusterGridX + cx
}

// GetClusterState returns the last built clustered forward state and lookup texture ID.
func (s *Lights) GetClusterState() (tex uint32, near float32, logScale float32, globalCount int32) {
	return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
}

// GetClusterCell reads the packed (offset, count) entry for cluster (cx, cy, cz) from the CPU staging buffer.
func (s *Lights) GetClusterCell(cx, cy, cz int) (offset uint32, count uint32) {
	hOff := ClusterID(cx, cy, cz) * 4
	offset = uint32(s.clusterBuf[hOff]) | (uint32(s.clusterBuf[hOff+2]) << 16)
	count = uint32(s.clusterBuf[hOff+1])
	return offset, count
}

// GetClusterPoolEntry reads a packed light index from the CPU cluster index pool.
func (s *Lights) GetClusterPoolEntry(slot int) uint16 {
	return s.clusterBuf[clusterPoolBaseUint16+slot]
}

// PrepareClusters builds the 16x9x16 log-Z clustered forward light grid and uploads it into u_clusterData.
func (s *Lights) PrepareClusters() (tex uint32, near float32, logScale float32, globalCount int32) {
	if s.activeLights <= 0 {
		s.globalCount = 0
		if s.clusterNear < 0.5 {
			s.clusterNear = 0.5
		}
		s.clustersReady = true
		return s.clusterTex, s.clusterNear, 0, 0
	}
	if s.clustersReady {
		return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
	}

	proj := s.mapMetrics.GetProj()
	zNear := s.mapMetrics.GetRoomZNear()
	zFar := s.mapMetrics.GetRoomZFar()

	usedPool := s.PrepareClustersWithProj(proj, zNear, zFar)
	if s.clusterTex != 0 {
		poolTexels := (usedPool + 3) / 4
		poolRows := (poolTexels + ClusterTexWidth - 1) / ClusterTexWidth
		totalRows := clusterHeaderRows + poolRows
		if totalRows > ClusterTexHeight {
			totalRows = ClusterTexHeight
		}
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, s.clusterTex)
		needed := totalRows * ClusterTexWidth * 4
		if needed > len(s.clusterBuf) {
			needed = len(s.clusterBuf)
		}
		s.ctx.TexSubImage2D(api.TEXTURE_2D, 0, 0, 0, ClusterTexWidth, int32(totalRows), api.RGBA_INTEGER, api.UNSIGNED_SHORT, s.ctx.Ptr(s.clusterBuf[:needed]))
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	s.clustersReady = true
	return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
}

// PrepareClustersWithProj bins active view-space lights into the 16x9x16 clustered grid using proj, zNear, zFar.
// Returns the total number of packed pool indices (global lights + per-cluster references) with zero heap allocations.
func (s *Lights) PrepareClustersWithProj(proj [16]float32, zNear, zFar float32) int {
	if zNear < 0.5 {
		zNear = 0.5
	}
	if zFar <= zNear+1.0 {
		zFar = zNear + 8192.0
	}
	s.clusterNear = zNear
	s.clusterLogScale = float32(float64(ClusterGridZ) / math.Log(float64(zFar/zNear)))

	clear(s.clusterCounts[:])

	n := int(s.activeLights)
	if n > 256 {
		n = 256
	}
	globalCount := int32(0)
	isPersp := math.Abs(float64(proj[11]+1.0)) < 1e-4 && math.Abs(float64(proj[0])) > 1e-6 && math.Abs(float64(proj[5])) > 1e-6

	for i := 0; i < n; i++ {
		off := i * lightFloatsPerEntry
		intensity := s.viewLights[off+7]
		if intensity <= 0.001 {
			continue
		}
		lightType := int(s.viewLights[off+3] + 0.5)
		effRadiusSq := s.viewLights[off+14]
		if lightType == 2 || effRadiusSq <= 0 || effRadiusSq > 1e12 {
			if int(globalCount) < clusterPoolMaxIndices {
				s.clusterBuf[clusterPoolBaseUint16+int(globalCount)] = uint16(i)
				globalCount++
			}
			continue
		}

		vx, vy, vz := s.viewLights[off], s.viewLights[off+1], s.viewLights[off+2]
		r := float32(math.Sqrt(float64(effRadiusSq)))
		dMin := -vz - r
		dMax := -vz + r
		if dMax < zNear || dMin > zFar {
			continue
		}
		dMinC := dMin
		if dMinC < zNear {
			dMinC = zNear
		}
		dMaxC := dMax
		if dMaxC > zFar {
			dMaxC = zFar
		}

		cz0 := int(math.Floor(math.Log(float64(dMinC/zNear)) * float64(s.clusterLogScale)))
		cz1 := int(math.Floor(math.Log(float64(dMaxC/zNear)) * float64(s.clusterLogScale)))
		if cz0 < 0 {
			cz0 = 0
		} else if cz0 >= ClusterGridZ {
			cz0 = ClusterGridZ - 1
		}
		if cz1 < cz0 {
			cz1 = cz0
		} else if cz1 >= ClusterGridZ {
			cz1 = ClusterGridZ - 1
		}

		var cx0, cx1, cy0, cy1 int
		if !isPersp || (vx*vx+vy*vy+vz*vz <= effRadiusSq) {
			cx0, cx1 = 0, ClusterGridX-1
			cy0, cy1 = 0, ClusterGridY-1
		} else {
			zA := -dMinC
			zB := -dMaxC
			xMin, xMax := vx-r, vx+r
			yMin, yMax := vy-r, vy+r

			minNdcX, maxNdcX := float32(1e9), float32(-1e9)
			for ix := 0; ix < 2; ix++ {
				xv := xMin
				if ix == 1 {
					xv = xMax
				}
				for iz := 0; iz < 2; iz++ {
					zv := zA
					if iz == 1 {
						zv = zB
					}
					ndcX := (proj[0]*xv + proj[8]*zv) / (proj[11] * zv)
					if ndcX < minNdcX {
						minNdcX = ndcX
					}
					if ndcX > maxNdcX {
						maxNdcX = ndcX
					}
				}
			}

			minNdcY, maxNdcY := float32(1e9), float32(-1e9)
			for iy := 0; iy < 2; iy++ {
				yv := yMin
				if iy == 1 {
					yv = yMax
				}
				for iz := 0; iz < 2; iz++ {
					zv := zA
					if iz == 1 {
						zv = zB
					}
					ndcY := (proj[5]*yv + proj[9]*zv) / (proj[11] * zv)
					if ndcY < minNdcY {
						minNdcY = ndcY
					}
					if ndcY > maxNdcY {
						maxNdcY = ndcY
					}
				}
			}

			if maxNdcX < -1.0 || minNdcX > 1.0 || maxNdcY < -1.0 || minNdcY > 1.0 {
				continue
			}

			cx0 = int(math.Floor(float64(minNdcX*0.5+0.5) * float64(ClusterGridX)))
			cx1 = int(math.Floor(float64(maxNdcX*0.5+0.5) * float64(ClusterGridX)))
			cy0 = int(math.Floor(float64(minNdcY*0.5+0.5) * float64(ClusterGridY)))
			cy1 = int(math.Floor(float64(maxNdcY*0.5+0.5) * float64(ClusterGridY)))

			if cx0 < 0 {
				cx0 = 0
			} else if cx0 >= ClusterGridX {
				cx0 = ClusterGridX - 1
			}
			if cx1 < cx0 {
				cx1 = cx0
			} else if cx1 >= ClusterGridX {
				cx1 = ClusterGridX - 1
			}
			if cy0 < 0 {
				cy0 = 0
			} else if cy0 >= ClusterGridY {
				cy0 = ClusterGridY - 1
			}
			if cy1 < cy0 {
				cy1 = cy0
			} else if cy1 >= ClusterGridY {
				cy1 = ClusterGridY - 1
			}
		}

		for cz := cz0; cz <= cz1; cz++ {
			for cy := cy0; cy <= cy1; cy++ {
				rowBase := (cz*ClusterGridY + cy) * ClusterGridX
				for cx := cx0; cx <= cx1; cx++ {
					cid := rowBase + cx
					cnt := s.clusterCounts[cid]
					if cnt < clusterMaxLightsPerBin {
						s.clusterCells[cid*clusterMaxLightsPerBin+int(cnt)] = uint16(i)
						s.clusterCounts[cid] = cnt + 1
					}
				}
			}
		}
	}

	s.globalCount = globalCount
	writeIdx := int(globalCount)
	for cid := 0; cid < ClusterCount; cid++ {
		cnt := int(s.clusterCounts[cid])
		avail := clusterPoolMaxIndices - writeIdx
		if cnt > avail {
			cnt = avail
		}
		hOff := cid * 4
		s.clusterBuf[hOff] = uint16(writeIdx & 0xFFFF)
		s.clusterBuf[hOff+1] = uint16(cnt)
		s.clusterBuf[hOff+2] = uint16(writeIdx >> 16)
		if cnt > 0 {
			cellBase := cid * clusterMaxLightsPerBin
			copy(s.clusterBuf[clusterPoolBaseUint16+writeIdx:], s.clusterCells[cellBase:cellBase+cnt])
			writeIdx += cnt
		}
	}
	return writeIdx
}

/*package shaders

import (
	"fmt"
	"unsafe"

	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
)

const (
	lightsDoubleBuffer = 2
)

// LightLoc represents an identifier for light-related uniform locations in a shader program.
type LightLoc int

// LightLocProjection specifies the light's projection matrix location.
// LightLocView specifies the light's view matrix location.
// LightLocInvView specifies the light's inverse view matrix location.
// LightLocRoomSpaceMatrix specifies the light's room-space matrix location.
// LightLocTexture specifies the light's texture resource location.
// LightLocNormalMap specifies the light's normal map resource location.
// LightLocRoomShadowMap specifies the light's room shadow map resource location.
// LightLocScreenResolution specifies the screen resolution location.
// LightLocAmbientLight specifies the ambient light intensity location.
// LightLocEnableShadows specifies whether shadows are enabled location.
// LightLocVolumetricSteps specifies the number of volumetric lighting steps location.
// LightLocBeamRatioFactor specifies the beam-to-light ratio factor location.
// LightLocNumLights specifies the number of active lights location.
// LightLocLast specifies the marker for the last LightLoc value.
const (
	LightLocProjection = LightLoc(iota)
	LightLocView
	LightLocTexture
	LightLocNormalMap

	LightLocScreenResolution
	LightLocAmbientLight

	LightLocNumLights
	LightLocShininessWall
	LightLocShininessFloor
	LightLocSpecBoostWall
	LightLocSpecBoostFloor
	LightLocDebugLights
	LightLocLast
)

//LightLocInvView
//LightLocVolumetricSteps
//LightLocRoomSpaceMatrix
//LightLocRoomShadowMap
//LightLocEnableShadows
//LightLocBeamRatioFactor

// Lights represents a collection of light data and OpenGL resources for managing and rendering dynamic scene lighting.
type Lights struct {
	ctx             api.IContext
	prg             uint32
	table           [LightLocLast]int32
	uboLights       [lightsDoubleBuffer]uint32
	viewLights      []float32
	viewLightsPtr   unsafe.Pointer
	frameIdx        int
	activeLights    int32
	shadows         int32
	stride          int32
	handler         *Handler
	cal             *model.Calibration
	main            *Main
	depth           *Depth
	mapMetrics      *metrics.Map
	debugLights     int
	clusterTex      uint32
	clusterBuf      []uint16
	clusterCounts   [ClusterCount]uint16
	clusterCells    []uint16
	clusterNear     float32
	clusterLogScale float32
	globalCount     int32
	clustersReady   bool
}

const lightFloatsPerEntry = 16

// NewLights initializes and returns a new instance of Lights with default settings.
func NewLights(stride int32) *Lights {
	viewLights := make([]float32, 1024*lightFloatsPerEntry)
	return &Lights{
		stride:        stride,
		viewLights:    viewLights,
		viewLightsPtr: unsafe.Pointer(&viewLights[0]),
		frameIdx:      0,
		debugLights:   0,
		clusterBuf:    make([]uint16, ClusterTexWidth*ClusterTexHeight*4),
		clusterCells:  make([]uint16, ClusterCount*clusterMaxLightsPerBin),
	}
}

// Setup initializes the Lights instance by assigning the rendering context from the provided Handler.
func (s *Lights) Setup(h *Handler) error {
	s.handler = h
	s.ctx = h.GetContext()
	s.cal = h.GetCalibration()
	s.main = h.GetMain()
	s.depth = h.GetDepth()
	s.mapMetrics = h.GetMapMetrics()
	return nil
}

// Init initializes the UBO and clustered forward lookup texture resources.
func (s *Lights) Init() error {
	size := 1024 * int(s.stride)

	s.ctx.GenBuffers(lightsDoubleBuffer, &s.uboLights[0])

	for i := 0; i < lightsDoubleBuffer; i++ {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[i])
		s.ctx.BufferData(api.UNIFORM_BUFFER, size, s.ctx.Ptr(nil), api.DYNAMIC_DRAW)
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)

	s.ctx.GenTextures(1, &s.clusterTex)
	s.ctx.BindTexture(api.TEXTURE_2D, s.clusterTex)
	s.ctx.TexImage2D(api.TEXTURE_2D, 0, int32(api.RGBA16UI), ClusterTexWidth, ClusterTexHeight, 0, api.RGBA_INTEGER, api.UNSIGNED_SHORT, s.ctx.Ptr(s.clusterBuf))
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MIN_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_MAG_FILTER, api.NEAREST)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	s.ctx.TexParameteri(api.TEXTURE_2D, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	s.ctx.BindTexture(api.TEXTURE_2D, 0)
	return nil
}

// SetupSamplers configures shader samplers for texture, normal map, and room shadow map locations.
func (s *Lights) SetupSamplers() error {
	if s.prg == 0 {
		return nil
	}
	s.ctx.UseProgram(s.prg)
	diffuseUnits := []int32{api.TexUnitDiffuseBase, api.TexUnitDiffuseBase + 1, api.TexUnitDiffuseBase + 2, api.TexUnitDiffuseBase + 3}
	normalUnits := []int32{api.TexUnitNormalBase, api.TexUnitNormalBase + 1, api.TexUnitNormalBase + 2, api.TexUnitNormalBase + 3}
	s.ctx.Uniform1iv(s.GetUniform(LightLocTexture), 4, &diffuseUnits[0]) // FlashLocTexture in shadow_light.go
	s.ctx.Uniform1iv(s.GetUniform(LightLocNormalMap), 4, &normalUnits[0])
	//s.ctx.Uniform1i(s.GetUniform(LightLocRoomShadowMap), 12) // gl_api.TEXTURE12 per Room, gl_api.TEXTURE13 per Flash
	return nil
}

// GetUniform retrieves the uniform location for a given LightLoc identifier from the internal table.
func (s *Lights) GetUniform(id LightLoc) int32 {
	return s.table[id]
}

// Compile compiles the shaders for the Lights object and initializes its uniform locations and shader program.
func (s *Lights) Compile(a IAssets) error {
	const vertId = "main.vert"
	const fragId = "lights.frag"

	vSrc, fSrc, err := a.ReadMulti(vertId, fragId)
	if err != nil {
		return err
	}
	vSh, err := ShaderCompile(s.ctx, vertId, string(vSrc), api.VERTEX_SHADER)
	if err != nil {
		return err
	}
	fSh, err := ShaderCompile(s.ctx, fragId, string(fSrc), api.FRAGMENT_SHADER)
	if err != nil {
		s.ctx.DeleteShader(vSh)
		return err
	}
	s.prg, err = ShaderCreateProgram(s.ctx, "lights", vSh, fSh)
	if err != nil {
		return err
	}

	s.table[LightLocProjection] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_projection\x00"))
	s.table[LightLocView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_view\x00"))
	//s.table[LightLocInvView] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_invView\x00"))
	//s.table[LightLocRoomSpaceMatrix] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_roomSpaceMatrix\x00"))
	s.table[LightLocTexture] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_texture\x00"))
	s.table[LightLocNormalMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_normalMap\x00"))
	//s.table[LightLocRoomShadowMap] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_roomShadowMap\x00"))
	s.table[LightLocScreenResolution] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_screenResolution\x00"))
	s.table[LightLocAmbientLight] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_ambient_light\x00"))
	//s.table[LightLocEnableShadows] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_enableShadows\x00"))
	//s.table[LightLocVolumetricSteps] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_volumetricSteps\x00"))
	//s.table[LightLocBeamRatioFactor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_beamRatioFactor\x00"))
	s.table[LightLocNumLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_numLights\x00"))
	s.table[LightLocShininessWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessWall\x00"))
	s.table[LightLocShininessFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_shininessFloor\x00"))
	s.table[LightLocSpecBoostWall] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostWall\x00"))
	s.table[LightLocSpecBoostFloor] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_specBoostFloor\x00"))
	s.table[LightLocDebugLights] = s.ctx.GetUniformLocation(s.prg, s.ctx.Str("u_debugLights\x00"))

	for idx, v := range s.table {
		if v < 0 {
			fmt.Printf("[warning] unused uniform location in lights [table index: %d]\n", idx)
		}
	}

	blockIndex := s.ctx.GetUniformBlockIndex(s.prg, s.ctx.Str("LightsBlock\x00"))
	if blockIndex != api.INVALID_INDEX {
		s.ctx.UniformBlockBinding(s.prg, blockIndex, 0)
	}

	return nil
}

// Prepare updates the uniform buffer object (UBO) with lighting data for rendering and sets the active lights count.
func (s *Lights) Prepare(frameLightsPtr unsafe.Pointer, lightSize int32, lightNum int32) {
	s.activeLights = lightNum
	s.frameIdx = (s.frameIdx + 1) % lightsDoubleBuffer
	if lightNum > 0 {
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
		// Scrittura asincrona garantita sull'UBO inattivo
		s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, int(lightSize), frameLightsPtr)
		s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
	}
}

// Render configures the shader program and draws geometry with lighting, shadows, and volumetric effects applied.
func (s *Lights) Render(renderGeometry func(), ambient float32) {
	vao := s.main.GetVAO()
	roomShadowTex := s.depth.GetRoomShadowTextures()
	proj := s.mapMetrics.GetProjPtr()
	view := s.mapMetrics.GetViewPtr()
	fbW, fbH := s.handler.GetSize()

	s.ctx.UseProgram(s.prg)
	s.ctx.BindVertexArray(vao)

	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocProjection), 1, false, proj)
	s.ctx.UniformMatrix4fv(s.GetUniform(LightLocView), 1, false, view)
	//s.ctx.UniformMatrix4fv(s.GetUniform(LightLocInvView), 1, false, invView)
	//s.ctx.UniformMatrix4fv(s.GetUniform(LightLocRoomSpaceMatrix), 1, false, roomSpace)
	s.ctx.Uniform1i(s.GetUniform(LightLocNumLights), s.activeLights)
	s.ctx.Uniform2f(s.GetUniform(LightLocScreenResolution), float32(fbW), float32(fbH))
	s.ctx.Uniform1f(s.GetUniform(LightLocAmbientLight), ambient)
	//s.ctx.Uniform1i(s.GetUniform(LightLocEnableShadows), s.shadows)
	//s.ctx.Uniform1i(s.GetUniform(LightLocVolumetricSteps), int32(s.cal.VolSteps))
	//s.ctx.Uniform1f(s.GetUniform(LightLocBeamRatioFactor), float32(s.cal.BeamRatio))
	s.ctx.Uniform1f(s.GetUniform(LightLocShininessWall), float32(s.cal.ShininessWall))
	s.ctx.Uniform1f(s.GetUniform(LightLocShininessFloor), float32(s.cal.ShininessFloor))
	s.ctx.Uniform1f(s.GetUniform(LightLocSpecBoostWall), float32(s.cal.SpecBoostWall))
	s.ctx.Uniform1f(s.GetUniform(LightLocSpecBoostFloor), float32(s.cal.SpecBoostFloor))
	s.ctx.Uniform1i(s.GetUniform(LightLocDebugLights), int32(s.debugLights))
	s.ctx.BindBufferBase(api.UNIFORM_BUFFER, 0, s.uboLights[s.frameIdx])
	if s.shadows != 0 {
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitRoomShadow)
		s.ctx.BindTexture(api.TEXTURE_2D, roomShadowTex)
	}
	renderGeometry()
}

// EnableShadows enables or disables shadow rendering based on the provided boolean value.
func (s *Lights) EnableShadows(e bool) {
	if e {
		s.shadows = 1
	} else {
		s.shadows = 0
	}
}

// GetActiveUBO returns the currently active UBO buffer for dynamic lights.
func (s *Lights) GetActiveUBO() uint32 {
	return s.uboLights[s.frameIdx]
}

// GetActiveLightsCount returns the number of active dynamic lights in the current frame.
func (s *Lights) GetActiveLightsCount() int32 {
	return s.activeLights
}

// ClusterID computes the linearized cluster header texel index for 3D grid cell (cx, cy, cz).
func ClusterID(cx, cy, cz int) int {
	return (cz*ClusterGridY+cy)*ClusterGridX + cx
}

// GetClusterState returns the last built clustered forward state and lookup texture ID.
func (s *Lights) GetClusterState() (tex uint32, near float32, logScale float32, globalCount int32) {
	return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
}

// GetClusterCell reads the packed (offset, count) entry for cluster (cx, cy, cz) from the CPU staging buffer.
func (s *Lights) GetClusterCell(cx, cy, cz int) (offset uint16, count uint16) {
	hOff := ClusterID(cx, cy, cz) * 4
	return s.clusterBuf[hOff], s.clusterBuf[hOff+1]
}

// GetClusterPoolEntry reads a packed light index from the CPU cluster index pool.
func (s *Lights) GetClusterPoolEntry(slot int) uint16 {
	return s.clusterBuf[clusterPoolBaseUint16+slot]
}

// PrepareClusters builds the 16x9x16 log-Z clustered forward light grid and uploads it into u_clusterData.
func (s *Lights) PrepareClusters() (tex uint32, near float32, logScale float32, globalCount int32) {
	if s.activeLights <= 0 {
		s.globalCount = 0
		if s.clusterNear < 0.5 {
			s.clusterNear = 0.5
		}
		s.clustersReady = true
		return s.clusterTex, s.clusterNear, 0, 0
	}
	if s.clustersReady {
		return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
	}
	proj := [16]float32{
		1, 0, 0, 0,
		0, 1, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}
	zNear := float32(1.0)
	zFar := float32(8192.0)
	if s.mapMetrics != nil {
		proj = s.mapMetrics.GetProj()
		zNear = s.mapMetrics.GetRoomZNear()
		zFar = s.mapMetrics.GetRoomZFar()
	}
	usedPool := s.PrepareClustersWithProj(proj, zNear, zFar)
	if s.clusterTex != 0 {
		poolTexels := (usedPool + 3) / 4
		poolRows := (poolTexels + ClusterTexWidth - 1) / ClusterTexWidth
		totalRows := clusterHeaderRows + poolRows
		if totalRows > ClusterTexHeight {
			totalRows = ClusterTexHeight
		}
		s.ctx.BindTexture(api.TEXTURE_2D, s.clusterTex)
		s.ctx.TexSubImage2D(api.TEXTURE_2D, 0, 0, 0, ClusterTexWidth, int32(totalRows), api.RGBA_INTEGER, api.UNSIGNED_SHORT, s.ctx.Ptr(s.clusterBuf))
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	s.clustersReady = true
	return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
}

// PrepareClustersWithProj bins active view-space lights into the 16x9x16 clustered grid using proj, zNear, zFar.
// Returns the total number of packed pool indices (global lights + per-cluster references) with zero heap allocations.
func (s *Lights) PrepareClustersWithProj(proj [16]float32, zNear, zFar float32) int {
	if zNear < 0.5 {
		zNear = 0.5
	}
	if zFar <= zNear+1.0 {
		zFar = zNear + 8192.0
	}
	s.clusterNear = zNear
	s.clusterLogScale = float32(float64(ClusterGridZ) / math.Log(float64(zFar/zNear)))

	clear(s.clusterCounts[:])

	n := int(s.activeLights)
	if n > 256 {
		n = 256
	}
	globalCount := int32(0)
	isPersp := math.Abs(float64(proj[11]+1.0)) < 1e-4 && math.Abs(float64(proj[0])) > 1e-6 && math.Abs(float64(proj[5])) > 1e-6

	for i := 0; i < n; i++ {
		off := i * lightFloatsPerEntry
		intensity := s.viewLights[off+7]
		if intensity <= 0.001 {
			continue
		}
		lightType := int(s.viewLights[off+3] + 0.5)
		effRadiusSq := s.viewLights[off+14]
		if lightType == 2 || effRadiusSq <= 0 || effRadiusSq > 1e12 {
			if int(globalCount) < clusterPoolMaxIndices {
				s.clusterBuf[clusterPoolBaseUint16+int(globalCount)] = uint16(i)
				globalCount++
			}
			continue
		}

		vx, vy, vz := s.viewLights[off], s.viewLights[off+1], s.viewLights[off+2]
		r := float32(math.Sqrt(float64(effRadiusSq)))
		dMin := -vz - r
		dMax := -vz + r
		if dMax < zNear || dMin > zFar {
			continue
		}
		dMinC := dMin
		if dMinC < zNear {
			dMinC = zNear
		}
		dMaxC := dMax
		if dMaxC > zFar {
			dMaxC = zFar
		}

		cz0 := int(math.Floor(math.Log(float64(dMinC/zNear)) * float64(s.clusterLogScale)))
		cz1 := int(math.Floor(math.Log(float64(dMaxC/zNear)) * float64(s.clusterLogScale)))
		if cz0 < 0 {
			cz0 = 0
		} else if cz0 >= ClusterGridZ {
			cz0 = ClusterGridZ - 1
		}
		if cz1 < cz0 {
			cz1 = cz0
		} else if cz1 >= ClusterGridZ {
			cz1 = ClusterGridZ - 1
		}

		var cx0, cx1, cy0, cy1 int
		if !isPersp || (vx*vx+vy*vy+vz*vz <= effRadiusSq) {
			cx0, cx1 = 0, ClusterGridX-1
			cy0, cy1 = 0, ClusterGridY-1
		} else {
			zA := -dMinC
			zB := -dMaxC
			xMin, xMax := vx-r, vx+r
			yMin, yMax := vy-r, vy+r

			minNdcX, maxNdcX := float32(1e9), float32(-1e9)
			for ix := 0; ix < 2; ix++ {
				xv := xMin
				if ix == 1 {
					xv = xMax
				}
				for iz := 0; iz < 2; iz++ {
					zv := zA
					if iz == 1 {
						zv = zB
					}
					ndcX := (proj[0]*xv + proj[8]*zv) / (proj[11] * zv)
					if ndcX < minNdcX {
						minNdcX = ndcX
					}
					if ndcX > maxNdcX {
						maxNdcX = ndcX
					}
				}
			}

			minNdcY, maxNdcY := float32(1e9), float32(-1e9)
			for iy := 0; iy < 2; iy++ {
				yv := yMin
				if iy == 1 {
					yv = yMax
				}
				for iz := 0; iz < 2; iz++ {
					zv := zA
					if iz == 1 {
						zv = zB
					}
					ndcY := (proj[5]*yv + proj[9]*zv) / (proj[11] * zv)
					if ndcY < minNdcY {
						minNdcY = ndcY
					}
					if ndcY > maxNdcY {
						maxNdcY = ndcY
					}
				}
			}

			if maxNdcX < -1.0 || minNdcX > 1.0 || maxNdcY < -1.0 || minNdcY > 1.0 {
				continue
			}

			cx0 = int(math.Floor(float64(minNdcX*0.5+0.5) * float64(ClusterGridX)))
			cx1 = int(math.Floor(float64(maxNdcX*0.5+0.5) * float64(ClusterGridX)))
			cy0 = int(math.Floor(float64(minNdcY*0.5+0.5) * float64(ClusterGridY)))
			cy1 = int(math.Floor(float64(maxNdcY*0.5+0.5) * float64(ClusterGridY)))

			if cx0 < 0 {
				cx0 = 0
			} else if cx0 >= ClusterGridX {
				cx0 = ClusterGridX - 1
			}
			if cx1 < cx0 {
				cx1 = cx0
			} else if cx1 >= ClusterGridX {
				cx1 = ClusterGridX - 1
			}
			if cy0 < 0 {
				cy0 = 0
			} else if cy0 >= ClusterGridY {
				cy0 = ClusterGridY - 1
			}
			if cy1 < cy0 {
				cy1 = cy0
			} else if cy1 >= ClusterGridY {
				cy1 = ClusterGridY - 1
			}
		}

		for cz := cz0; cz <= cz1; cz++ {
			for cy := cy0; cy <= cy1; cy++ {
				rowBase := (cz*ClusterGridY + cy) * ClusterGridX
				for cx := cx0; cx <= cx1; cx++ {
					cid := rowBase + cx
					cnt := s.clusterCounts[cid]
					if cnt < clusterMaxLightsPerBin {
						s.clusterCells[cid*clusterMaxLightsPerBin+int(cnt)] = uint16(i)
						s.clusterCounts[cid] = cnt + 1
					}
				}
			}
		}
	}

	s.globalCount = globalCount
	writeIdx := int(globalCount)
	for cid := 0; cid < ClusterCount; cid++ {
		cnt := int(s.clusterCounts[cid])
		avail := clusterPoolMaxIndices - writeIdx
		if cnt > avail {
			cnt = avail
		}
		hOff := cid * 4
		s.clusterBuf[hOff] = uint16(writeIdx)
		s.clusterBuf[hOff+1] = uint16(cnt)
		if cnt > 0 {
			cellBase := cid * clusterMaxLightsPerBin
			copy(s.clusterBuf[clusterPoolBaseUint16+writeIdx:], s.clusterCells[cellBase:cellBase+cnt])
			writeIdx += cnt
		}
	}
	return writeIdx
}


*/
