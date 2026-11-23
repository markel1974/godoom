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
	ctx                api.IContext
	prg                uint32
	table              [LightLocLast]int32
	uboLights          [lightsDoubleBuffer]uint32
	viewLights         []float32
	viewLightsPtr      unsafe.Pointer
	frameIdx           int
	activeLights       int32
	shadows            int32
	stride             int32
	handler            *Handler
	cal                *model.Calibration
	main               *Main
	depth              *Depth
	mapMetrics         *metrics.Map
	debugLights        int
	clusterTex         uint32
	clusterBuf         []uint16
	clusterCounts      [ClusterCount]uint16
	clusterActive      [ClusterCount]uint16
	clusterActiveCount int
	clusterCells       []uint16
	clusterNear        float32
	clusterLogScale    float32
	globalCount        int32
	clustersReady      bool
	heap               [maxUBOLights]lightCandidate
}

const lightFloatsPerEntry = 16

type lightBlock [lightFloatsPerEntry]float32

type lightCandidate struct {
	idx      int32
	score    float32
	vx       float32
	vy       float32
	vz       float32
	effSq    float32
	invScale float32
}

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

	for i := range lightsDoubleBuffer {
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

	srcCount := int(numLights)
	outCount := min(srcCount, maxUBOLights)
	neededFloats := outCount * lightFloatsPerEntry
	if neededFloats > len(s.viewLights) {
		s.viewLights = make([]float32, neededFloats)
		s.viewLightsPtr = unsafe.Pointer(&s.viewLights[0])
	}

	src := unsafe.Slice((*lightBlock)(frameLights), srcCount)
	dst := unsafe.Slice((*lightBlock)(s.viewLightsPtr), outCount)

	vm := s.mapMetrics.GetViewMatrix()
	m0, m1, m2 := vm[0], vm[1], vm[2]
	m4, m5, m6 := vm[4], vm[5], vm[6]
	m8, m9, m10 := vm[8], vm[9], vm[10]
	m12, m13, m14 := vm[12], vm[13], vm[14]

	uploadBytes := int(lightSize)
	if srcCount <= maxUBOLights {
		for i := range srcCount {
			sp := &src[i]
			dp := &dst[i]
			*dp = *sp

			x, y, z := sp[0], sp[1], sp[2]
			dp[0] = m0*x + m4*y + m8*z + m12
			dp[1] = m1*x + m5*y + m9*z + m13
			dp[2] = m2*x + m6*y + m10*z + m14

			lType := int(sp[3] + 0.5)
			if lType == 1 || lType == 2 {
				dx, dy, dz := sp[8], sp[9], sp[10]
				vx := m0*dx + m4*dy + m8*dz
				vy := m1*dx + m5*dy + m9*dz
				vz := m2*dx + m6*dy + m10*dz
				lenSq := vx*vx + vy*vy + vz*vz
				if lenSq > 1e-12 {
					invLen := float32(1.0 / math.Sqrt(float64(lenSq)))
					dp[8] = vx * invLen
					dp[9] = vy * invLen
					dp[10] = vz * invLen
				} else {
					dp[8], dp[9], dp[10] = vx, vy, vz
				}
			}

			intensity := sp[7]
			if intensity < 0.1 {
				intensity = 0.1
			}
			falloff := sp[11]
			if falloff < 1.0 {
				falloff = 1.0
			}
			scale := falloff * intensity
			effRadius := float32(4.605) * scale
			dp[14] = effRadius * effRadius
			dp[15] = float32(1.0) / scale
		}
	} else {
		heapLen := 0
		for i := range srcCount {
			sp := &src[i]
			rawInt := sp[7]
			if rawInt <= 0.001 {
				continue
			}

			x, y, z := sp[0], sp[1], sp[2]
			vx := m0*x + m4*y + m8*z + m12
			vy := m1*x + m5*y + m9*z + m13
			vz := m2*x + m6*y + m10*z + m14

			effInt := rawInt
			if effInt < 0.1 {
				effInt = 0.1
			}
			falloff := sp[11]
			if falloff < 1.0 {
				falloff = 1.0
			}
			scale := falloff * effInt
			effRadius := float32(4.605) * scale
			effSq := effRadius * effRadius
			invScale := float32(1.0) / scale

			var score float32
			if int(sp[3]+0.5) == 2 {
				score = -1e30
			} else {
				score = (vx*vx + vy*vy + vz*vz) - effSq
			}

			if heapLen < maxUBOLights {
				ptr := &s.heap[heapLen]
				ptr.idx = int32(i)
				ptr.score = score
				ptr.vx = vx
				ptr.vy = vy
				ptr.vz = vz
				ptr.effSq = effSq
				ptr.invScale = invScale
				heapLen++
				if heapLen == maxUBOLights {
					for p := maxUBOLights/2 - 1; p >= 0; p-- {
						siftDownLightMaxHeap(&s.heap, p, maxUBOLights)
					}
				}
			} else if score < s.heap[0].score {
				ptr := &s.heap[0]
				ptr.idx = int32(i)
				ptr.score = score
				ptr.vx = vx
				ptr.vy = vy
				ptr.vz = vz
				ptr.effSq = effSq
				ptr.invScale = invScale
				siftDownLightMaxHeap(&s.heap, 0, maxUBOLights)
			}
		}

		if heapLen == 0 {
			s.activeLights = 0
			s.globalCount = 0
			return
		}
		s.activeLights = int32(heapLen)
		uploadBytes = heapLen * lightFloatsPerEntry * 4

		for k := 0; k < heapLen; k++ {
			c := &s.heap[k]
			sp := &src[c.idx]
			dp := &dst[k]
			*dp = *sp
			dp[0], dp[1], dp[2] = c.vx, c.vy, c.vz
			dp[14], dp[15] = c.effSq, c.invScale
			lType := int(sp[3] + 0.5)
			if lType == 1 || lType == 2 {
				dx, dy, dz := sp[8], sp[9], sp[10]
				dvx := m0*dx + m4*dy + m8*dz
				dvy := m1*dx + m5*dy + m9*dz
				dvz := m2*dx + m6*dy + m10*dz
				lenSq := dvx*dvx + dvy*dvy + dvz*dvz
				if lenSq > 1e-12 {
					invLen := float32(1.0 / math.Sqrt(float64(lenSq)))
					dp[8] = dvx * invLen
					dp[9] = dvy * invLen
					dp[10] = dvz * invLen
				} else {
					dp[8], dp[9], dp[10] = dvx, dvy, dvz
				}
			}
		}
	}

	s.ctx.BindBuffer(api.UNIFORM_BUFFER, s.uboLights[s.frameIdx])
	// Scrittura asincrona garantita sull'UBO inattivo
	s.ctx.BufferSubData(api.UNIFORM_BUFFER, 0, uploadBytes, s.viewLightsPtr)
	s.ctx.BindBuffer(api.UNIFORM_BUFFER, 0)
}

