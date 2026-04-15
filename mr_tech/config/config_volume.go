package config

// Volume represents a 3D volume in a configuration, containing faces, lighting information, and a unique identifier.
type Volume struct {
	Id    string  `json:"id"`
	Faces []*Face `json:"faces"`
	//Light *Light  `json:"light"`
	Tag string `json:"tag"`
}

// NewConfigVolume creates and returns a new instance of Volume with specified ID, light settings, and tag.
func NewConfigVolume(id string, tag string) *Volume {
	return &Volume{
		Id:    id,
		Faces: make([]*Face, 0),
		Tag:   tag,
		//Light: nil,
	}
}

// AddFace adds a Face instance to the Faces slice of the Volume.
func (cv *Volume) AddFace(face *Face) {
	cv.Faces = append(cv.Faces, face)
}

// Scale uniformly scales the geometry of all faces in the Volume by the given scale factor.
func (cv *Volume) Scale(scale float64) {
	for _, face := range cv.Faces {
		for i := range face.Points {
			face.Points[i].Scale(scale)
		}
	}
}
