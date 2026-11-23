package vramtex

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
)

// Textures manages a collection of 2D textures and their associated VRAM buckets for efficient rendering and sampling.
type Textures struct {
	ctx           api.IContext
	slots         []int
	buckets       []*Bucket
	textures      map[*textures.Texture]float32
	envMapTex     uint32
	envMaxLod     float32
	lightmapTex   uint32
	lightmapCount int32
}

// NewTextures creates and returns a new instance of Textures with an initialized map to store texture references.
func NewTextures(ctx api.IContext) *Textures {
	slots := []int{64, 256, 1024}
	tx := &Textures{
		ctx:       ctx,
		slots:     slots,
		buckets:   make([]*Bucket, len(slots)),
		textures:  make(map[*textures.Texture]float32),
		envMaxLod: 5.0,
	}
	for i := range tx.slots {
		tx.buckets[i] = NewBucket(tx.slots[i])
	}
	return tx
}

// Setup initializes texture arrays and their respective VRAM buckets for efficient rendering and management.
func (tx *Textures) Setup(t textures.ITextures) error {
	now := time.Now()
	fmt.Println("starting compiling textures")

	names := t.GetNames()
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)

	type Usable struct {
		maxDim int
		tex    *textures.Texture
	}

	var valid []*Usable
	for _, id := range names {
		tn := t.GetFrames([]string{id})
		if len(tn) == 0 {
			continue
		}
		tex := tn[0]
		if tex == nil {
			continue
		}
		w, h := tex.GetSize()
		if w <= 0 || h <= 0 {
			continue
		}
		maxDim := int(w)
		if int(h) > maxDim {
			maxDim = int(h)
		}
		usable := &Usable{maxDim: maxDim, tex: tex}
		valid = append(valid, usable)
		idx := tx.idxByDim(maxDim)
		tx.buckets[idx].Count++
	}

	// Allocazione VRAM selettiva
	for i := range tx.buckets {
		if tx.buckets[i].Count > 0 {
			tx.buckets[i].DiffuseArray = tx.createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].NormalArray = tx.createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].EmissiveArray = tx.createTextureArray(tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
		}
	}

	for _, usable := range valid {
		bIdx := tx.idxByDim(usable.maxDim)
		size := tx.buckets[bIdx].Size
		layer := tx.buckets[bIdx].Layer
		diffuse, normal, emissive := usable.tex.Compile(size)
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[bIdx].DiffuseArray)
		tx.ctx.TexSubImage3D(api.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(diffuse))
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[bIdx].NormalArray)
		tx.ctx.TexSubImage3D(api.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(normal))
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[bIdx].EmissiveArray)
		tx.ctx.TexSubImage3D(api.TEXTURE_2D_ARRAY, 0, 0, 0, layer, int32(size), int32(size), 1, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(emissive))
		//Impacchettiamo Bucket e Layer in un solo Float
		packedValue := float32(bIdx*1000) + float32(layer)
		tx.textures[usable.tex] = packedValue
		tx.buckets[bIdx].Layer++
	}

	// Generazione MipMaps globali (aggiunto l'emissivo!)
	for i, l := range tx.buckets {
		if l.Layer > 0 {
			tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[i].DiffuseArray)
			tx.ctx.GenerateMipmap(api.TEXTURE_2D_ARRAY)
			tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[i].NormalArray)
			tx.ctx.GenerateMipmap(api.TEXTURE_2D_ARRAY)
			tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.buckets[i].EmissiveArray)
			tx.ctx.GenerateMipmap(api.TEXTURE_2D_ARRAY)
		}
	}

	fmt.Println("textures compiled in", time.Since(now))
	return nil
}

// Get retrieves the layer value associated with the given texture and returns a boolean indicating if the texture exists.
func (tx *Textures) Get(tex *textures.Texture) (float32, bool) {
	t, ok := tx.textures[tex]
	return t, ok
}

// GetBucketsLen returns the number of buckets in the Textures structure.
func (tx *Textures) GetBucketsLen() int {
	return len(tx.buckets)
}

// GetBucket retrieves the DiffuseArray, NormalArray, and EmissiveArray from the specified bucket index b.
func (tx *Textures) GetBucket(b int) (uint32, uint32, uint32) {
	return tx.buckets[b].DiffuseArray, tx.buckets[b].NormalArray, tx.buckets[b].EmissiveArray
}