func siftDownLightMaxHeap(heap *[maxUBOLights]lightCandidate, i, n int) {
	item := heap[i]
	for {
		left := 2*i + 1
		if left >= n {
			break
		}
		largest := left
		right := left + 1
		if right < n && heap[right].score > heap[left].score {
			largest = right
		}
		if heap[largest].score <= item.score {
			break
		}
		heap[i] = heap[largest]
		i = largest
	}
	heap[i] = item
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

	proj := s.mapMetrics.GetProjMatrix()
	zNear := s.mapMetrics.GetRoomZNear()
	zFar := s.mapMetrics.GetRoomZFar()

	usedPool := s.PrepareClustersWithProjPtr(proj, zNear, zFar)
	if s.clusterTex != 0 {
		poolTexels := (usedPool + 3) / 4
		poolRows := (poolTexels + ClusterTexWidth - 1) / ClusterTexWidth
		totalRows := min(clusterHeaderRows+poolRows, ClusterTexHeight)
		s.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitClusterData)
		s.ctx.BindTexture(api.TEXTURE_2D, s.clusterTex)
		needed := min(totalRows*ClusterTexWidth*4, len(s.clusterBuf))
		s.ctx.TexSubImage2D(api.TEXTURE_2D, 0, 0, 0, ClusterTexWidth, int32(totalRows), api.RGBA_INTEGER, api.UNSIGNED_SHORT, s.ctx.Ptr(s.clusterBuf[:needed]))
		s.ctx.BindTexture(api.TEXTURE_2D, 0)
	}
	s.clustersReady = true
	return s.clusterTex, s.clusterNear, s.clusterLogScale, s.globalCount
}

// PrepareClustersWithProj bins active view-space lights into the 16x9x16 clustered grid using proj, zNear, zFar.
// Returns the total number of packed pool indices (global lights + per-cluster references) with zero heap allocations.
func (s *Lights) PrepareClustersWithProj(proj [16]float32, zNear, zFar float32) int {
	return s.PrepareClustersWithProjPtr(&proj, zNear, zFar)
}

