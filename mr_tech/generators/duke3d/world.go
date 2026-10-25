package duke3d

import (
	"fmt"
	"math"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

// World represents a 3D game world, managing level data, textures, scaling factors, and axis inversion for rendering.
type World struct {
	level        *lumps.Level
	textures     *lumps.Textures
	scaleSectorH float64
	scaleXY      float64
	flipY        bool
}

// NewWorld initializes and returns a new World instance with the provided level and textures for rendering and configuration.
func NewWorld(level *lumps.Level, textures *lumps.Textures) *World {
	return &World{
		level:        level,
		textures:     textures,
		scaleSectorH: 1.0,
		scaleXY:      1.0,
		flipY:        true,
	}
}

// SetScales updates scaling parameters for sector height, XY scaling, texture width/height, and vertical flip.
func (w *World) SetScales(scaleSectorH, scaleXY float64, flipY bool) {
	w.scaleSectorH = scaleSectorH
	w.scaleXY = scaleXY
	w.flipY = flipY
}

// computeHeight transforms a Build engine Z-coordinate to a scaled world height based on sector height scale.
func (w *World) computeHeight(k int32) float64 {
	// Build engine: 16 units of Z per 1 unit of X/Y.
	// Scaled to match 1 world unit = 16 Build X/Y units = 256 Build Z units:
	return -float64(k) / 256.0 * w.scaleSectorH
}

// computeLight calculates the light type, intensity, and falloff for a given sector based on its shading and height properties.
func (w *World) computeLight(sec *lumps.Sector, ceilY, floorY float64) (config.LightKind, float64, float64) {
	// Shade in Build: -128 (bright) to 127 (dark), 0 is normal.
	shade := float64(sec.CeilingShade+sec.FloorShade) * 0.5
	intensity := math.Max(1.2, math.Min(3.5, 2.5-shade/32.0))
	intensity *= 0.4

	roomH := math.Abs(ceilY - floorY)
	if roomH < 128 {
		roomH = 256
	}
	falloff := math.Max(1200.0, roomH*4.0)

	lightKind := config.LightKindAmbient
	if sec.IsSkyCeiling() {
		lightKind = config.LightKindOpenAir
	}

	return lightKind, intensity, falloff
}

// Build generates sector configurations, global vertex geometry, volumes, and light definitions for the loaded level.
func (w *World) Build() ([]*config.Sector, geometry.Polygon, []*config.Volume, []*config.Light, error) {
	if w.level == nil {
		return nil, nil, nil, nil, fmt.Errorf("duke3d: no level loaded")
	}

	configSectors := make([]*config.Sector, 0, len(w.level.Sectors))
	globalVertices := make(geometry.Polygon, 0, len(w.level.Walls)*2)
	var lights []*config.Light

	for secIdx, sec := range w.level.Sectors {
		sectorId := fmt.Sprintf("sector_%d", secIdx)

		floorY := w.computeHeight(sec.FloorZ)
		ceilY := w.computeHeight(sec.CeilingZ)

		lightKind, intensity, falloff := w.computeLight(&sec, ceilY, floorY)

		cSector := config.NewConfigSector(sectorId, intensity, lightKind, falloff)
		cSector.Tag = sectorId
		cSector.Light.R = 1.0
		cSector.Light.G = 0.98
		cSector.Light.B = 0.95
		cSector.FloorY = floorY
		cSector.CeilY = ceilY

		// Floor material
		floorTile := w.textures.GetTile(int(sec.FloorPicNum))
		floorKind := config.MaterialKindLoop
		if sec.IsSkyFloor() {
			floorKind = config.MaterialKindSky
			floorTile = w.textures.GetSkyTile(int(sec.FloorPicNum))
		}
		cSector.Floor = config.NewConfigMaterial([]string{floorTile}, floorKind, 1.0, 1.0, 0, 0)

		// Ceiling material
		ceilTile := w.textures.GetTile(int(sec.CeilingPicNum))
		ceilKind := config.MaterialKindLoop
		if sec.IsSkyCeiling() {
			ceilKind = config.MaterialKindSky
			ceilTile = w.textures.GetSkyTile(int(sec.CeilingPicNum))
		}
		cSector.Ceil = config.NewConfigMaterial([]string{ceilTile}, ceilKind, 1.0, 1.0, 0, 0)

		// First wall reference for UV alignment and slopes
		wallStart := int(sec.WallPtr)
		wallCount := int(sec.WallNum)
		if wallStart >= 0 && wallStart < len(w.level.Walls) {
			w1 := w.level.Walls[wallStart]
			p2 := int(w1.Point2)
			if p2 >= 0 && p2 < len(w.level.Walls) {
				w2 := w.level.Walls[p2]
				vFirst1 := geometry.XY{X: float64(w1.X) * w.scaleXY, Y: float64(w1.Y) * w.scaleXY}
				vFirst2 := geometry.XY{X: float64(w2.X) * w.scaleXY, Y: float64(w2.Y) * w.scaleXY}
				if w.flipY {
					vFirst1.Y = -vFirst1.Y
					vFirst2.Y = -vFirst2.Y
				}

				cSector.FloorAlign = &config.UVAlign{
					FixUV:     true,
					AlignWall: sec.AlignFirstWallFloor(),
					Start:     vFirst1,
					End:       vFirst2,
					SwapXY:    sec.SwapXYFloor(),
					FlipX:     sec.XFlipFloor(),
					FlipY:     sec.YFlipFloor(),
					Panning:   geometry.XY{X: float64(sec.FloorXPanning), Y: float64(sec.FloorYPanning)},
				}
				cSector.CeilAlign = &config.UVAlign{
					FixUV:     true,
					AlignWall: sec.AlignFirstWallCeiling(),
					Start:     vFirst1,
					End:       vFirst2,
					SwapXY:    sec.SwapXYCeiling(),
					FlipX:     sec.XFlipCeiling(),
					FlipY:     sec.YFlipCeiling(),
					Panning:   geometry.XY{X: float64(sec.CeilingXPanning), Y: float64(sec.CeilingYPanning)},
				}
			}
		}

		// Slopes
		if sec.IsSlopedFloor() {
			if grad, gx, gy, ok := w.level.ComputeFloorSlope(secIdx, w.scaleSectorH, w.scaleXY, w.flipY); ok {
				cSector.SlopeFloor.Gradient = grad
				cSector.SlopeFloor.G.X = gx
				cSector.SlopeFloor.G.Y = gy
			}
		}

		if sec.IsSlopedCeiling() {
			if grad, gx, gy, ok := w.level.ComputeCeilingSlope(secIdx, w.scaleSectorH, w.scaleXY, w.flipY); ok {
				cSector.SlopeCeiling.Gradient = grad
				cSector.SlopeCeiling.G.X = gx
				cSector.SlopeCeiling.G.Y = gy
			}
		}

		// Process walls belonging to this sector
		if wallStart >= 0 && wallStart+wallCount <= len(w.level.Walls) {
			cSector.Segments = make([]*config.Segment, 0, wallCount)

			for i := 0; i < wallCount; i++ {
				wIdx := wallStart + i
				wall := w.level.Walls[wIdx]

				p2Idx := int(wall.Point2)
				if p2Idx < 0 || p2Idx >= len(w.level.Walls) {
					continue
				}
				nextWall := w.level.Walls[p2Idx]

				v1 := geometry.XY{X: float64(wall.X) * w.scaleXY, Y: float64(wall.Y) * w.scaleXY}
				v2 := geometry.XY{X: float64(nextWall.X) * w.scaleXY, Y: float64(nextWall.Y) * w.scaleXY}
				if w.flipY {
					v1.Y = -v1.Y
					v2.Y = -v2.Y
				}

				globalVertices = append(globalVertices, v1, v2)

				cSeg := config.NewConfigSegment(sectorId, config.SegmentUnknown, v1, v2)

				cSeg.UVAlign = &config.UVAlign{
					FixUV:       true,
					Length:      math.Hypot(v2.X-v1.X, v2.Y-v1.Y),
					Panning:     geometry.XY{X: float64(wall.XPanning), Y: float64(wall.YPanning)},
					FlipX:       wall.HasHorizontalFlip(),
					FlipY:       wall.HasVerticalFlip(),
					AlignBottom: wall.IsBottomAligned(),
				}

				// First wall is the pivot reference for floor/ceiling slopes
				if i == 0 {
					if sec.IsSlopedFloor() {
						cSeg.SlopedFloorRef = true
					}
					if sec.IsSlopedCeiling() {
						cSeg.SlopedCeilingRef = true
					}
				}

				if wall.NextSector < 0 {
					// 1-sided solid wall
					cSeg.Kind = config.SegmentWall
					tile := w.textures.GetTile(int(wall.PicNum))
					cSeg.Middle = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
				} else {
					// 2-sided portal / adjoin
					cSeg.Kind = config.SegmentUnknown
					adjSec := w.level.GetSector(int(wall.NextSector))
					if adjSec != nil {
						adjCeilY := w.computeHeight(adjSec.CeilingZ)
						adjFloorY := w.computeHeight(adjSec.FloorZ)

						// Upper step wall
						if ceilY > adjCeilY || sec.IsSlopedCeiling() || adjSec.IsSlopedCeiling() {
							if sec.IsSkyCeiling() && adjSec.IsSkyCeiling() {
								cSeg.Upper = config.NewConfigMaterial(nil, config.MaterialKindSky, 1.0, 1.0, 0, 0)
							} else {
								tile := w.textures.GetTile(int(wall.PicNum))
								cSeg.Upper = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
							}
						}
						// Lower step wall
						if floorY < adjFloorY || sec.IsSlopedFloor() || adjSec.IsSlopedFloor() {
							if sec.IsSkyFloor() && adjSec.IsSkyFloor() {
								cSeg.Lower = config.NewConfigMaterial(nil, config.MaterialKindSky, 1.0, 1.0, 0, 0)
							} else {
								tile := w.textures.GetTile(int(wall.PicNum))
								cSeg.Lower = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
							}
						}
						// Masked / grating wall / one-way
						if wall.IsMasked() || wall.IsOneWay() {
							tilePic := int(wall.OverPicNum)
							if wall.IsOneWay() || tilePic < 0 {
								tilePic = int(wall.PicNum)
							}
							tile := w.textures.GetTile(tilePic)
							cSeg.Middle = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
						}
					}
				}

				cSector.Segments = append(cSector.Segments, cSeg)
			}
		}

		configSectors = append(configSectors, cSector)
	}

	volumes := w.buildSprites()

	return configSectors, globalVertices, volumes, lights, nil
}

// buildSprites converts sprite data from the level into 3D volumes with faces, materials, and geometry transformations.
func (w *World) buildSprites() []*config.Volume {
	sNum := w.level.GetNumSprite()
	if w.level == nil || sNum == 0 {
		return nil
	}

	volumes := make([]*config.Volume, 0, sNum)

	for spIdx := 0; spIdx < sNum; spIdx++ {
		sp := w.level.GetSprite(spIdx)
		if sp == nil {
			fmt.Println("nil sprite idx", spIdx)
			continue
		}
		switch sp.GetType() {
		case lumps.WallSprite:
			tileW, tileH, ok := w.textures.GetTileSize(int(sp.PicNum))
			if !ok || tileW <= 0 || tileH <= 0 {
				tileW, tileH = 64, 64
			}
			tileH *= 3
			angRad := ToRadians(sp.Ang)
			cosA := math.Cos(angRad)
			sinA := math.Sin(angRad)
			normalX := cosA
			normalY := sinA
			// Tangent along wall sprite (left to right when facing front)
			tangentX := sinA
			tangentY := -cosA
			widthBuild := float64(tileW) * float64(sp.XRepeat) / 4.0
			halfW := widthBuild / 2.0
			pLeftX := float64(sp.X) - tangentX*halfW
			pLeftY := float64(sp.Y) - tangentY*halfW
			pRightX := float64(sp.X) + tangentX*halfW
			pRightY := float64(sp.Y) + tangentY*halfW
			// Offset 1-sided decals slightly along normal to prevent Z-fighting with walls
			if !sp.IsTwoSided() {
				const decalOffset = 2.0
				pLeftX += normalX * decalOffset
				pLeftY += normalY * decalOffset
				pRightX += normalX * decalOffset
				pRightY += normalY * decalOffset
			}
			pLeft := geometry.XY{X: pLeftX * w.scaleXY, Y: pLeftY * w.scaleXY}
			pRight := geometry.XY{X: pRightX * w.scaleXY, Y: pRightY * w.scaleXY}
			if w.flipY {
				pLeft.Y = -pLeft.Y
				pRight.Y = -pRight.Y
			}
			heightBuildZ := int32(tileH) * int32(sp.YRepeat)
			var bottomZBuild, topZBuild int32
			if sp.HasVerticalAlignment() {
				bottomZBuild = sp.Z + heightBuildZ/2
				topZBuild = sp.Z - heightBuildZ/2
			} else { // Bottom-aligned
				bottomZBuild = sp.Z
				topZBuild = sp.Z - heightBuildZ
			}
			bottomZ := w.computeHeight(bottomZBuild)
			topZ := w.computeHeight(topZBuild)
			v0 := geometry.XYZ{X: pLeft.X, Y: pLeft.Y, Z: bottomZ}
			v1 := geometry.XYZ{X: pRight.X, Y: pRight.Y, Z: bottomZ}
			v2 := geometry.XYZ{X: pRight.X, Y: pRight.Y, Z: topZ}
			v3 := geometry.XYZ{X: pLeft.X, Y: pLeft.Y, Z: topZ}
			uMin, uMax := 0.0, 1.0
			if sp.HasHorizontalFlip() {
				uMin, uMax = 1.0, 0.0
			}
			vBottom, vTop := -1.0, 0.0
			if sp.HasVerticalFlip() {
				vBottom, vTop = 0.0, -1.0
			}
			tile := w.textures.GetTile(int(sp.PicNum))
			mat := config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
			vol := config.NewConfigVolume(fmt.Sprintf("sp_wall_%d", spIdx), "sp_wall")

			fPoints := []geometry.XYZ{v0, v1, v2, v3}
			uvs := [][2]float64{{uMin, vBottom}, {uMax, vBottom}, {uMax, vTop}, {uMin, vTop}}
			frontFace := config.NewConfigFace(fPoints, uvs, mat, fmt.Sprintf("sp_wall_%d_f", spIdx))
			vol.AddFace(frontFace)

			// If 2-sided (sp.CStat & 64 == 0), add back face
			if sp.IsTwoSided() {
				bPoints := []geometry.XYZ{v1, v0, v3, v2}
				backFace := config.NewConfigFace(bPoints, uvs, mat, fmt.Sprintf("sp_wall_%d_b", spIdx))
				vol.AddFace(backFace)
			}

			volumes = append(volumes, vol)

		case lumps.FloorSprite:
			tileW, tileH, ok := w.textures.GetTileSize(int(sp.PicNum))
			if !ok || tileW <= 0 || tileH <= 0 {
				tileW, tileH = 64, 64
			}
			angRad := ToRadians(sp.Ang)
			cosA := math.Cos(angRad)
			sinA := math.Sin(angRad)

			dirX := cosA
			dirY := sinA
			perpX := sinA
			perpY := -cosA

			halfW := float64(tileW) * float64(sp.XRepeat) / 8.0
			halfH := float64(tileH) * float64(sp.YRepeat) / 8.0

			c0X := float64(sp.X) - perpX*halfW - dirX*halfH
			c0Y := float64(sp.Y) - perpY*halfW - dirY*halfH
			c1X := float64(sp.X) + perpX*halfW - dirX*halfH
			c1Y := float64(sp.Y) + perpY*halfW - dirY*halfH
			c2X := float64(sp.X) + perpX*halfW + dirX*halfH
			c2Y := float64(sp.Y) + perpY*halfW + dirY*halfH
			c3X := float64(sp.X) - perpX*halfW + dirX*halfH
			c3Y := float64(sp.Y) - perpY*halfW + dirY*halfH

			pt0 := geometry.XY{X: c0X * w.scaleXY, Y: c0Y * w.scaleXY}
			pt1 := geometry.XY{X: c1X * w.scaleXY, Y: c1Y * w.scaleXY}
			pt2 := geometry.XY{X: c2X * w.scaleXY, Y: c2Y * w.scaleXY}
			pt3 := geometry.XY{X: c3X * w.scaleXY, Y: c3Y * w.scaleXY}
			if w.flipY {
				pt0.Y = -pt0.Y
				pt1.Y = -pt1.Y
				pt2.Y = -pt2.Y
				pt3.Y = -pt3.Y
			}

			zVal := w.computeHeight(sp.Z) + 0.1

			v0 := geometry.XYZ{X: pt0.X, Y: pt0.Y, Z: zVal}
			v1 := geometry.XYZ{X: pt1.X, Y: pt1.Y, Z: zVal}
			v2 := geometry.XYZ{X: pt2.X, Y: pt2.Y, Z: zVal}
			v3 := geometry.XYZ{X: pt3.X, Y: pt3.Y, Z: zVal}

			uMin, uMax := 0.0, 1.0
			if sp.CStat&4 != 0 {
				uMin, uMax = 1.0, 0.0
			}
			vMin, vMax := -1.0, 0.0
			if sp.CStat&8 != 0 {
				vMin, vMax = 0.0, -1.0
			}

			tile := w.textures.GetTile(int(sp.PicNum))
			mat := config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
			vol := config.NewConfigVolume(fmt.Sprintf("sp_floor_%d", spIdx), "sp_floor")
			tPoints := []geometry.XYZ{v0, v1, v2, v3}
			uvs := [][2]float64{{uMin, vMin}, {uMax, vMin}, {uMax, vMax}, {uMin, vMax}}
			topFace := config.NewConfigFace(tPoints, uvs, mat, fmt.Sprintf("sp_floor_%d_t", spIdx))
			vol.AddFace(topFace)
			volumes = append(volumes, vol)
		}
	}

	return volumes
}