// BindBuckets binds texture buckets to specific OpenGL texture units for rendering diffuse, normal, and emissive maps.
func (tx *Textures) BindBuckets() {
	// Unit groups: Diffuse [0..2] | Normal [3..5] | Emissive [6..8]
	for i := uint32(0); i < uint32(len(tx.buckets)); i++ {
		diffuse := tx.buckets[i].DiffuseArray
		normal := tx.buckets[i].NormalArray
		emissive := tx.buckets[i].EmissiveArray
		tx.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitDiffuseBase + i)
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, diffuse)
		tx.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitNormalBase + i)
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, normal)
		tx.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitEmissiveBase + i)
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, emissive)
	}
	tx.BindEnvironmentAndLightmaps()
}

// BindEnvironmentAndLightmaps binds the cubemap environment texture and baked lightmap array
// onto their fixed texture units (TexUnitEnvMap and TexUnitLightmap).
func (tx *Textures) BindEnvironmentAndLightmaps() {
	if tx.envMapTex != 0 {
		tx.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitEnvMap)
		tx.ctx.BindTexture(api.TEXTURE_CUBE_MAP, tx.envMapTex)
	}
	if tx.lightmapTex != 0 {
		tx.ctx.ActiveTexture(api.TEXTURE0 + api.TexUnitLightmap)
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.lightmapTex)
	}
}

// GetEnvMapTexture returns the GPU cubemap texture ID.
func (tx *Textures) GetEnvMapTexture() uint32 {
	return tx.envMapTex
}

// GetEnvMapMaxLod returns the maximum mipmap LOD index available in the environment cubemap.
func (tx *Textures) GetEnvMapMaxLod() float32 {
	return tx.envMaxLod
}

// GetLightmapTexture returns the GPU 2D array lightmap texture ID.
func (tx *Textures) GetLightmapTexture() uint32 {
	return tx.lightmapTex
}

// HasLightmaps reports whether real baked lightmaps were uploaded.
func (tx *Textures) HasLightmaps() bool {
	return tx.lightmapCount > 0
}

