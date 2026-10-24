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

// Build generates sector configurations, global vertex geometry, and light definitions for the loaded level.
func (w *World) Build() ([]*config.Sector, geometry.Polygon, []*config.Light, error) {
	if w.level == nil {
		return nil, nil, nil, fmt.Errorf("duke3d: no level loaded")
	}

	configSectors := make([]*config.Sector, 0, len(w.level.Sectors))
	globalVertices := make(geometry.Polygon, 0, len(w.level.Walls)*2)
	var lights []*config.Light

	for secIdx, sec := range w.level.Sectors {
		sectorId := fmt.Sprintf("sector_%d", secIdx)

		// Build engine: 16 units of Z per 1 unit of X/Y.
		// Scaled to match 1 world unit = 16 Build X/Y units = 256 Build Z units:
		floorY := -float64(sec.FloorZ) / 256.0 * w.scaleSectorH
		ceilY := -float64(sec.CeilingZ) / 256.0 * w.scaleSectorH

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

		cSector := config.NewConfigSector(sectorId, intensity, lightKind, falloff)
		cSector.Tag = sectorId
		cSector.Light.R = 1.0
		cSector.Light.G = 0.98
		cSector.Light.B = 0.95
		cSector.FloorY = floorY
		cSector.CeilY = ceilY

		// Floor material
		floorTile := w.textures.GetTile(int(sec.FloorPicNum))
		cSector.Floor = config.NewConfigMaterial([]string{floorTile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)

		// Ceiling material
		ceilTile := w.textures.GetTile(int(sec.CeilingPicNum))
		ceilKind := config.MaterialKindLoop
		if sec.IsSkyCeiling() {
			ceilKind = config.MaterialKindSky
		}
		cSector.Ceil = config.NewConfigMaterial([]string{ceilTile}, ceilKind, 1.0, 1.0, 0, 0)

		// Slopes
		if sec.IsSlopedFloor() {
			if grad, gx, gy, ok := w.level.ComputeFloorSlope(secIdx, w.scaleSectorH, w.scaleXY, w.flipY); ok {
				cSector.SlopedFloorGradient = grad
				cSector.SlopedFloorGX = gx
				cSector.SlopedFloorGY = gy
			}
		}

		if sec.IsSlopedCeiling() {
			if grad, gx, gy, ok := w.level.ComputeCeilingSlope(secIdx, w.scaleSectorH, w.scaleXY, w.flipY); ok {
				cSector.SlopedCeilingGradient = grad
				cSector.SlopedCeilingGX = gx
				cSector.SlopedCeilingGY = gy
			}
		}

		// Process walls belonging to this sector
		wallStart := int(sec.WallPtr)
		wallCount := int(sec.WallNum)
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
						adjCeilY := -float64(adjSec.CeilingZ) / 256.0 * w.scaleSectorH
						adjFloorY := -float64(adjSec.FloorZ) / 256.0 * w.scaleSectorH

						// Upper step wall
						if ceilY > adjCeilY || sec.IsSlopedCeiling() || adjSec.IsSlopedCeiling() {
							tile := w.textures.GetTile(int(wall.PicNum))
							cSeg.Upper = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
						}
						// Lower step wall
						if floorY < adjFloorY || sec.IsSlopedFloor() || adjSec.IsSlopedFloor() {
							tile := w.textures.GetTile(int(wall.PicNum))
							cSeg.Lower = config.NewConfigMaterial([]string{tile}, config.MaterialKindLoop, 1.0, 1.0, 0, 0)
						}
						// Masked / grating wall
						if wall.IsMasked() {
							tilePic := int(wall.OverPicNum)
							if tilePic < 0 {
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

	return configSectors, globalVertices, lights, nil
}
