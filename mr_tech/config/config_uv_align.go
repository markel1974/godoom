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

// NewUVAlignWall creates a new UVAlign instance configured for wall alignment with specified transformation options.
func NewUVAlignWall(alignWall bool, start geometry.XY, end geometry.XY, swapXY bool, flipX bool, flipY bool, panning geometry.XY) *UVAlign {
	return NewUVAlignWallScaled(alignWall, start, end, swapXY, flipX, flipY, panning, 1.0, 1.0)
}

// NewUVAlignWallScaled creates a new UVAlign instance with scale factors for floors and ceilings.
func NewUVAlignWallScaled(alignWall bool, start geometry.XY, end geometry.XY, swapXY bool, flipX bool, flipY bool, panning geometry.XY, scaleX, scaleY float64) *UVAlign {
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

// NewUVAlignBottom creates and returns a new UVAlign instance with specified alignment, flipping, panning, length, and scaling.
// alignBottom determines whether alignment is anchored at the bottom (true) or not.
// flipX specifies whether the UV mapping is flipped horizontally.
// flipY specifies whether the UV mapping is flipped vertically.
// panning defines the XY offset to adjust the UV mapping position.
// length sets the length for the UV mapping, with 0 implying no defined length.
// scaleY specifies the vertical scaling factor applied to the UV mapping.
func NewUVAlignBottom(alignBottom bool, flipX bool, flipY bool, panning geometry.XY, length float64, scaleY float64) *UVAlign {
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

// ComputeWall calculates UV coordinates and their alignment based on wall geometry, scaling, and panning properties.
func (align *UVAlign) ComputeWall(w, h float64, t1, t2 float64, s geometry.XY, e geometry.XY, zBS, zBE, zTS, zTE float64) (float64, float64, float64, float64, float64, float64) {
	wallLen := align.Length
	if wallLen <= 0 {
		wallLen = math.Hypot(e.X-s.X, e.Y-s.Y)
	}

	d1 := t1 * wallLen
	d2 := t2 * wallLen

	uStart := (d1 + align.Panning.X) / w
	uEnd := (d2 + align.Panning.X) / w
	if align.FlipX {
		uStart, uEnd = uEnd, uStart
	}

	scaleY := align.ScaleY
	if scaleY <= 0 {
		scaleY = 1.0
	}

	var vBS, vBE, vTS, vTE float64
	if align.AlignBottom {
		hWallS := (zTS - zBS) * scaleY
		hWallE := (zTE - zBE) * scaleY
		vBS = align.Panning.Y / h
		vBE = align.Panning.Y / h
		vTS = (hWallS + align.Panning.Y) / h
		vTE = (hWallE + align.Panning.Y) / h
	} else {
		hWallS := (zTS - zBS) * scaleY
		hWallE := (zTE - zBE) * scaleY
		vTS = 1.0 + align.Panning.Y/h
		vTE = 1.0 + align.Panning.Y/h
		vBS = vTS - hWallS/h
		vBE = vTE - hWallE/h
	}
	if align.FlipY {
		vBS, vTS = vTS, vBS
		vBE, vTE = vTE, vBE
	}

	return uStart, uEnd, vBS, vBE, vTS, vTE
}

func (align *UVAlign) Compute(w, h float64, tri [3]geometry.XYZ) (float64, float64, float64, float64, float64, float64) {
	var tx, ty, nx, ny float64
	if align.AlignWall {
		dx := align.End.X - align.Start.X
		dy := align.End.Y - align.Start.Y
		length := math.Hypot(dx, dy)
		if length < 1e-6 {
			dx, dy, length = 1, 0, 1
		}
		tx = dx / length
		ty = dy / length
		nx = -ty
		ny = tx
	}

	scaleX := align.ScaleX
	if scaleX <= 0 {
		scaleX = 1.0
	}
	scaleY := align.ScaleY
	if scaleY <= 0 {
		scaleY = 1.0
	}

	var uvs [3][2]float64
	for i := 0; i < 3; i++ {
		var u, v float64
		if align.AlignWall {
			relX := tri[i].X - align.Start.X
			relY := tri[i].Y - align.Start.Y
			u = relX*tx + relY*ty
			v = relX*nx + relY*ny
		} else {
			u = tri[i].X
			v = tri[i].Y
		}
		u *= scaleX
		v *= scaleY
		if align.SwapXY {
			u, v = v, u
		}
		if align.FlipX {
			u = -u
		}
		if align.FlipY {
			v = -v
		}
		if align.Panning.X != 0 {
			u += align.Panning.X
		}
		if align.Panning.Y != 0 {
			v += align.Panning.Y
		}
		uvs[i][0] = u / w
		uvs[i][1] = v / h
	}
	return uvs[0][0], uvs[0][1], uvs[1][0], uvs[1][1], uvs[2][0], uvs[2][1]
}
