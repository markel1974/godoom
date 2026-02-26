package lumps

// PictureHeader represents the metadata for a picture, including its dimensions and positional offsets.
type PictureHeader struct {
	Width      int16
	Height     int16
	LeftOffset int16
	TopOffset  int16
}
