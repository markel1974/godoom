package vramtex

import "math"

// buildBaseEnvFaces generates a cubemap's base mip from input RGBA faces or creates a fallback procedural sky environment.
func buildBaseEnvFaces(envMaps map[string][6][]byte) (int, [6][]float32) {
	var out [6][]float32
	for _, faces := range envMaps {
		faceLen := len(faces[0])
		if faceLen < 4 {
			continue
		}
		dim := int(math.Sqrt(float64(faceLen / 4)))
		if dim >= 4 && dim*dim*4 == faceLen {
			valid := true
			for f := 1; f < 6; f++ {
				if len(faces[f]) != faceLen {
					valid = false
					break
				}
			}
			if valid {
				for f := range 6 {
					src := faces[f]
					lin := make([]float32, dim*dim*3)
					for i, j := 0, 0; i < len(src); i, j = i+4, j+3 {
						lin[j] = float32(math.Pow(float64(src[i])/255.0, 2.2))
						lin[j+1] = float32(math.Pow(float64(src[i+1])/255.0, 2.2))
						lin[j+2] = float32(math.Pow(float64(src[i+2])/255.0, 2.2))
					}
					out[f] = lin
				}
				return dim, out
			}
		}
	}

	const size = 64
	sunX, sunY, sunZ := 0.44, 0.72, 0.54
	sunL := math.Sqrt(sunX*sunX + sunY*sunY + sunZ*sunZ)
	sunX, sunY, sunZ = sunX/sunL, sunY/sunL, sunZ/sunL
	for f := range 6 {
		lin := make([]float32, size*size*3)
		for y := range size {
			v := ((float64(y)+0.5)/float64(size))*2.0 - 1.0
			for x := range size {
				u := ((float64(x)+0.5)/float64(size))*2.0 - 1.0
				nx, ny, nz := cubeUVToDir(f, u, v)
				var r, g, b float64
				if ny >= 0 {
					t := math.Pow(ny, 0.55)
					// Linear sky gradient (horizon mist -> steel zenith)
					r = (1.0-t)*0.30 + t*0.035
					g = (1.0-t)*0.39 + t*0.075
					b = (1.0-t)*0.52 + t*0.18
					sunDot := math.Max(0, nx*sunX+ny*sunY+nz*sunZ)
					sunGlow := math.Pow(sunDot, 16.0)*0.35 + math.Pow(sunDot, 128.0)*1.35
					r += sunGlow * 1.05
					g += sunGlow * 0.94
					b += sunGlow * 0.76
				} else {
					t := math.Pow(-ny, 0.5)
					// Linear ground bounce (horizon mist -> dark slate floor)
					r = (1.0-t)*0.18 + t*0.015
					g = (1.0-t)*0.23 + t*0.018
					b = (1.0-t)*0.29 + t*0.026
				}
				off := (y*size + x) * 3
				lin[off] = float32(r)
				lin[off+1] = float32(g)
				lin[off+2] = float32(b)
			}
		}
		out[f] = lin
	}
	return size, out
}

// encodeEnvFacesSRGB encodes 6 cube faces with sRGB gamma correction and returns the encoded byte arrays.
func encodeEnvFacesSRGB(faces [6][]float32, dim int) [6][]byte {
	var encoded [6][]byte
	const invGamma = 1.0 / 2.2
	clampB := func(v float64) byte {
		if v <= 0 {
			return 0
		}
		s := math.Pow(v, invGamma)
		if s >= 1 {
			return 255
		}
		return byte(s*255.0 + 0.5)
	}
	texels := dim * dim
	for f := 0; f < 6; f++ {
		src := faces[f]
		dst := make([]byte, texels*4)
		for i, j := 0, 0; i < texels; i, j = i+1, j+3 {
			o := i * 4
			dst[o] = clampB(float64(src[j]))
			dst[o+1] = clampB(float64(src[j+1]))
			dst[o+2] = clampB(float64(src[j+2]))
			dst[o+3] = 255
		}
		encoded[f] = dst
	}
	return encoded
}

