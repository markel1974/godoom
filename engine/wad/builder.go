package wad

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
)

//http://www.gamers.org/dhs/helpdocs/dmsp1666.html
//http://doomwiki.org/
//https://github.com/penberg/godoom

type XY struct {
	X float64
	Y float64
}

type Point3 struct {
	X float64
	Y float64
	Z float64
	U float64
	V float64
}

func MakePoint3F(x, y, z, u, v float64) Point3{
	return Point3{ X:x, Y:y, Z:z, U:u, V: v }
}

func MakePoint3(x, y, z, u, v int16) Point3{
	return MakePoint3F(float64(x), float64(y), float64(z), float64(u), float64(v))
}




type Builder struct {
	w * WAD
}

func NewBuilder() * Builder {
	return &Builder{
	}
}

func (b * Builder) Setup(wadFile string, levelNumber int) error {
	b.w = New()
	if err := b.w.Load(wadFile); err != nil {
		return err
	}
	levelNames := b.w.GetLevels()
	if len(levelNames) == 0 {
		return errors.New("error: No levels found")
	}
	levelIdx := levelNumber - 1
	if levelIdx >= len(levelNames) {
		return errors.New(fmt.Sprintf("error: No such level number %d", levelNumber))
	}
	levelName := levelNames[levelIdx]
	fmt.Printf("Loading level %s ...\n", levelName)
	level, err := b.w.GetLevel(levelName)
	if err != nil {
		fmt.Printf("error: %s\n", err)
		os.Exit(1)
	}
	b.createSubSector(level)
	return nil
}

func (b * Builder) createSubSector(level *Level) {
	for _, subSector := range level.SubSectors {
		for seg := subSector.StartSeg; seg < subSector.StartSeg + subSector.NumSegments; seg++ {
			b.createSegment(level, int(seg))
		}
	}
}

func (b * Builder) createSegment(level *Level, segmentId int) {
	segment := level.Segments[segmentId]

	//meshes := scene.meshes[subSectorId]

	lineDef := level.LineDefs[int(segment.LineNum)]
	sideDef := b.segmentSideDef(level, &segment, &lineDef)
	if sideDef == nil { return }
	sector := level.Sectors[sideDef.SectorRef]

	oppositeSideDef := b.segmentOppositeSideDef(level, &segment, &lineDef)

	start := level.Vertexes[segment.VertexStart]
	end := level.Vertexes[segment.VertexEnd]

	upperTexture := sideDef.UpperTexture
	middleTexture := sideDef.MiddleTexture
	lowerTexture := sideDef.LowerTexture

	//fmt.Println(sector, start, end)

	if upperTexture != "-" && oppositeSideDef != nil {
		neighbor := level.Sectors[oppositeSideDef.SectorRef]
		//sector.Neighbor = append(sector.Neighbor, neighbor)
		//sector.Vertex = append(sector.Vertex, XY{X:float64(start.XCoord), Y: float64(start.YCoord)})
		//sector.Vertex = append(sector.Vertex, XY{X:float64(start.XCoord), Y: float64(start.YCoord)})

		var vertices []Point3
		vertices = append(vertices, MakePoint3(start.XCoord, start.YCoord, sector.CeilingHeight,0.0,1.0))
		vertices = append(vertices, MakePoint3(end.XCoord, end.YCoord, sector.CeilingHeight,0.0,1.0))
		vertices = append(vertices, MakePoint3(end.XCoord, end.YCoord, neighbor.CeilingHeight,0.0,1.0))

		//vertices = append(vertices, Point3{X: float64(-start.XCoord), Y: float64(sector.CeilingHeight), Z: float64(start.YCoord), U: 0.0, V: 1.0})
		//vertices = append(vertices, Point3{X: float64(-start.XCoord), Y: float64(sector.CeilingHeight), Z: float64(start.YCoord), U: 0.0, V: 0.0})
		//vertices = append(vertices, Point3{X: float64(-end.XCoord), Y: float64(sector.CeilingHeight), Z: float64(end.YCoord), U: 1.0, V: 0.0})

		//vertices = append(vertices, Point3{X: float64(-end.XCoord), Y: float64(sector.CeilingHeight), Z: float64(end.YCoord), U: 1.0, V: 0.0})
		//vertices = append(vertices, Point3{X: float64(-end.XCoord), Y: float64(sector.CeilingHeight), Z: float64(end.YCoord), U: 1.0, V: 1.0})
		//vertices = append(vertices, Point3{X: float64(-start.XCoord), Y: float64(sector.CeilingHeight), Z: float64(start.YCoord), U: 0.0, V: 1.0})

		//meshes = append(meshes, NewMesh(upperTexture, sector.LightLevel, vertices))
		//scene.CacheTexture(wad, upperTexture)

		fmt.Println(vertices)
	}

	if middleTexture != "-" {
		//fmt.Println("TEST")
		/*
		vertices := []Point3{}

		vertices = append(vertices, Point3{X: -start.XCoord, Y: sector.FloorHeight, Z: start.YCoord, U: 0.0, V: 1.0})
		vertices = append(vertices, Point3{X: -start.XCoord, Y: sector.CeilingHeight, Z: start.YCoord, U: 0.0, V: 0.0})
		vertices = append(vertices, Point3{X: -end.XCoord, Y: sector.CeilingHeight, Z: end.YCoord, U: 1.0, V: 0.0})

		vertices = append(vertices, Point3{X: -end.XCoord, Y: sector.CeilingHeight, Z: end.YCoord, U: 1.0, V: 0.0})
		vertices = append(vertices, Point3{X: -end.XCoord, Y: sector.FloorHeight, Z: end.YCoord, U: 1.0, V: 1.0})
		vertices = append(vertices, Point3{X: -start.XCoord, Y: sector.FloorHeight, Z: start.YCoord, U: 0.0, V: 1.0})

		meshes = append(meshes, NewMesh(middleTexture, sector.LightLevel, vertices))

		scene.CacheTexture(wad, middleTexture)
		*/
	}

	if lowerTexture != "-" && oppositeSideDef != nil {
		oppositeSector := level.Sectors[oppositeSideDef.SectorRef]

		fmt.Println(oppositeSector)

		/*
		vertices := []Point3{}

		vertices = append(vertices, Point3{X: -start.XCoord, Y: sector.FloorHeight, Z: start.YCoord, U: 0.0, V: 1.0})
		vertices = append(vertices, Point3{X: -start.XCoord, Y: oppositeSector.FloorHeight, Z: start.YCoord, U: 0.0, V: 0.0})
		vertices = append(vertices, Point3{X: -end.XCoord, Y: oppositeSector.FloorHeight, Z: end.YCoord, U: 1.0, V: 0.0})

		vertices = append(vertices, Point3{X: -end.XCoord, Y: oppositeSector.FloorHeight, Z: end.YCoord, U: 1.0, V: 0.0})
		vertices = append(vertices, Point3{X: -end.XCoord, Y: sector.FloorHeight, Z: end.YCoord, U: 1.0, V: 1.0})
		vertices = append(vertices, Point3{X: -start.XCoord, Y: sector.FloorHeight, Z: start.YCoord, U: 0.0, V: 1.0})

		meshes = append(meshes, NewMesh(lowerTexture, sector.LightLevel, vertices))

		scene.CacheTexture(wad, lowerTexture)
		*/
	}

	//scene.meshes[ssectorId] = meshes
}

