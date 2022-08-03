package pixels

import (
	"errors"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/markel1974/godoom/pixels/executor"
)

type GLShader struct {
	s      *executor.Shader
	vf, uf executor.AttrFormat
	vs, fs string

	uniforms []gsUniformAttr

	uniformDefaults struct {
		transform mgl32.Mat3
		colormask mgl32.Vec4
		bounds    mgl32.Vec4
		texbounds mgl32.Vec4
		cliprect  mgl32.Vec4
	}
}

type gsUniformAttr struct {
	Name      string
	Type      executor.AttrType
	value     interface{}
	ispointer bool
}

const (
	canvasPosition int = iota
	canvasColor
	canvasTexCoords
	canvasIntensity
	canvasClip
)

var defaultCanvasVertexFormat = executor.AttrFormat{
	canvasPosition:  executor.Attr{Name: "aPosition", Type: executor.Vec2},
	canvasColor:     executor.Attr{Name: "aColor", Type: executor.Vec4},
	canvasTexCoords: executor.Attr{Name: "aTexCoords", Type: executor.Vec2},
	canvasIntensity: executor.Attr{Name: "aIntensity", Type: executor.Float},
	canvasClip:      executor.Attr{Name: "aClipRect", Type: executor.Vec4},
}

func NewGLShader(fragmentShader string) *GLShader {
	gs := &GLShader{
		vf: defaultCanvasVertexFormat,
		vs: baseCanvasVertexShader,
		fs: fragmentShader,
	}

	gs.SetUniform("uTransform", &gs.uniformDefaults.transform)
	gs.SetUniform("uColorMask", &gs.uniformDefaults.colormask)
	gs.SetUniform("uBounds", &gs.uniformDefaults.bounds)
	gs.SetUniform("uTexBounds", &gs.uniformDefaults.texbounds)

	gs.Update()

	return gs
}

func (gs *GLShader) Update() {
	gs.uf = make([]executor.Attr, len(gs.uniforms))
	for idx := range gs.uniforms {
		gs.uf[idx] = executor.Attr{
			Name: gs.uniforms[idx].Name,
			Type: gs.uniforms[idx].Type,
		}
	}

	var shader *executor.Shader
	executor.Thread.Call(func() {
		var err error
		shader, err = executor.NewShader(
			gs.vf,
			gs.uf,
			gs.vs,
			gs.fs,
		)
		if err != nil {
			panic(errors.New("failed to create GLCanvas, there's a bug in the shader:" + err.Error()))
		}
	})

	gs.s = shader
}

func (gs *GLShader) getUniform(Name string) int {
	for i, u := range gs.uniforms {
		if u.Name == Name {
			return i
		}
	}
	return -1
}

func (gs *GLShader) SetUniform(name string, value interface{}) {
	t, p := getAttrType(value)
	if loc := gs.getUniform(name); loc > -1 {
		gs.uniforms[loc].Name = name
		gs.uniforms[loc].Type = t
		gs.uniforms[loc].ispointer = p
		gs.uniforms[loc].value = value
		return
	}
	gs.uniforms = append(gs.uniforms, gsUniformAttr{
		Name:      name,
		Type:      t,
		ispointer: p,
		value:     value,
	})
}

func (gu *gsUniformAttr) Value() interface{} {
	if !gu.ispointer {
		return gu.value
	}
	switch gu.Type {
	case executor.Vec2:
		return *gu.value.(*mgl32.Vec2)
	case executor.Vec3:
		return *gu.value.(*mgl32.Vec3)
	case executor.Vec4:
		return *gu.value.(*mgl32.Vec4)
	case executor.Mat2:
		return *gu.value.(*mgl32.Mat2)
	case executor.Mat23:
		return *gu.value.(*mgl32.Mat2x3)
	case executor.Mat24:
		return *gu.value.(*mgl32.Mat2x4)
	case executor.Mat3:
		return *gu.value.(*mgl32.Mat3)
	case executor.Mat32:
		return *gu.value.(*mgl32.Mat3x2)
	case executor.Mat34:
		return *gu.value.(*mgl32.Mat3x4)
	case executor.Mat4:
		return *gu.value.(*mgl32.Mat4)
	case executor.Mat42:
		return *gu.value.(*mgl32.Mat4x2)
	case executor.Mat43:
		return *gu.value.(*mgl32.Mat4x3)
	case executor.Int:
		return *gu.value.(*int32)
	case executor.Float:
		return *gu.value.(*float32)
	default:
		panic("invalid attrtype")
	}
}

func getAttrType(v interface{}) (executor.AttrType, bool) {
	switch v.(type) {
	case int32:
		return executor.Int, false
	case float32:
		return executor.Float, false
	case mgl32.Vec2:
		return executor.Vec2, false
	case mgl32.Vec3:
		return executor.Vec3, false
	case mgl32.Vec4:
		return executor.Vec4, false
	case mgl32.Mat2:
		return executor.Mat2, false
	case mgl32.Mat2x3:
		return executor.Mat23, false
	case mgl32.Mat2x4:
		return executor.Mat24, false
	case mgl32.Mat3:
		return executor.Mat3, false
	case mgl32.Mat3x2:
		return executor.Mat32, false
	case mgl32.Mat3x4:
		return executor.Mat34, false
	case mgl32.Mat4:
		return executor.Mat4, false
	case mgl32.Mat4x2:
		return executor.Mat42, false
	case mgl32.Mat4x3:
		return executor.Mat43, false
	case *mgl32.Vec2:
		return executor.Vec2, true
	case *mgl32.Vec3:
		return executor.Vec3, true
	case *mgl32.Vec4:
		return executor.Vec4, true
	case *mgl32.Mat2:
		return executor.Mat2, true
	case *mgl32.Mat2x3:
		return executor.Mat23, true
	case *mgl32.Mat2x4:
		return executor.Mat24, true
	case *mgl32.Mat3:
		return executor.Mat3, true
	case *mgl32.Mat3x2:
		return executor.Mat32, true
	case *mgl32.Mat3x4:
		return executor.Mat34, true
	case *mgl32.Mat4:
		return executor.Mat4, true
	case *mgl32.Mat4x2:
		return executor.Mat42, true
	case *mgl32.Mat4x3:
		return executor.Mat43, true
	case *int32:
		return executor.Int, true
	case *float32:
		return executor.Float, true
	default:
		panic("invalid AttrType")
	}
}
