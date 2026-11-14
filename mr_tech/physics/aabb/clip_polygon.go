package aabb

// ClipPolygon stores up to eight 3D vertices for polygon clipping during contact manifold construction.
// ClipPolygon is reusable scratch space for plane clipping without allocations.
type ClipPolygon struct {
	x     [8]float64
	y     [8]float64
	z     [8]float64
	count int
}

// Add appends a vertex to the polygon if capacity remains.
func (p *ClipPolygon) Add(vx, vy, vz float64) {
	if p.count < len(p.x) {
		p.x[p.count] = vx
		p.y[p.count] = vy
		p.z[p.count] = vz
		p.count++
	}
}

// ClipPlane clips the polygon against the half-space nx*x+ny*y+nz*z <= limit.
// It writes the clipped vertices to out and returns the resulting vertex count.
func (p *ClipPolygon) ClipPlane(nx, ny, nz, limit float64, out *ClipPolygon) int {
	out.count = 0
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
			out.Add(ex, ey, ez)
		} else if dS <= 0.0 && dE > 0.0 {
			t := dS / (dS - dE)
			out.Add(sx+t*(ex-sx), sy+t*(ey-sy), sz+t*(ez-sz))
		} else if dS > 0.0 && dE <= 0.0 {
			t := dS / (dS - dE)
			out.Add(sx+t*(ex-sx), sy+t*(ey-sy), sz+t*(ez-sz))
			out.Add(ex, ey, ez)
		}
	}
	return out.count
}
