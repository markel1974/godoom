package script

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const (
	AspectRatio  = 1.333
	ScaleSectorH = 1.0
	PlayerHeight = 6.0 * ScaleSectorH
	PlayerRadius = 0.8
	PlayerSpeed  = 60.0
	PlayerMass   = 10.0

	GForce    = 9.8 * 20
	JumpForce = GForce * 6
)

// scaleW defines the scaling factor for width calculations.
// scaleH defines the scaling factor for height calculations.
const (
	scaleW      = 1.0
	scaleH      = 1.0
	texUnitSize = 8.0 // 1 full 1024x1024 texture every 8 map units
	scaleTex    = 1024.0 / texUnitSize
)

// Builder is a type used to construct and configure complex objects or data structures.
type Builder struct {
}

// NewBuilder initializes and returns a new instance of Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build processes the given data string to construct a Root configuration object, parsing vertices, sectors, and player info.
func (p *Builder) Build(res common.IResources, id string) (*config.Root, error) {
	basePath := "resources" + string(os.PathSeparator) + "textures" + string(os.PathSeparator)
	t, tErr := NewTextures(res, basePath)
	if tErr != nil {
		return nil, tErr
	}
	zp := NewPlayer(0, 0, 0)
	player := zp.Build()
	_ = common.NewPlayer(player)

	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	cal.AspectRatio = AspectRatio

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	cfg := config.NewConfigRoot(cal, nil, player, nil, scaleFactor, t)

	crosshair := common.NewSimpleCrosshair()
	t.AddTexture(crosshair.GetTexture())
	cfg.AddUIElement(crosshair.Create())

	oldData := strings.Split(id, "\n")
	configSectorIdx := 0

	for _, data := range oldData {
		var verb string
		r := strings.NewReader(data)
		if _, err := fmt.Fscanf(r, "%s", &verb); err != nil {
			continue
		}
		switch verb {
		case "#":
			//comment
		case "vertex":
			vertex, err := p.parseVertex(r)
			if err != nil {
				return nil, err
			}
			cfg.Vertices = append(cfg.Vertices, vertex...)
		case "sector":
			cs, err := p.parseSector(r, cfg.Vertices, configSectorIdx)
			if err != nil {
				return nil, err
			}
			configSectorIdx++
			cfg.Sectors = append(cfg.Sectors, cs)
		case "player":
			if err := p.parsePlayer(r, cfg); err != nil {
				return nil, err
			}
		default:
			continue
		}
	}

	if cfg.Vertices == nil {
		return nil, errors.New("nil vertices")
	}

	p.finalize(cfg)

	return cfg, nil
}

// segmentsOverlapOpposite checks if two line segments are collinear, point in opposite directions, and overlap.
func segmentsOverlapOpposite(a1, a2, b1, b2 geometry.XY) bool {
	dxA := a2.X - a1.X
	dyA := a2.Y - a1.Y
	dxB := b2.X - b1.X
	dyB := b2.Y - b1.Y

	// Must run in opposite directions
	dot := dxA*dxB + dyA*dyB
	if dot >= -1e-6 {
		return false
	}

	// Must be parallel (cross product close to 0)
	cross := dxA*dyB - dyA*dxB
	if math.Abs(cross) > 1e-4 {
		return false
	}

	lenSqA := dxA*dxA + dyA*dyA
	if lenSqA < 1e-8 {
		return false
	}

	// Point b1 must lie on line a1-a2
	crossP := (b1.X-a1.X)*dyA - (b1.Y-a1.Y)*dxA
	if math.Abs(crossP) > 1e-4 {
		return false
	}

	// Project B endpoints onto line A parameter interval [0, 1]
	tB1 := ((b1.X-a1.X)*dxA + (b1.Y-a1.Y)*dyA) / lenSqA
	tB2 := ((b2.X-a1.X)*dxA + (b2.Y-a1.Y)*dyA) / lenSqA

	minT := math.Min(tB1, tB2)
	maxT := math.Max(tB1, tB2)

	overlap := math.Min(1.0, maxT) - math.Max(0.0, minT)
	return overlap > 1e-4
}

// finalize processes and updates sector segment relationships within the configuration by performing a topological rescan.
func (p *Builder) finalize(cfg *config.Root) {
	for _, sector := range cfg.Sectors {
		for _, s := range sector.Segments {
			foundNeighbor := false
			var neighborSector *config.Sector
			for _, otherSector := range cfg.Sectors {
				if otherSector.Id == sector.Id {
					continue
				}
				for _, otherSeg := range otherSector.Segments {
					if segmentsOverlapOpposite(s.Start, s.End, otherSeg.Start, otherSeg.End) {
						foundNeighbor = true
						neighborSector = otherSector
						break
					}
				}
				if foundNeighbor {
					break
				}
			}
			if foundNeighbor && neighborSector != nil {
				s.Tag = neighborSector.Id
				s.Kind = config.SegmentUnknown
				s.Middle = nil
				s.UVAlign = nil
			} else {
				s.Kind = config.SegmentWall
				s.Tag = "unknown"
				s.Upper = nil
				s.Lower = nil
				s.UVAlignUpper = nil
				s.UVAlignLower = nil
			}
		}
	}
}

