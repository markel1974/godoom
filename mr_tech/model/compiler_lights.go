package model

import "github.com/markel1974/godoom/mr_tech/config"

// CompilerLights represents a type responsible for compiling and configuring light entities for rendering or simulation.
type CompilerLights struct {
	cLights []*config.Light
}

// NewCompilerLights creates and returns a new instance of CompilerLights.
func NewCompilerLights(cLights []*config.Light) *CompilerLights {
	return &CompilerLights{
		cLights: cLights,
	}
}

// Compile processes and transforms configuration light data into runtime light objects, returning compiled lights or errors.
func (r *CompilerLights) Compile() ([]*Volume, []*Light, error) {
	var out []*Light
	for _, cl := range r.cLights {
		if cl == nil {
			continue
		}
		light := NewLight()
		light.Setup(cl, cl.Pos)
		out = append(out, light)
	}
	return nil, out, nil
}
