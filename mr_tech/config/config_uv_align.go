package config

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// UVAlign defines UV mapping alignment properties for 2D or 3D surfaces with options for scaling, flipping, and panning.
type UVAlign struct {
	AlignWall   bool        `json:"alignWall"`
	Start       geometry.XY `json:"start"`
	End         geometry.XY `json:"end"`
	SwapXY      bool        `json:"swapXY"`
	FlipX       bool        `json:"flipX"`
	FlipY       bool        `json:"flipY"`
	Panning     geometry.XY `json:"panning"`
	Length      float64     `json:"length,omitempty"`
	AlignBottom bool        `json:"alignBottom,omitempty"`
	ScaleX      float64     `json:"scaleX,omitempty"`
	ScaleY      float64     `json:"scaleY,omitempty"`
	HasRefZ     bool        `json:"hasRefZ,omitempty"`
	RefZ        float64     `json:"refZ,omitempty"`
}

// SetRefZ configures an explicit reference Z altitude for vertical texture mapping.
func (an *UVAlign) SetRefZ(refZ float64) {
	an.HasRefZ = true
	an.RefZ = refZ
}

// NewUVAlignSurface creates and initializes a new UVAlign instance for aligning UV mappings on surfaces.
// alignWall specifies if alignment should follow wall constraints.
// start and end define the start and end points for alignment.
// swapXY swaps the X and Y components of UV mapping when true.
// flipX flips the UV mapping across the X-axis when true.
// flipY flips the UV mapping across the Y-axis when true.
// panning adjusts UV mapping by the given XY offset.
// scaleX and scaleY define scaling factors for the UV coordinates; values less than or equal to 0 default to 1.0.
func NewUVAlignSurface(alignWall bool, start geometry.XY, end geometry.XY, swapXY bool, flipX bool, flipY bool, panning geometry.XY, scaleX, scaleY float64) *UVAlign {
	if scaleX <= 0 {
		scaleX = 1.0
	}
	if scaleY <= 0 {
		scaleY = 1.0
	}
	return &UVAlign{
		AlignWall:   alignWall,
		AlignBottom: false,
		Start:       start,
		End:         end,
		SwapXY:      swapXY,
		FlipX:       flipX,
		FlipY:       flipY,
		Panning:     panning,
		Length:      0,
		ScaleX:      scaleX,
		ScaleY:      scaleY,
	}
}

// NewUVAlignVertical creates a new vertical UV alignment configuration for mapping textures onto 3D models.
// alignBottom determines if UVs should align to the bottom of the geometry.
// flipX and flipY control horizontal and vertical flipping of the UV coordinates, respectively.
// panning specifies the UV offset in 2D space.
// length adjusts the scaling factor along the vertical axis based on the geometry's height.
// scaleY specifies the scaling factor applied to the vertical direction of the UV mapping.
// Returns a pointer to a UVAlign instance.
func NewUVAlignVertical(alignBottom bool, flipX bool, flipY bool, panning geometry.XY, length float64, scaleY float64) *UVAlign {
	return &UVAlign{
		AlignWall:   false,
		AlignBottom: alignBottom,
		Start:       geometry.XY{},
		End:         geometry.XY{},
		SwapXY:      false,
		FlipX:       flipX,
		FlipY:       flipY,
		Panning:     panning,
		Length:      length,
		ScaleX:      1.0,
		ScaleY:      scaleY,
	}
}

