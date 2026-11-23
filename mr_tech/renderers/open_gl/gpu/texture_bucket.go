package gpu

// Bucket stores VRAM 2D texture array IDs and layer counts for one mip bucket size.
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
