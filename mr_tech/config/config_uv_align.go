package config

import (
	"math"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

// UVAlign represents UV alignment properties for texture mapping, including transformations, scaling, and wall alignment.
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
}

// NewUVAlignSurface creates a new UVAlign instance with scale factors for floors and ceilings.
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

// NewUVAlignVertical creates and returns a new UVAlign instance with specified alignment, flipping, panning, length, and scaling.
// alignBottom determines whether alignment is anchored at the bottom (true) or not.
// flipX specifies whether the UV mapping is flipped horizontally.
// flipY specifies whether the UV mapping is flipped vertically.
// panning defines the XY offset to adjust the UV mapping position.
// length sets the length for the UV mapping, with 0 implying no defined length.
// scaleY specifies the vertical scaling factor applied to the UV mapping.
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

// ComputeWall calculates the UV coordinates for wall textures based on alignment, scaling, panning, and flipping settings.
func (an *UVAlign) ComputeWall(w, h float64, t1, t2 float64, s geometry.XY, e geometry.XY, zBS, zBE, zTS, zTE float64) (float64, float64, float64, float64, float64, float64) {
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
	if an.AlignBottom {
		hWallS := (zTS - zBS) * scaleY
		hWallE := (zTE - zBE) * scaleY
		if an.FlipY {
			vBS = (hWallS + an.Panning.Y) / h
			vBE = (hWallE + an.Panning.Y) / h
			vTS = an.Panning.Y / h
			vTE = an.Panning.Y / h
		} else {
			vBS = an.Panning.Y / h
			vBE = an.Panning.Y / h
			vTS = (hWallS + an.Panning.Y) / h
			vTE = (hWallE + an.Panning.Y) / h
		}
	} else {
		hWallS := (zTS - zBS) * scaleY
		hWallE := (zTE - zBE) * scaleY
		if an.FlipY {
			vTS = -an.Panning.Y / h
			vTE = -an.Panning.Y / h
			vBS = vTS + hWallS/h
			vBE = vTE + hWallE/h
		} else {
			vTS = 1.0 + an.Panning.Y/h
			vTE = 1.0 + an.Panning.Y/h
			vBS = vTS - hWallS/h
			vBE = vTE - hWallE/h
		}
	}

	return uStart, uEnd, vBS, vBE, vTS, vTE
}

// Compute calculates UV texture coordinates for a triangular surface based on alignment, scaling, and transformation settings.
func (an *UVAlign) Compute(w, h float64, tri [3]geometry.XYZ) (float64, float64, float64, float64, float64, float64) {
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