// UploadEnvironment uploads a 6-face RGBA environment cubemap with a GGX prefiltered mip chain
// from config or builds a procedural sky/horizon/sun/ground cubemap fallback.
func (tx *Textures) UploadEnvironment(envMaps map[string][6][]byte) {
	if tx.envMapTex != 0 {
		return
	}
	tx.ctx.GenTextures(1, &tx.envMapTex)
	tx.ctx.BindTexture(api.TEXTURE_CUBE_MAP, tx.envMapTex)

	dim, baseLinear := buildBaseEnvFaces(envMaps)
	maxMip := int32(math.Floor(math.Log2(float64(dim))))
	if maxMip > 5 {
		maxMip = 5
	}
	if maxMip < 0 {
		maxMip = 0
	}
	tx.envMaxLod = float32(maxMip)

	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_BASE_LEVEL, 0)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_MAX_LEVEL, maxMip)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_MIN_FILTER, api.LINEAR_MIPMAP_LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_MAG_FILTER, api.LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_R, api.CLAMP_TO_EDGE)

	mip0Bytes := encodeEnvFacesSRGB(baseLinear, dim)
	for f := 0; f < 6; f++ {
		tx.ctx.TexImage2D(api.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(f), 0, api.RGBA8, int32(dim), int32(dim), 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(mip0Bytes[f]))
	}

	// Build GGX-importance prefiltered mip levels cleanly across cube boundaries.
	for level := int32(1); level <= maxMip; level++ {
		mipSize := dim >> level
		if mipSize < 1 {
			mipSize = 1
		}
		roughness := float64(level) / float64(maxMip)
		samples := 24
		if mipSize >= 128 {
			samples = 12
		}
		filtered := prefilterEnvMip(baseLinear, dim, mipSize, roughness, samples)
		mipBytes := encodeEnvFacesSRGB(filtered, mipSize)
		for f := 0; f < 6; f++ {
			tx.ctx.TexImage2D(api.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(f), level, api.RGBA8, int32(mipSize), int32(mipSize), 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(mipBytes[f]))
		}
	}

	tx.ctx.BindTexture(api.TEXTURE_CUBE_MAP, 0)
}

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
				for f := 0; f < 6; f++ {
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
	for f := 0; f < 6; f++ {
		lin := make([]float32, size*size*3)
		for y := 0; y < size; y++ {
			v := ((float64(y)+0.5)/float64(size))*2.0 - 1.0
			for x := 0; x < size; x++ {
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

func prefilterEnvMip(src [6][]float32, srcDim, dstDim int, roughness float64, sampleCount int) [6][]float32 {
	a := roughness * roughness
	a2 := a * a
	if a2 < 1e-4 {
		a2 = 1e-4
	}
	// Precompute local GGX half-vectors around +Z in tangent space.
	type tangentSample struct{ hx, hy, hz float64 }
	hVecs := make([]tangentSample, sampleCount)
	for i := 0; i < sampleCount; i++ {
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
	for f := 0; f < 6; f++ {
		dst := make([]float32, dstDim*dstDim*3)
		for y := 0; y < dstDim; y++ {
			v := ((float64(y)+0.5)/float64(dstDim))*2.0 - 1.0
			for x := 0; x < dstDim; x++ {
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
				for s := 0; s < sampleCount; s++ {
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

// UploadLightmaps uploads baked 2D lightmap layers to a GPU TEXTURE_2D_ARRAY, or creates
// a 1x1x1 white dummy array when the level has no pre-baked lightmaps.
func (tx *Textures) UploadLightmaps(lms []*config.Lightmap) {
	if tx.lightmapTex != 0 {
		tx.ctx.DeleteTextures(1, &tx.lightmapTex)
		tx.lightmapTex = 0
	}
	tx.ctx.GenTextures(1, &tx.lightmapTex)
	tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tx.lightmapTex)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MIN_FILTER, api.LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAG_FILTER, api.LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)

	if len(lms) == 0 {
		tx.lightmapCount = 0
		white := []byte{255, 255, 255, 255}
		tx.ctx.TexImage3D(api.TEXTURE_2D_ARRAY, 0, api.RGBA8, 1, 1, 1, 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(white))
		tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, 0)
		return
	}

	w, h := int32(lms[0].Width), int32(lms[0].Height)
	if w <= 0 || h <= 0 {
		w, h = 128, 128
	}
	layers := int32(len(lms))
	tx.lightmapCount = layers
	tx.ctx.TexImage3D(api.TEXTURE_2D_ARRAY, 0, api.RGBA8, w, h, layers, 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(nil))
	for i, lm := range lms {
		if lm == nil || len(lm.Data) < int(w*h*4) {
			continue
		}
		tx.ctx.TexSubImage3D(api.TEXTURE_2D_ARRAY, 0, 0, 0, int32(i), w, h, 1, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(lm.Data))
	}
	tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, 0)
}

// idxByDim returns the index of the first slot in the `slots` array with a value greater than or equal to `maxDim`.
func (tx *Textures) idxByDim(maxDim int) int {
	bIdx := len(tx.slots) - 1
	for x := 0; x < bIdx; x++ {
		if maxDim <= tx.slots[x] {
			return x
		}
	}
	return bIdx
}

// computeMipMapLevel calculates the number of mipmap levels based on the maximum dimension of the texture.
func (tx *Textures) computeMipMapLevel(width, height int) int32 {
	maxDim := float64(width)
	if height > width {
		maxDim = float64(height)
	}
	mipCount := int32(math.Floor(math.Log2(maxDim))) + 1
	return mipCount
}

// createTextureArray creates a texture array with specified width, height, and number of layers, and sets up mipmaps and filtering.
func (tx *Textures) createTextureArray(width, height int, layers int32) uint32 {
	var tex uint32
	tx.ctx.GenTextures(1, &tex)
	tx.ctx.BindTexture(api.TEXTURE_2D_ARRAY, tex)

	mipCount := tx.computeMipMapLevel(width, height)

	// Set MipMap limits BEFORE allocation
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_BASE_LEVEL, 0)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAX_LEVEL, mipCount-1)

	// Explicit "Immutable-Style" allocation
	for i := int32(0); i < mipCount; i++ {
		w := int32(width >> i)
		h := int32(height >> i)
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
		tx.ctx.TexImage3D(api.TEXTURE_2D_ARRAY, i, api.RGBA8, w, h, layers, 0, api.RGBA, api.UNSIGNED_BYTE, nil)
	}

	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_MIN_FILTER, gl_api.NEAREST_MIPMAP_NEAREST)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_MAG_FILTER, gl_api.NEAREST)

	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MIN_FILTER, api.LINEAR_MIPMAP_LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAG_FILTER, api.LINEAR)

	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_S, api.REPEAT)
	tx.ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_T, api.REPEAT)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_WRAP_S, gl_api.CLAMP_TO_EDGE)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_WRAP_T, gl_api.CLAMP_TO_EDGE)

	// Anisotropic filtering (capped cleanly to avoid blowing memory bandwidth on integrated GPUs)
	const maxAllowedAniso = float32(4.0)
	var maxAniso float32
	tx.ctx.GetFloatv(api.MAX_TEXTURE_MAX_ANISOTROPY, &maxAniso)
	if maxAniso >= 1.0 {
		if maxAniso > maxAllowedAniso {
			maxAniso = maxAllowedAniso
		}
		tx.ctx.TexParameterf(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAX_ANISOTROPY, maxAniso)
	}

	return tex
}

/*
// nextPowerOfTwo returns the smallest power of two greater than or equal to the given integer v.
func nextPowerOfTwo(v int) int {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
*/