// parseVertex reads vertex data from an io.Reader, parsing it into a slice of geometry.XY points. Returns an error on failure.
func (p *Builder) parseVertex(r io.Reader) ([]geometry.XY, error) {
	var cfgVertices []geometry.XY
	for {
		var vertexY float64
		if _, err := fmt.Fscanf(r, "%f", &vertexY); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		for {
			xy := geometry.XY{Y: vertexY}
			if _, err := fmt.Fscanf(r, "%f", &xy.X); err != nil {
				if err != io.EOF {
					return nil, err
				}
				break
			}
			cfgVertices = append(cfgVertices, xy)
		}
	}
	return cfgVertices, nil
}

// parseSector parses sector data from the provided io.Reader and constructs a Sector object using given vertices and index.
// Returns the constructed Sector or an error if parsing fails.
func (p *Builder) parseSector(r io.Reader, cfgVertices []geometry.XY, configSectorIdx int) (*config.Sector, error) {
	if cfgVertices == nil {
		return nil, errors.New("nil vertices")
	}
	const falloff = 50.0
	const lightIntensity = 1.5
	cs := config.NewConfigSector(strconv.Itoa(configSectorIdx), lightIntensity, config.LightKindAmbient, falloff)
	if _, err := fmt.Fscanf(r, "%f%f", &cs.FloorY, &cs.CeilY); err != nil {
		return nil, err
	}
	type data struct {
		Val  int
		Kind int
	}
	var numbers []data
	var word string
	for {
		if _, err := fmt.Fscanf(r, "%32s", &word); err != nil {
			break
		}
		var d data
		d.Val = config.SegmentUnknown
		d.Kind = config.SegmentUnknown
		if word[0] != 'x' {
			if val, err := strconv.Atoi(word); err == nil {
				d.Val = val
			}
		}
		numbers = append(numbers, d)
	}

	if len(numbers) == 0 || len(numbers)%2 > 0 {
		return nil, errors.New("empty or invalid sector definition")
	}

	m := len(numbers) / 2
	for idx := 0; idx < m; idx++ {
		v1Idx := numbers[idx]
		v2Idx := numbers[(idx+1)%m] // Chiusura topologica
		neighborId := numbers[idx+m]

		if v1Idx.Val < 0 || v1Idx.Val >= len(cfgVertices) || v2Idx.Val < 0 || v2Idx.Val >= len(cfgVertices) {
			return nil, fmt.Errorf("invalid vertex index, max: %d", len(cfgVertices))
		}

		start := cfgVertices[v1Idx.Val]
		end := cfgVertices[v2Idx.Val]

		seg := config.NewConfigSegment("", neighborId.Kind, start, end)
		seg.Middle = config.NewConfigMaterial([]string{"wall2.ppm"}, config.MaterialKindLoop, 0, 0)
		seg.Lower = config.NewConfigMaterial([]string{"wall.ppm"}, config.MaterialKindLoop, 0, 0)
		seg.Upper = config.NewConfigMaterial([]string{"wall3.ppm"}, config.MaterialKindLoop, 0, 0)

		wallLen := math.Hypot(end.X-start.X, end.Y-start.Y) * scaleTex
		panning := geometry.XY{}
		seg.UVAlign = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
		seg.UVAlignUpper = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)
		seg.UVAlignLower = config.NewUVAlignVertical(false, false, false, panning, wallLen, scaleTex)

		cs.Segments = append(cs.Segments, seg)
	}
	cs.Floor = config.NewConfigMaterial([]string{"floor.ppm"}, config.MaterialKindLoop, 0, 0)
	cs.Ceil = config.NewConfigMaterial([]string{"ceil.ppm"}, config.MaterialKindLoop, 0, 0)

	cs.FloorAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)
	cs.CeilAlign = config.NewUVAlignSurface(false, geometry.XY{}, geometry.XY{}, false, false, false, geometry.XY{}, scaleTex, scaleTex)

	return cs, nil
}

// parsePlayer reads position and angle data from the given reader and populates the specified Player structure.
func (p *Builder) parsePlayer(r io.Reader, cfg *config.Root) error {
	var sectorIdx int = -1
	n, err := fmt.Fscanf(r, "%f %f %f %d", &cfg.Player.Position.X, &cfg.Player.Position.Y, &cfg.Player.Angle, &sectorIdx)
	if err != nil && err != io.EOF && n < 3 {
		return err
	}
	if sectorIdx >= 0 && sectorIdx < len(cfg.Sectors) {
		sec := cfg.Sectors[sectorIdx]
		cfg.Player.Position.Z = sec.FloorY + 0.0001
	} else {
		// Fallback: search for sector containing player (px, py) with lowest FloorY
		playerPt := geometry.XY{X: cfg.Player.Position.X, Y: cfg.Player.Position.Y}
		var bestFloor float64 = 0.0
		found := false
		for _, sec := range cfg.Sectors {
			poly := make(geometry.Polygon, len(sec.Segments))
			for i, seg := range sec.Segments {
				poly[i] = seg.Start
			}
			if poly.PointInPolygon(playerPt) {
				if !found || sec.FloorY < bestFloor {
					bestFloor = sec.FloorY
					found = true
				}
			}
		}
		if found {
			cfg.Player.Position.Z = bestFloor + 0.0001
		}
	}
	return nil
}