// cubeUVToDir converts a cube map face index and UV coordinates into a normalized direction vector (x, y, z).
func cubeUVToDir(face int, u, v float64) (float64, float64, float64) {
	var dx, dy, dz float64
	switch face {
	case 0:
		dx, dy, dz = 1, -v, -u
	case 1:
		dx, dy, dz = -1, -v, u
	case 2:
		dx, dy, dz = u, 1, v
	case 3:
		dx, dy, dz = u, -1, -v
	case 4:
		dx, dy, dz = u, -v, 1
	default:
		dx, dy, dz = -u, -v, -1
	}
	invL := 1.0 / math.Sqrt(dx*dx+dy*dy+dz*dz)
	return dx * invL, dy * invL, dz * invL
}

// dirToCubeUV converts a 3D direction vector (nx, ny, nz) to its corresponding cubemap face index and normalized UV coordinates.
func dirToCubeUV(nx, ny, nz float64) (int, float64, float64) {
	ax, ay, az := math.Abs(nx), math.Abs(ny), math.Abs(nz)
	var face int
	var ma, uc, vc float64
	if ax >= ay && ax >= az {
		ma = ax
		if nx >= 0 {
			face = 0
			uc, vc = -nz, -ny
		} else {
			face = 1
			uc, vc = nz, -ny
		}
	} else if ay >= ax && ay >= az {
		ma = ay
		if ny >= 0 {
			face = 2
			uc, vc = nx, nz
		} else {
			face = 3
			uc, vc = nx, -nz
		}
	} else {
		ma = az
		if nz >= 0 {
			face = 4
			uc, vc = nx, -ny
		} else {
			face = 5
			uc, vc = -nx, -ny
		}
	}
	if ma <= 1e-12 {
		return face, 0, 0
	}
	return face, uc / ma, vc / ma
}

// sampleCubeLinear performs bilinear sampling of a cubemap using normalized direction coordinates (nx, ny, nz).
// Returns the interpolated red, green, and blue color components as float64 values.
// The cubemap is represented by 6 faces, each a 1D array of float32 colors, and `dim` specifies the face resolution.
func sampleCubeLinear(faces [6][]float32, dim int, nx, ny, nz float64) (float64, float64, float64) {
	f, u, v := dirToCubeUV(nx, ny, nz)
	fx := (u*0.5+0.5)*float64(dim) - 0.5
	fy := (v*0.5+0.5)*float64(dim) - 0.5
	if fx < 0 {
		fx = 0
	} else if fx > float64(dim-1) {
		fx = float64(dim - 1)
	}
	if fy < 0 {
		fy = 0
	} else if fy > float64(dim-1) {
		fy = float64(dim - 1)
	}
	x0 := int(fx)
	y0 := int(fy)
	x1 := x0 + 1
	if x1 >= dim {
		x1 = dim - 1
	}
	y1 := y0 + 1
	if y1 >= dim {
		y1 = dim - 1
	}
	tx := fx - float64(x0)
	ty := fy - float64(y0)
	src := faces[f]
	i00 := (y0*dim + x0) * 3
	i10 := (y0*dim + x1) * 3
	i01 := (y1*dim + x0) * 3
	i11 := (y1*dim + x1) * 3
	w00 := (1 - tx) * (1 - ty)
	w10 := tx * (1 - ty)
	w01 := (1 - tx) * ty
	w11 := tx * ty
	r := float64(src[i00])*w00 + float64(src[i10])*w10 + float64(src[i01])*w01 + float64(src[i11])*w11
	g := float64(src[i00+1])*w00 + float64(src[i10+1])*w10 + float64(src[i01+1])*w01 + float64(src[i11+1])*w11
	b := float64(src[i00+2])*w00 + float64(src[i10+2])*w10 + float64(src[i01+2])*w01 + float64(src[i11+2])*w11
	return r, g, b
}

