package open_gl

import (
	"math"
	"sort"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/textures"
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

func NewBucket(size int) *Bucket {
	return &Bucket{Size: size, Count: 0}
}

// Textures manages a collection of 2D textures and their associated VRAM buckets for efficient rendering and sampling.
type Textures struct {
	ctx      api.IContext
	slots    []int
	buckets  []*Bucket
	textures map[*textures.Texture]float32
}

// NewTextures creates and returns a new instance of Textures with an initialized map to store texture references.
func NewTextures(ctx api.IContext) *Textures {
	slots := []int{64, 128, 256, 1024}
	tx := &Textures{
		ctx:      ctx,
		slots:    slots,
		buckets:  make([]*Bucket, len(slots)),
		textures: make(map[*textures.Texture]float32),
	}
	for i := range tx.slots {
		tx.buckets[i] = NewBucket(tx.slots[i])
	}
	return tx
}

// Setup initializes texture arrays and their respective VRAM buckets for efficient rendering and management.
func (tx *Textures) Setup(t textures.ITextures) error {
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
		tn := t.GetMulti([]string{id})
		if len(tn) == 0 {
			continue
		}
		tex := tn[0]
		if tex == nil {
			continue
		}
		w, h := tex.Size()
		if w <= 0 || h <= 0 {
			continue
		}
		maxDim := w
		if h > maxDim {
			maxDim = h
		}
		usable := &Usable{maxDim: maxDim, tex: tex}
		valid = append(valid, usable)
		idx := tx.idxByDim(maxDim)
		tx.buckets[idx].Count++
	}

	// Allocazione VRAM selettiva
	for i := range tx.buckets {
		if tx.buckets[i].Count > 0 {
			tx.buckets[i].DiffuseArray = createTextureArray(tx.ctx, tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].NormalArray = createTextureArray(tx.ctx, tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
			tx.buckets[i].EmissiveArray = createTextureArray(tx.ctx, tx.buckets[i].Size, tx.buckets[i].Size, tx.buckets[i].Count)
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

// createTextureArray creates a texture array with specified width, height, and number of layers, and sets up mipmaps and filtering.
func createTextureArray(ctx api.IContext, width, height int, layers int32) uint32 {
	computeMipMapLevel := func(width, height int) int32 {
		maxDim := float64(width)
		if height > width {
			maxDim = float64(height)
		}
		mipCount := int32(math.Floor(math.Log2(maxDim))) + 1
		return mipCount
	}
	var tex uint32
	ctx.GenTextures(1, &tex)
	ctx.BindTexture(api.TEXTURE_2D_ARRAY, tex)

	mipCount := computeMipMapLevel(width, height)

	// Set MipMap limits BEFORE allocation
	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_BASE_LEVEL, 0)
	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAX_LEVEL, mipCount-1)

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
		ctx.TexImage3D(api.TEXTURE_2D_ARRAY, i, api.RGBA8, w, h, layers, 0, api.RGBA, api.UNSIGNED_BYTE, nil)
	}

	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_MIN_FILTER, gl_api.NEAREST_MIPMAP_NEAREST)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_MAG_FILTER, gl_api.NEAREST)

	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MIN_FILTER, api.LINEAR_MIPMAP_LINEAR)
	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAG_FILTER, api.LINEAR)

	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_S, api.REPEAT)
	ctx.TexParameteri(api.TEXTURE_2D_ARRAY, api.TEXTURE_WRAP_T, api.REPEAT)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_WRAP_S, gl_api.CLAMP_TO_EDGE)
	//tx.ctx.TexParameteri(gl_api.TEXTURE_2D_ARRAY, gl_api.TEXTURE_WRAP_T, gl_api.CLAMP_TO_EDGE)

	// Anisotropic filtering
	var maxAniso float32
	ctx.GetFloatv(api.MAX_TEXTURE_MAX_ANISOTROPY, &maxAniso)
	ctx.TexParameterf(api.TEXTURE_2D_ARRAY, api.TEXTURE_MAX_ANISOTROPY, maxAniso)

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
