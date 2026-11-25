package metrics

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/model"
)

// SunCascadeCount defines the number of orthographic shadow cascades for outdoor directional sun lighting.
const SunCascadeCount = 4

// SunCSM computes and caches texel-snapped Cascaded Shadow Map matrices and split distances for outdoor sunlight.
type SunCSM struct {
	enabled bool

	// OpenGL Y-up normalized direction pointing FROM sun TOWARD world (light rays).
	dirX, dirY, dirZ float32

	colorR, colorG, colorB float32
	intensity              float32
	skyR, skyG, skyB       float32
	groundR, groundG       float32
	groundB                float32

	shadowDist float32
	lambda     float32
	tileRes    float32

	splitsView      [SunCascadeCount]float32
	texelWorld      [SunCascadeCount]float32
	depthBiasScale  [SunCascadeCount]float32
	cascadeSpace    [SunCascadeCount][16]float32
	cascadeSpacePtr [SunCascadeCount]*float32
}

// NewSunCSM initializes a SunCSM controller with outdoor daylight defaults.
func NewSunCSM() *SunCSM {
	s := &SunCSM{
		dirX:       0.35,
		dirY:       -0.82,
		dirZ:       0.45,
		colorR:     1.0,
		colorG:     0.96,
		colorB:     0.88,
		intensity:  2.5,
		skyR:       0.42,
		skyG:       0.62,
		skyB:       0.92,
		groundR:    0.22,
		groundG:    0.21,
		groundB:    0.19,
		shadowDist: 0,
		lambda:     0.78,
		tileRes:    512.0,
	}
	s.dirX, s.dirY, s.dirZ = normalizeSafe(s.dirX, s.dirY, s.dirZ, 0.35, -0.82, 0.45)
	for i := range SunCascadeCount {
		s.cascadeSpacePtr[i] = &s.cascadeSpace[i][0]
	}
	return s
}

// ConfigureFromCalibration applies sun and CSM configuration from model.Calibration.
func (s *SunCSM) ConfigureFromCalibration(cal *model.Calibration) {
	if cal == nil {
		return
	}
	s.enabled = cal.EnableSunCSM
	if cal.SunDirX != 0 || cal.SunDirY != 0 || cal.SunDirZ != 0 {
		s.SetDirectionEngine(float32(cal.SunDirX), float32(cal.SunDirY), float32(cal.SunDirZ))
	}
	if cal.SunColor[0] > 0 || cal.SunColor[1] > 0 || cal.SunColor[2] > 0 {
		s.colorR = float32(cal.SunColor[0])
		s.colorG = float32(cal.SunColor[1])
		s.colorB = float32(cal.SunColor[2])
	}
	if cal.SunIntensity > 0 {
		s.intensity = float32(cal.SunIntensity)
	}
	if cal.SunSkyColor[0] > 0 || cal.SunSkyColor[1] > 0 || cal.SunSkyColor[2] > 0 {
		s.skyR = float32(cal.SunSkyColor[0])
		s.skyG = float32(cal.SunSkyColor[1])
		s.skyB = float32(cal.SunSkyColor[2])
	}
	if cal.SunGroundColor[0] > 0 || cal.SunGroundColor[1] > 0 || cal.SunGroundColor[2] > 0 {
		s.groundR = float32(cal.SunGroundColor[0])
		s.groundG = float32(cal.SunGroundColor[1])
		s.groundB = float32(cal.SunGroundColor[2])
	}
	if cal.SunShadowDistance > 0 {
		s.shadowDist = float32(cal.SunShadowDistance)
	} else {
		s.shadowDist = 0
	}
	if cal.SunCascadeLambda > 0 && cal.SunCascadeLambda <= 1.0 {
		s.lambda = float32(cal.SunCascadeLambda)
	}
}

// SetEnabled enables or disables directional Sun + CSM shadow calculation.
func (s *SunCSM) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// IsEnabled reports whether directional Sun + CSM shadows are enabled.
func (s *SunCSM) IsEnabled() bool {
	return s.enabled
}

// SetDirectionEngine updates the sun ray direction given engine Z-up coordinates (shining from sun toward ground).
func (s *SunCSM) SetDirectionEngine(ex, ey, ez float32) {
	s.SetDirectionGL(ex, ez, -ey)
}

// SetDirectionGL updates the sun ray direction directly in OpenGL Y-up world coordinates.
func (s *SunCSM) SetDirectionGL(gx, gy, gz float32) {
	s.dirX, s.dirY, s.dirZ = normalizeSafe(gx, gy, gz, 0.35, -0.82, 0.45)
}

// GetDirectionGL returns the normalized light travel direction in OpenGL Y-up world coordinates.
func (s *SunCSM) GetDirectionGL() (float32, float32, float32) {
	return s.dirX, s.dirY, s.dirZ
}

