package open_gl

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	textures3 "github.com/markel1974/godoom/mr_tech/textures"
)

// Bucket represents a structure used for storing texture-related arrays and metadata for rendering operations.
type Bucket struct {
	Size          int
	Count         int32
	Layer         int32
	DiffuseArray  uint32
	NormalArray   uint32
	EmissiveArray uint32
}

// NewBucket creates and initializes a new Bucket with the specified size and a default count of zero.
func NewBucket(size int) *Bucket {
	return &Bucket{Size: size, Count: 0}
}

// Textures manages a collection of 2D textures and their associated VRAM buckets for efficient rendering and sampling.
type Textures struct {
	ctx           api.IContext
	slots         []int
	buckets       []*Bucket
	textures      map[*textures3.Texture]float32
	envMapTex     uint32
	lightmapTex   uint32
	lightmapCount int32
}

// NewTextures creates and returns a new instance of Textures with an initialized map to store texture references.
func NewTextures(ctx api.IContext) *Textures {
	slots := []int{64, 128, 256, 1024}
	tx := &Textures{
		ctx:      ctx,
		slots:    slots,
		buckets:  make([]*Bucket, len(slots)),
		textures: make(map[*textures3.Texture]float32),
	}
	for i := range tx.slots {
		tx.buckets[i] = NewBucket(tx.slots[i])
	}
	return tx
}

// Setup initializes texture arrays and their respective VRAM buckets for efficient rendering and management.
func (tx *Textures) Setup(t textures3.ITextures) error {
	now := time.Now()
	fmt.Println("starting compiling textures")

	names := t.GetNames()
	if len(names) == 0 {
		return nil
	}
	sort.Strings(names)

	type Usable struct {
		maxDim int
		tex    *textures3.Texture
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
func (tx *Textures) Get(tex *textures3.Texture) (float32, bool) {
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
	// Unità 0-3: Diffuse | 4-7: Normal | 8-11: Emissive
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

// GetLightmapTexture returns the GPU 2D array lightmap texture ID.
func (tx *Textures) GetLightmapTexture() uint32 {
	return tx.lightmapTex
}

// HasLightmaps reports whether real baked lightmaps were uploaded.
func (tx *Textures) HasLightmaps() bool {
	return tx.lightmapCount > 0
}

// UploadEnvironment uploads a 6-face RGBA environment cubemap from config or builds a procedural
// sky/horizon/ground cubemap fallback for metallic and glass Fresnel reflections.
func (tx *Textures) UploadEnvironment(envMaps map[string][6][]byte) {
	if tx.envMapTex != 0 {
		return
	}
	tx.ctx.GenTextures(1, &tx.envMapTex)
	tx.ctx.BindTexture(api.TEXTURE_CUBE_MAP, tx.envMapTex)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_MIN_FILTER, api.LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_MAG_FILTER, api.LINEAR)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_S, api.CLAMP_TO_EDGE)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_T, api.CLAMP_TO_EDGE)
	tx.ctx.TexParameteri(api.TEXTURE_CUBE_MAP, api.TEXTURE_WRAP_R, api.CLAMP_TO_EDGE)

	uploaded := false
	for _, faces := range envMaps {
		faceLen := len(faces[0])
		if faceLen < 4 {
			continue
		}
		dim := int32(math.Sqrt(float64(faceLen / 4)))
		if dim > 0 && int(dim*dim*4) == faceLen {
			valid := true
			for f := 1; f < 6; f++ {
				if len(faces[f]) != faceLen {
					valid = false
					break
				}
			}
			if valid {
				for f := 0; f < 6; f++ {
					tx.ctx.TexImage2D(api.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(f), 0, api.RGBA8, dim, dim, 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(faces[f]))
				}
				uploaded = true
				break
			}
		}
	}

	if !uploaded {
		const size = 64
		buf := make([]byte, size*size*4)
		sunX, sunY, sunZ := 0.44, 0.72, 0.54
		sunL := math.Sqrt(sunX*sunX + sunY*sunY + sunZ*sunZ)
		sunX, sunY, sunZ = sunX/sunL, sunY/sunL, sunZ/sunL
		clampB := func(v float64) byte {
			if v <= 0 {
				return 0
			}
			if v >= 1 {
				return 255
			}
			return byte(v*255.0 + 0.5)
		}
		for f := 0; f < 6; f++ {
			for y := 0; y < size; y++ {
				v := ((float64(y)+0.5)/float64(size))*2.0 - 1.0
				for x := 0; x < size; x++ {
					u := ((float64(x)+0.5)/float64(size))*2.0 - 1.0
					var dx, dy, dz float64
					switch f {
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
					nx, ny, nz := dx*invL, dy*invL, dz*invL
					var r, g, b float64
					if ny >= 0 {
						t := math.Pow(ny, 0.55)
						// Horizon mist -> steel zenith
						r = (1.0-t)*0.56 + t*0.16
						g = (1.0-t)*0.64 + t*0.26
						b = (1.0-t)*0.74 + t*0.42
						sunDot := math.Max(0, nx*sunX+ny*sunY+nz*sunZ)
						sunGlow := math.Pow(sunDot, 12.0)*0.45 + math.Pow(sunDot, 96.0)*0.9
						r += sunGlow * 1.0
						g += sunGlow * 0.92
						b += sunGlow * 0.78
					} else {
						t := math.Pow(-ny, 0.5)
						// Horizon mist -> dark metallic floor bounce
						r = (1.0-t)*0.42 + t*0.10
						g = (1.0-t)*0.48 + t*0.11
						b = (1.0-t)*0.54 + t*0.14
					}
					off := (y*size + x) * 4
					buf[off] = clampB(r)
					buf[off+1] = clampB(g)
					buf[off+2] = clampB(b)
					buf[off+3] = 255
				}
			}
			tx.ctx.TexImage2D(api.TEXTURE_CUBE_MAP_POSITIVE_X+uint32(f), 0, api.RGBA8, size, size, 0, api.RGBA, api.UNSIGNED_BYTE, tx.ctx.Ptr(buf))
		}
	}
	tx.ctx.BindTexture(api.TEXTURE_CUBE_MAP, 0)
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
