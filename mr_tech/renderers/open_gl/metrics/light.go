package metrics

// Light represents the properties and attributes of a light source in 3D space, including position, direction, and intensity.
type Light struct {
	X, Y, Z                   float32
	Kind                      float32
	R, G, B, Intensity        float32
	DirX, DirY, DirZ, Falloff float32
	CutOff, OuterCutOff       float32
	Score                     float32
}

// NewLight creates and returns a new instance of the Light structure with default values.
func NewLight() *Light {
	return &Light{}
}

// Assign sets the properties of a Light instance, including position, color, intensity, direction, falloff, and cutoff values.
func (l *Light) Assign(posX, posY, posZ, lightType, colR, colG, colB, intensity, dirX, dirY, dirZ, falloff, cutOff, outerCutOff, score float32) {
	l.X, l.Y, l.Z = posX, posY, posZ
	l.Kind = lightType
	l.R, l.G, l.B = colR, colG, colB
	l.Intensity = intensity
	l.DirX, l.DirY, l.DirZ = dirX, dirY, dirZ
	l.Falloff = falloff
	l.CutOff, l.OuterCutOff = cutOff, outerCutOff
	l.Score = score
}