// GetToSunGL returns the normalized vector pointing toward the sun in OpenGL Y-up world coordinates.
func (s *SunCSM) GetToSunGL() (float32, float32, float32) {
	return -s.dirX, -s.dirY, -s.dirZ
}

// GetDirectionView transforms the normalized sun travel direction into camera view space.
func (s *SunCSM) GetDirectionView(m *Map) (float32, float32, float32) {
	if m == nil {
		return s.dirX, s.dirY, s.dirZ
	}
	return m.TransformDirectionNormalizedToView(s.dirX, s.dirY, s.dirZ)
}

// SetColor updates the sun RGB radiance color and multiplier intensity.
func (s *SunCSM) SetColor(r, g, b, intensity float32) {
	s.colorR, s.colorG, s.colorB = r, g, b
	if intensity < 0 {
		intensity = 0
	}
	s.intensity = intensity
}

// GetColorIntensity returns linear sun RGB color and scalar intensity.
func (s *SunCSM) GetColorIntensity() (r, g, b, intensity float32) {
	return s.colorR, s.colorG, s.colorB, s.intensity
}

// SetSkyAmbient updates hemisphere ambient sky zenith and ground bounce colors.
func (s *SunCSM) SetSkyAmbient(skyR, skyG, skyB, groundR, groundG, groundB float32) {
	s.skyR, s.skyG, s.skyB = skyR, skyG, skyB
	s.groundR, s.groundG, s.groundB = groundR, groundG, groundB
}

// GetSkyAmbient returns linear hemisphere sky zenith and ground bounce colors.
func (s *SunCSM) GetSkyAmbient() (skyR, skyG, skyB, groundR, groundG, groundB float32) {
	return s.skyR, s.skyG, s.skyB, s.groundR, s.groundG, s.groundB
}

// SetShadowDistance updates maximum cascade distance (0 auto-derives from camera zFar).
func (s *SunCSM) SetShadowDistance(dist float32) {
	s.shadowDist = dist
}

// SetTileResolution updates the cascade shadow tile resolution in pixels used for texel snapping.
func (s *SunCSM) SetTileResolution(res float32) {
	if res >= 32 {
		s.tileRes = res
	}
}

// GetCascadeSplitsView returns the view-space far split distances for all 4 cascades.
func (s *SunCSM) GetCascadeSplitsView() [SunCascadeCount]float32 {
	return s.splitsView
}

// GetCascadeTexelWorld returns world-space texel size per cascade.
func (s *SunCSM) GetCascadeTexelWorld() [SunCascadeCount]float32 {
	return s.texelWorld
}

// GetCascadeDepthBias returns normalized [0..1] depth bias base per cascade scaled by (texel / depthSpan).
func (s *SunCSM) GetCascadeDepthBias() [SunCascadeCount]float32 {
	return s.depthBiasScale
}

// GetCascadeMatrix returns a copy of the column-major ortho light-space matrix for cascade idx.
func (s *SunCSM) GetCascadeMatrix(idx int) [16]float32 {
	if idx < 0 || idx >= SunCascadeCount {
		return [16]float32{}
	}
	return s.cascadeSpace[idx]
}

// GetCascadeMatrixPtr returns a pointer to the column-major ortho light-space matrix for cascade idx.
func (s *SunCSM) GetCascadeMatrixPtr(idx int) *float32 {
	if idx < 0 || idx >= SunCascadeCount {
		return nil
	}
	return s.cascadeSpacePtr[idx]
}

