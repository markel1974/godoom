package config

// ConfigVolume represents a 3D volume in a configuration, containing faces, lighting information, and a unique identifier.
type ConfigVolume struct {
	Id    string        `json:"id"`
	Faces []*ConfigFace `json:"faces"`
	Light *ConfigLight  `json:"light"`
	Tag   string        `json:"tag"`
}

// NewConfigVolume creates and returns a new instance of ConfigVolume with specified ID, light settings, and tag.
func NewConfigVolume(id string, tag string) *ConfigVolume {
	return &ConfigVolume{
		Id:    id,
		Faces: make([]*ConfigFace, 0),
		Tag:   tag,
		Light: nil,
	}
}

// AddFace adds a ConfigFace instance to the Faces slice of the ConfigVolume.
func (cv *ConfigVolume) AddFace(face *ConfigFace) {
	cv.Faces = append(cv.Faces, face)
}
