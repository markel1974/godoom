package aabb

import "math"

const maxClipPolygonVertices = 8

// ClipPolygon stores up to eight 3D vertices for polygon clipping during contact manifold construction.
// ClipPolygon is reusable scratch space for plane clipping without allocations.
type ClipPolygon struct {
	x     [maxClipPolygonVertices]float64
	y     [maxClipPolygonVertices]float64
	z     [maxClipPolygonVertices]float64
	count int
}

// NewClipPolygon creates an empty reusable polygon clipping buffer.
func NewClipPolygon() *ClipPolygon {
	return &ClipPolygon{}
}

// Get returns the x, y, and z coordinates of the vertex at index x.
func (p *ClipPolygon) Get(x int) (float64, float64, float64) {
	return p.x[x], p.y[x], p.z[x]
}

// Add appends a vertex to the polygon if capacity remains.
func (p *ClipPolygon) Add(vx, vy, vz float64) {
	if p.count < maxClipPolygonVertices {
		p.x[p.count] = vx
		p.y[p.count] = vy
		p.z[p.count] = vz
		p.count++
	}
}

// ClipPlane clips the polygon against the half-space nx*x+ny*y+nz*z <= limit.
// It writes the clipped vertices to out and returns the resulting vertex count.
func (p *ClipPolygon) ClipPlane(dst *ClipPolygon, nx, ny, nz, limit float64) int {
	dst.count = 0
	for i := 0; i < p.count; i++ {
		next := i + 1
		if next == p.count {
			next = 0
		}
		sx, sy, sz := p.x[i], p.y[i], p.z[i]
		ex, ey, ez := p.x[next], p.y[next], p.z[next]
		dS := sx*nx + sy*ny + sz*nz - limit
		dE := ex*nx + ey*ny + ez*nz - limit
		if dS <= 0.0 && dE <= 0.0 {
			dst.Add(ex, ey, ez)
		} else if dS <= 0.0 && dE > 0.0 {
			t := dS / (dS - dE)
			dst.Add(sx+t*(ex-sx), sy+t*(ey-sy), sz+t*(ez-sz))
		} else if dS > 0.0 && dE <= 0.0 {
			t := dS / (dS - dE)
			dst.Add(sx+t*(ex-sx), sy+t*(ey-sy), sz+t*(ez-sz))
			dst.Add(ex, ey, ez)
		}
	}
	return dst.count
}

// FillIncidentFaceQuad populates clip with the 4 world-space vertices of the OBB face most anti-parallel to (refNX, refNY, refNZ).
func (p *ClipPolygon) FillIncidentFaceQuad(incX, incY, incZ float64, incAxes *Axes, incHX, incHY, incHZ, refNX, refNY, refNZ float64) {
	incFaceIdx := 0
	maxAbsDot := -1.0
	incDotSign := 1.0
	for k := 0; k < 3; k++ {
		kx, ky, kz := incAxes.GetRow(k)
		d := refNX*kx + refNY*ky + refNZ*kz
		if ad := math.Abs(d); ad > maxAbsDot {
			maxAbsDot = ad
			incFaceIdx = k
			if d > 0.0 {
				incDotSign = -1.0
			} else {
				incDotSign = 1.0
			}
		}
	}

	inx, iny, inz := incAxes.GetRow(incFaceIdx)
	incHalfN := incDotSign * axisHalfExtent(incHX, incHY, incHZ, incFaceIdx)
	icx := incX + inx*incHalfN
	icy := incY + iny*incHalfN
	icz := incZ + inz*incHalfN

	iu := (incFaceIdx + 1) % 3
	iv := (incFaceIdx + 2) % 3
	ux, uy, uz := incAxes.GetRow(iu)
	vx, vy, vz := incAxes.GetRow(iv)
	hu := axisHalfExtent(incHX, incHY, incHZ, iu)
	hv := axisHalfExtent(incHX, incHY, incHZ, iv)
	ux *= hu
	uy *= hu
	uz *= hu
	vx *= hv
	vy *= hv
	vz *= hv

	p.count = 0
	p.Add(icx+ux+vx, icy+uy+vy, icz+uz+vz)
	p.Add(icx-ux+vx, icy-uy+vy, icz-uz+vz)
	p.Add(icx-ux-vx, icy-uy-vy, icz-uz-vz)
	p.Add(icx+ux-vx, icy+uy-vy, icz+uz-vz)
}

// ClipEdgeHalfSpace clips src against the outward normal plane of edge (ax,ay,az)->(bx,by,bz) in the plane with aligned normal effN.
func (p *ClipPolygon) ClipEdgeHalfSpace(dst *ClipPolygon, ax, ay, az, bx, by, bz, effNX, effNY, effNZ float64) bool {
	ex, ey, ez := bx-ax, by-ay, bz-az
	ox := ey*effNZ - ez*effNY
	oy := ez*effNX - ex*effNZ
	oz := ex*effNY - ey*effNX
	oLen := math.Sqrt(ox*ox + oy*oy + oz*oz)
	if oLen <= 1e-12 {
		return false
	}
	invLen := 1.0 / oLen
	ox *= invLen
	oy *= invLen
	oz *= invLen
	limit := ax*ox + ay*oy + az*oz + 1e-4
	return p.ClipPlane(dst, ox, oy, oz, limit) > 0
}