// Update recomputes PSSM cascade splits and texel-snapped orthographic light matrices from camera metrics.
func (s *SunCSM) Update(m *Map, scaleX, scaleY, zNear, zFar float32) {
	if !s.enabled || m == nil {
		return
	}
	n := zNear
	if n < 0.1 {
		n = 0.1
	}
	fMax := zFar
	if fMax <= n+1.0 {
		fMax = n + 4096.0
	}
	shadowFar := s.shadowDist
	if shadowFar <= 0 {
		shadowFar = fMax
		if shadowFar > 3500.0 {
			shadowFar = 3500.0
		}
	}
	if shadowFar > fMax {
		shadowFar = fMax
	}
	if shadowFar < n+32.0 {
		shadowFar = n + 32.0
	}

	logNear := shadowFar / 200.0
	if logNear < n {
		logNear = n
	}
	if logNear > shadowFar*0.5 {
		logNear = shadowFar * 0.5
	}
	lambda := s.lambda
	if lambda < 0 {
		lambda = 0
	} else if lambda > 1 {
		lambda = 1
	}

	ratio := float64(shadowFar / logNear)
	span := float64(shadowFar - logNear)
	for c := range SunCascadeCount {
		p := float64(c+1) / float64(SunCascadeCount)
		dLog := float64(logNear) * math.Pow(ratio, p)
		dUni := float64(logNear) + span*p
		split := float32(float64(lambda)*dLog + (1.0-float64(lambda))*dUni)
		if c == SunCascadeCount-1 {
			split = shadowFar
		}
		s.splitsView[c] = split
	}

	ffX, ffY, ffZ := s.dirX, s.dirY, s.dirZ
	upX, upY, upZ := float32(0), float32(1), float32(0)
	if math.Abs(float64(ffY)) > 0.99 {
		upX, upY, upZ = 0, 0, -1
	}
	cRightX, cRightY, cRightZ := crossProduct(ffX, ffY, ffZ, upX, upY, upZ)
	rrX, rrY, rrZ := normalizeSafe(cRightX, cRightY, cRightZ, 1, 0, 0)
	uuX, uuY, uuZ := crossProduct(rrX, rrY, rrZ, ffX, ffY, ffZ)

	sx := scaleX
	if math.Abs(float64(sx)) < 1e-4 {
		sx = 1.0
	}
	sy := scaleY
	if math.Abs(float64(sy)) < 1e-4 {
		sy = 1.0
	}
	invSX := 1.0 / sx
	invSY := 1.0 / sy
	kSq := invSX*invSX + invSY*invSY

	tileRes := s.tileRes
	if tileRes < 32 {
		tileRes = 512.0
	}

	inv := m.GetInvViewMatrix()
	prevSplit := n
	for c := range SunCascadeCount {
		sliceNear := prevSplit
		sliceFar := s.splitsView[c]
		if sliceFar <= sliceNear {
			sliceFar = sliceNear + 1.0
		}
		prevSplit = sliceFar

		cDepth := float32(0.5) * (sliceNear + sliceFar) * (1.0 + kSq)
		if cDepth > sliceFar {
			cDepth = sliceFar
		} else if cDepth < sliceNear {
			cDepth = sliceNear
		}

		dxFar := sliceFar - cDepth
		rFarSq := sliceFar*sliceFar*kSq + dxFar*dxFar
		dxNear := cDepth - sliceNear
		rNearSq := sliceNear*sliceNear*kSq + dxNear*dxNear
		rSq := rFarSq
		if rNearSq > rSq {
			rSq = rNearSq
		}

		// Scale slightly above analytical sphere radius so 1-texel snapping never clips slice corners.
		radius := float32(math.Ceil(math.Sqrt(float64(rSq))*1.01 + 4.0))
		texel := (2.0 * radius) / tileRes
		s.texelWorld[c] = texel

		wcX := -cDepth*inv[8] + inv[12]
		wcY := -cDepth*inv[9] + inv[13]
		wcZ := -cDepth*inv[10] + inv[14]

		cxLight := dotProduct(wcX, wcY, wcZ, rrX, rrY, rrZ)
		cyLight := dotProduct(wcX, wcY, wcZ, uuX, uuY, uuZ)
		czLight := dotProduct(wcX, wcY, wcZ, ffX, ffY, ffZ)

		cxSnap := float32(math.Floor(float64(cxLight/texel))) * texel
		cySnap := float32(math.Floor(float64(cyLight/texel))) * texel
		czSnap := float32(math.Floor(float64(czLight/texel))) * texel

		zBack := radius * 3.0
		if zBack < shadowFar*1.5 {
			zBack = shadowFar * 1.5
		}
		if zBack < 4096.0 {
			zBack = 4096.0
		}
		orthoNear := float32(1.0)
		orthoFar := zBack + radius*2.0
		depthSpan := orthoFar - orthoNear
		invDepthSpan := 1.0 / depthSpan
		s.depthBiasScale[c] = texel * invDepthSpan

		mView := [16]float32{
			rrX, uuX, -ffX, 0,
			rrY, uuY, -ffY, 0,
			rrZ, uuZ, -ffZ, 0,
			-cxSnap, -cySnap, czSnap - zBack, 1,
		}
		mProj := [16]float32{
			1.0 / radius, 0, 0, 0,
			0, 1.0 / radius, 0, 0,
			0, 0, -2.0 * invDepthSpan, 0,
			0, 0, -(orthoFar + orthoNear) * invDepthSpan, 1,
		}
		MatrixMultiply4x4Ptr(s.cascadeSpacePtr[c], &mProj[0], &mView[0])
	}
}

func normalizeSafe(x, y, z, fbX, fbY, fbZ float32) (float32, float32, float32) {
	lenSq := x*x + y*y + z*z
	if lenSq <= 1e-12 {
		return fbX, fbY, fbZ
	}
	invLen := float32(1.0 / math.Sqrt(float64(lenSq)))
	return x * invLen, y * invLen, z * invLen
}