// PrepareClustersWithProjPtr bins active view-space lights into the 16x9x16 clustered grid without copying proj.
func (s *Lights) PrepareClustersWithProjPtr(proj *[16]float32, zNear, zFar float32) int {
	if zNear < 0.5 {
		zNear = 0.5
	}
	if zFar <= zNear+1.0 {
		zFar = zNear + 8192.0
	}
	s.clusterNear = zNear
	s.clusterLogScale = float32(float64(ClusterGridZ) / math.Log(float64(zFar/zNear)))
	logNear := math.Log(float64(zNear))

	for k := 0; k < s.clusterActiveCount; k++ {
		hOff := int(s.clusterActive[k]) * 4
		s.clusterBuf[hOff] = 0
		s.clusterBuf[hOff+1] = 0
		s.clusterBuf[hOff+2] = 0
	}
	s.clusterActiveCount = 0

	n := min(int(s.activeLights), maxUBOLights)
	if n <= 0 {
		s.globalCount = 0
		return 0
	}

	activeCount := 0
	globalCount := int32(0)
	p0, p5, p8, p9, p11 := proj[0], proj[5], proj[8], proj[9], proj[11]
	isPersp := math.Abs(float64(p11+1.0)) < 1e-4 && math.Abs(float64(p0)) > 1e-6 && math.Abs(float64(p5)) > 1e-6
	isSymPersp := isPersp && p8 == 0 && p9 == 0

	viewEntries := unsafe.Slice((*lightBlock)(s.viewLightsPtr), n)
	for i := range n {
		vb := &viewEntries[i]
		intensity := vb[7]
		if intensity <= 0.001 {
			continue
		}
		lightType := int(vb[3] + 0.5)
		effRadiusSq := vb[14]
		if lightType == 2 || effRadiusSq <= 0 || effRadiusSq > 1e12 {
			if int(globalCount) < clusterPoolMaxIndices {
				s.clusterBuf[clusterPoolBaseUint16+int(globalCount)] = uint16(i)
				globalCount++
			}
			continue
		}

		vx, vy, vz := vb[0], vb[1], vb[2]
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

		cz0 := int(math.Floor((math.Log(float64(dMinC)) - logNear) * float64(s.clusterLogScale)))
		cz1 := int(math.Floor((math.Log(float64(dMaxC)) - logNear) * float64(s.clusterLogScale)))
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
			xMin, xMax := vx-r, vx+r
			yMin, yMax := vy-r, vy+r

			var minNdcX, maxNdcX, minNdcY, maxNdcY float32
			if isSymPersp {
				invZMin := float32(1.0) / dMinC
				invZMax := float32(1.0) / dMaxC
				if p0 >= 0 {
					if xMin < 0 {
						minNdcX = p0 * xMin * invZMin
					} else {
						minNdcX = p0 * xMin * invZMax
					}
					if xMax > 0 {
						maxNdcX = p0 * xMax * invZMin
					} else {
						maxNdcX = p0 * xMax * invZMax
					}
				} else {
					if xMax > 0 {
						minNdcX = p0 * xMax * invZMin
					} else {
						minNdcX = p0 * xMax * invZMax
					}
					if xMin < 0 {
						maxNdcX = p0 * xMin * invZMin
					} else {
						maxNdcX = p0 * xMin * invZMax
					}
				}
				if p5 >= 0 {
					if yMin < 0 {
						minNdcY = p5 * yMin * invZMin
					} else {
						minNdcY = p5 * yMin * invZMax
					}
					if yMax > 0 {
						maxNdcY = p5 * yMax * invZMin
					} else {
						maxNdcY = p5 * yMax * invZMax
					}
				} else {
					if yMax > 0 {
						minNdcY = p5 * yMax * invZMin
					} else {
						minNdcY = p5 * yMax * invZMax
					}
					if yMin < 0 {
						maxNdcY = p5 * yMin * invZMin
					} else {
						maxNdcY = p5 * yMin * invZMax
					}
				}
			} else {
				zA := -dMinC
				zB := -dMaxC

				minNdcX, maxNdcX = float32(1e9), float32(-1e9)
				for ix := range 2 {
					xv := xMin
					if ix == 1 {
						xv = xMax
					}
					for iz := range 2 {
						zv := zA
						if iz == 1 {
							zv = zB
						}
						ndcX := (p0*xv + p8*zv) / (p11 * zv)
						if ndcX < minNdcX {
							minNdcX = ndcX
						}
						if ndcX > maxNdcX {
							maxNdcX = ndcX
						}
					}
				}

				minNdcY, maxNdcY = float32(1e9), float32(-1e9)
				for iy := range 2 {
					yv := yMin
					if iy == 1 {
						yv = yMax
					}
					for iz := range 2 {
						zv := zA
						if iz == 1 {
							zv = zB
						}
						ndcY := (p5*yv + p9*zv) / (p11 * zv)
						if ndcY < minNdcY {
							minNdcY = ndcY
						}
						if ndcY > maxNdcY {
							maxNdcY = ndcY
						}
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
					if cnt == 0 && activeCount < ClusterCount {
						s.clusterActive[activeCount] = uint16(cid)
						activeCount++
					}
					if cnt < clusterMaxLightsPerBin {
						s.clusterCells[cid*clusterMaxLightsPerBin+int(cnt)] = uint16(i)
						s.clusterCounts[cid] = cnt + 1
					}
				}
			}
		}
	}

	s.clusterActiveCount = activeCount
	s.globalCount = globalCount
	writeIdx := int(globalCount)
	for k := 0; k < activeCount; k++ {
		cid := int(s.clusterActive[k])
		cnt := int(s.clusterCounts[cid])
		s.clusterCounts[cid] = 0
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