// ComputeVertical calculates UV coordinates for vertical texture alignment based on geometry, scaling, and panning settings.
func (an *UVAlign) ComputeVertical(w, h float64, t1, t2 float64, s geometry.XY, e geometry.XY, zBS, zBE, zTS, zTE float64) (float64, float64, float64, float64, float64, float64) {
	wallLen := an.Length
	if wallLen <= 0 {
		wallLen = math.Hypot(e.X-s.X, e.Y-s.Y)
	}

	effT1 := t1
	effT2 := t2
	if an.FlipX {
		effT1 = 1.0 - effT1
		effT2 = 1.0 - effT2
	}

	d1 := effT1 * wallLen
	d2 := effT2 * wallLen

	uStart := (d1 + an.Panning.X) / w
	uEnd := (d2 + an.Panning.X) / w

	scaleY := an.ScaleY
	if scaleY <= 0 {
		scaleY = 1.0
	}

	var vBS, vBE, vTS, vTE float64
	dt := t2 - t1
	if an.AlignBottom {
		zBotStart := zBS
		if an.HasRefZ {
			zBotStart = an.RefZ
		} else if math.Abs(dt) > 1e-6 && math.Abs(t1) > 1e-6 {
			zBotStart = zBS - (t1/dt)*(zBE-zBS)
		}
		if an.FlipY {
			zTopStart := zTS
			if an.HasRefZ {
				zTopStart = an.RefZ
			} else if math.Abs(dt) > 1e-6 && math.Abs(t1) > 1e-6 {
				zTopStart = zTS - (t1/dt)*(zTE-zTS)
			}
			vTS = an.Panning.Y/h + ((zTopStart-zTS)*scaleY)/h
			vTE = an.Panning.Y/h + ((zTopStart-zTE)*scaleY)/h
			vBS = an.Panning.Y/h + ((zTopStart-zBS)*scaleY)/h
			vBE = an.Panning.Y/h + ((zTopStart-zBE)*scaleY)/h
		} else {
			vBS = an.Panning.Y/h + ((zBS-zBotStart)*scaleY)/h
			vBE = an.Panning.Y/h + ((zBE-zBotStart)*scaleY)/h
			vTS = an.Panning.Y/h + ((zTS-zBotStart)*scaleY)/h
			vTE = an.Panning.Y/h + ((zTE-zBotStart)*scaleY)/h
		}
	} else {
		zTopStart := zTS
		if an.HasRefZ {
			zTopStart = an.RefZ
		} else if math.Abs(dt) > 1e-6 && math.Abs(t1) > 1e-6 {
			zTopStart = zTS - (t1/dt)*(zTE-zTS)
		}
		if an.FlipY {
			vTS = -an.Panning.Y/h + ((zTopStart-zTS)*scaleY)/h
			vTE = -an.Panning.Y/h + ((zTopStart-zTE)*scaleY)/h
			vBS = -an.Panning.Y/h + ((zTopStart-zBS)*scaleY)/h
			vBE = -an.Panning.Y/h + ((zTopStart-zBE)*scaleY)/h
		} else {
			vTS = 1.0 + an.Panning.Y/h + ((zTS-zTopStart)*scaleY)/h
			vTE = 1.0 + an.Panning.Y/h + ((zTE-zTopStart)*scaleY)/h
			vBS = 1.0 + an.Panning.Y/h + ((zBS-zTopStart)*scaleY)/h
			vBE = 1.0 + an.Panning.Y/h + ((zBE-zTopStart)*scaleY)/h
		}
	}

	return uStart, uEnd, vBS, vBE, vTS, vTE
}

// ComputePlanar calculates UV coordinates for a triangle based on planar mapping and alignment settings of the UVAlign struct.
func (an *UVAlign) ComputePlanar(w, h float64, tri [3]geometry.XYZ) (float64, float64, float64, float64, float64, float64) {
	var tx, ty, nx, ny float64
	if an.AlignWall {
		dx := an.End.X - an.Start.X
		dy := an.End.Y - an.Start.Y
		length := math.Hypot(dx, dy)
		if length < 1e-6 {
			dx, dy, length = 1, 0, 1
		}
		tx = dx / length
		ty = dy / length
		nx = ty
		ny = -tx
	}

	scaleX := an.ScaleX
	if scaleX <= 0 {
		scaleX = 1.0
	}
	scaleY := an.ScaleY
	if scaleY <= 0 {
		scaleY = 1.0
	}

	var uvs [3][2]float64
	for i := 0; i < 3; i++ {
		var u, v float64
		if an.AlignWall {
			relX := tri[i].X - an.Start.X
			relY := tri[i].Y - an.Start.Y
			u = relX*tx + relY*ty
			v = relX*nx + relY*ny
		} else {
			u = tri[i].X
			v = tri[i].Y
		}
		u *= scaleX
		v *= scaleY
		if an.SwapXY {
			u, v = v, u
		}
		if an.FlipX {
			u = -u
		}
		if an.FlipY {
			v = -v
		}
		if an.Panning.X != 0 {
			u += an.Panning.X
		}
		if an.Panning.Y != 0 {
			v += an.Panning.Y
		}
		uvs[i][0] = u / w
		uvs[i][1] = v / h
	}
	return uvs[0][0], uvs[0][1], uvs[1][0], uvs[1][1], uvs[2][0], uvs[2][1]
}