func (b * Builder) segmentSideDef(level *Level, seg *Seg, lineDef *LineDef) *SideDef {
	if seg.SegmentSide == 0 { return &level.SideDefs[lineDef.SideDefRight] }
	if lineDef.SideDefLeft == -1 { return nil }
	return &level.SideDefs[lineDef.SideDefLeft]
}

func (b * Builder) segmentOppositeSideDef(level *Level, seg *Seg, lineDef *LineDef) *SideDef {
	if seg.SegmentSide == 0 {
		if lineDef.SideDefLeft == -1 { return nil }
		return &level.SideDefs[lineDef.SideDefLeft]
	}
	return &level.SideDefs[lineDef.SideDefRight]
}

func (b * Builder) loadTexture(wad *WAD, textureName string) (*image.RGBA, error) {
	texture, ok := wad.GetTexture(textureName)
	if !ok {
		return nil, errors.New("unknown texture " + textureName)
	}
	if texture.Header == nil {
		return nil, nil
	}
	bounds := image.Rect(0, 0, int(texture.Header.Width), int(texture.Header.Height))
	rgba := image.NewRGBA(bounds)
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return nil, fmt.Errorf("unsupported stride")
	}
	for _, patch := range texture.Patches {
		img, ok := wad.GetImage(patch.PNameNumber)
		if !ok {
			return nil, errors.New(fmt.Sprintf("unknown patch %d for %s", patch.PNameNumber, textureName))
		}
		for y := 0; y < img.Height; y++ {
			for x := 0; x < img.Width; x++ {
				pixel := img.Pixels[y*img.Width+x]
				var alpha uint8
				if pixel == wad.transparentPaletteIndex {
					alpha = 0
				} else {
					alpha = 255
				}
				rgb := wad.playPal.Palettes[0].Table[pixel]
				rgba.Set(int(patch.XOffset) + x, int(patch.YOffset) + y, color.RGBA{R: rgb.Red, G: rgb.Green, B: rgb.Blue, A: alpha})
			}
		}
	}
	return rgba, nil

	/*
	var texId uint32
	gl.GenTextures(1, &texId)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texId)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	return texId, nil
	*/
}