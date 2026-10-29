package textures

// Texture represents a 2D texture with metadata and pixel data.
// name is the identifier name of the texture.
// id is the unique identifier for the texture.
// w and h define the width and height of the texture in pixels.
// data holds the actual pixel information as a 2D array of integers.
type Texture struct {
	name         string
	id           uint32
	w            int
	h            int
	diffuse      [][]int
	normal       [][]int
	emissive     [][]int
	autoEmissive bool
	identifier   float32
}

// NewTexture creates a new Texture instance with the given name, ID, width, and height.
func NewTexture(name string, id uint32, w int, h int, autoEmissive bool) *Texture {
	z := &Texture{
		name:         name,
		id:           id,
		w:            w,
		h:            h,
		diffuse:      make([][]int, w),
		autoEmissive: autoEmissive,
		identifier:   -1,
	}
	for i := range z.diffuse {
		z.diffuse[i] = make([]int, h)
	}
	return z
}

// CreateNormal initializes the normal map as a 2D slice with dimensions matching the texture's width and height.
func (t *Texture) CreateNormal() {
	t.normal = make([][]int, t.w)
	for i := range t.normal {
		t.normal[i] = make([]int, t.h)
	}
}

// CreateEmissive initializes the emissive map as a 2D slice with dimensions matching the texture's width and height.
func (t *Texture) CreateEmissive() {
	t.emissive = make([][]int, t.w)
	for i := range t.emissive {
		t.emissive[i] = make([]int, t.h)
	}
}

// GetIdentifier retrieves the unique floating-point identifier associated with the Texture instance.
func (t *Texture) GetIdentifier() float32 {
	return t.identifier
}

// SetIdentifier assigns a new float32 value to the identifier property of the Texture instance.
func (t *Texture) SetIdentifier(identifier float32) {
	t.identifier = identifier
}

// GetSize returns the width and height of the texture as two float64 values.
func (t *Texture) GetSize() (float64, float64) {
	return float64(t.w), float64(t.h)
}

// Get retrieves the color value at the specified wrapped texture coordinates (x, y).
func (t *Texture) Get(x int, y int) int {
	return t.GetDiffuse(x, y)
}

// Set updates the color of the pixel at the specified coordinates (x, y) after wrapping them within texture bounds.
func (t *Texture) Set(x int, y int, color int) {
	t.SetDiffuse(x, y, color)
}

// GetDiffuse retrieves the diffuse color value at the specified wrapped texture coordinates (x, y).
func (t *Texture) GetDiffuse(x int, y int) int {
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		return t.diffuse[x][y]
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	return t.diffuse[x][y]
}

// GetNormal retrieves the normal value at the specified wrapped texture coordinates (x, y).
func (t *Texture) GetNormal(x int, y int) int {
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		return t.normal[x][y]
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	return t.normal[x][y]
}

// GetEmissive retrieves the emissive color value at the specified wrapped texture coordinates (x, y).
func (t *Texture) GetEmissive(x int, y int) int {
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		return t.emissive[x][y]
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	return t.emissive[x][y]
}

// SetDiffuse updates the diffuse color of the pixel at specified coordinates (x, y), wrapping them within texture bounds.
func (t *Texture) SetDiffuse(x int, y int, color int) {
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		t.diffuse[x][y] = color
		return
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	t.diffuse[x][y] = color
}

// SetNormal sets the normal value at the specified coordinates (x, y) after wrapping them within texture bounds.
func (t *Texture) SetNormal(x int, y int, color int) {
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		t.normal[x][y] = color
		return
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	t.normal[x][y] = color
}

func (t *Texture) SetEmissive(x int, y int, color int) {
	if t.emissive == nil {
		return
	}
	if uint(x) < uint(t.w) && uint(y) < uint(t.h) {
		t.emissive[x][y] = color
		return
	}
	x %= t.w
	y %= t.h
	if x < 0 {
		x += t.w
	}
	if y < 0 {
		y += t.h
	}
	t.emissive[x][y] = color
}

// BeginX returns the starting X coordinate for the texture, typically used as an offset in texture mapping.
func (t *Texture) BeginX() int {
	return 0
}

// BeginY returns the starting Y coordinate for the texture, typically used for texture mapping or rendering operations.
func (t *Texture) BeginY() int {
	return 0
}

// Size returns the width and height of the texture as two integer values.
func (t *Texture) Size() (int, int) {
	return t.w, t.h
}

// GetId returns the unique identifier (id) of the Texture instance as a uint32.
func (t *Texture) GetId() uint32 {
	return t.id
}

// GetName returns the name of the texture as a string.
func (t *Texture) GetName() string {
	return t.name
}

func (t *Texture) RGBA() (int, int, []uint8) {
	w, h := t.Size()
	return w, h, t.rgba(t.diffuse)
}

// Diffuse returns the width, height, and a pixel array representing the texture in RGBA format.
func (t *Texture) rgba(data [][]int) []uint8 {
	const stride = 4
	width, height := t.Size()
	pixels := make([]uint8, width*height*stride)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := data[x][y]
			idx := (y*width + x) * stride
			pixels[idx] = uint8(c >> 24)   // R
			pixels[idx+1] = uint8(c >> 16) // G
			pixels[idx+2] = uint8(c >> 8)  // B
			pixels[idx+3] = uint8(c)       // A
		}
	}
	return pixels
}

// Compile generates normal, diffuse, and emissive maps for the texture given a target size in pixels.
func (t *Texture) Compile(size int) ([]uint8, []uint8, []uint8) {
	const stride = 4
	aspectRatio := float64(t.w) / float64(t.h)
	var diffuseScaled []uint8
	var normalScaled []uint8
	var emissiveScaled []uint8
	const fast = false
	if fast {
		diffuseScaled = UpscaleNearest(t.rgba(t.diffuse), t.w, t.h, size, size, stride)
	} else {
		diffuseScaled = UpscaleLanczosSeparable(t.rgba(t.diffuse), t.w, t.h, size, size, stride)
	}
	if len(t.normal) > 0 {
		normalScaled = UpscaleNearest(t.rgba(t.normal), t.w, t.h, size, size, stride)
	} else {
		normalScaled = GenerateNormalMapScharr(diffuseScaled, size, size, stride, 15, 10, 3, aspectRatio)
	}
	if len(t.emissive) > 0 {
		emissiveScaled = UpscaleNearest(t.rgba(t.emissive), t.w, t.h, size, size, stride)
	} else {
		emissiveScaled = diffuseScaled
		if !t.autoEmissive {
			emissiveScaled = CreateBlackPixels(size, size, stride)
		}
	}
	return diffuseScaled, normalScaled, emissiveScaled
}