// prefilterEnvMip performs GGX importance sampling to generate a prefiltered mip level for environment cubemaps.
// It accepts input cubemap faces, source dimensions, destination dimensions, surface roughness, and sample count.
// Returns an array of 6 cubemap faces representing the filtered environment for the specified roughness level.
func prefilterEnvMip(src [6][]float32, srcDim, dstDim int, roughness float64, sampleCount int) [6][]float32 {
	a := roughness * roughness
	a2 := a * a
	if a2 < 1e-4 {
		a2 = 1e-4
	}
	// Precompute local GGX half-vectors around +Z in tangent space.
	type tangentSample struct{ hx, hy, hz float64 }
	hVecs := make([]tangentSample, sampleCount)
	for i := range sampleCount {
		u1 := (float64(i) + 0.5) / float64(sampleCount)
		// Van der Corput radical inverse base 2
		bits := uint32(i + 1)
		bits = (bits << 16) | (bits >> 16)
		bits = ((bits & 0x55555555) << 1) | ((bits & 0xAAAAAAAA) >> 1)
		bits = ((bits & 0x33333333) << 2) | ((bits & 0xCCCCCCCC) >> 2)
		bits = ((bits & 0x0F0F0F0F) << 4) | ((bits & 0xF0F0F0F0) >> 4)
		bits = ((bits & 0x00FF00FF) << 8) | ((bits & 0xFF00FF00) >> 8)
		u2 := float64(bits) * 2.3283064365386963e-10
		phi := 2.0 * math.Pi * u1
		cosTheta := math.Sqrt((1.0 - u2) / (1.0 + (a2-1.0)*u2))
		sinTheta := math.Sqrt(math.Max(0.0, 1.0-cosTheta*cosTheta))
		hVecs[i] = tangentSample{
			hx: math.Cos(phi) * sinTheta,
			hy: math.Sin(phi) * sinTheta,
			hz: cosTheta,
		}
	}

	var out [6][]float32
	for f := range 6 {
		dst := make([]float32, dstDim*dstDim*3)
		for y := range dstDim {
			v := ((float64(y)+0.5)/float64(dstDim))*2.0 - 1.0
			for x := range dstDim {
				u := ((float64(x)+0.5)/float64(dstDim))*2.0 - 1.0
				nx, ny, nz := cubeUVToDir(f, u, v)
				upX, upY, upZ := 0.0, 1.0, 0.0
				if math.Abs(ny) > 0.999 {
					upX, upY, upZ = 0.0, 0.0, 1.0
				}
				// T = normalize(cross(up, N))
				tx := upY*nz - upZ*ny
				ty := upZ*nx - upX*nz
				tz := upX*ny - upY*nx
				tInv := 1.0 / math.Sqrt(tx*tx+ty*ty+tz*tz)
				tx, ty, tz = tx*tInv, ty*tInv, tz*tInv
				// B = cross(N, T)
				bx := ny*tz - nz*ty
				by := nz*tx - nx*tz
				bz := nx*ty - ny*tx

				var sumR, sumG, sumB, totalW float64
				for s := range sampleCount {
					hv := hVecs[s]
					hx := tx*hv.hx + bx*hv.hy + nx*hv.hz
					hy := ty*hv.hx + by*hv.hy + ny*hv.hz
					hz := tz*hv.hx + bz*hv.hy + nz*hv.hz
					nDotH := nx*hx + ny*hy + nz*hz
					lx := 2.0*nDotH*hx - nx
					ly := 2.0*nDotH*hy - ny
					lz := 2.0*nDotH*hz - nz
					nDotL := nx*lx + ny*ly + nz*lz
					if nDotL > 0 {
						cr, cg, cb := sampleCubeLinear(src, srcDim, lx, ly, lz)
						sumR += cr * nDotL
						sumG += cg * nDotL
						sumB += cb * nDotL
						totalW += nDotL
					}
				}
				off := (y*dstDim + x) * 3
				if totalW > 1e-6 {
					invW := 1.0 / totalW
					dst[off] = float32(sumR * invW)
					dst[off+1] = float32(sumG * invW)
					dst[off+2] = float32(sumB * invW)
				} else {
					cr, cg, cb := sampleCubeLinear(src, srcDim, nx, ny, nz)
					dst[off] = float32(cr)
					dst[off+1] = float32(cg)
					dst[off+2] = float32(cb)
				}
			}
		}
		out[f] = dst
	}
	return out
}
