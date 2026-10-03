package shaders

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
)

// IAssets represents an interface for handling asset operations such as resolving base paths and reading asset files.
type IAssets interface {
	BasePath(vPath string) string

	Read(p string) ([]byte, error)

	ReadMulti(a string, b string) ([]byte, []byte, error)
}

// ShaderCompile compiles a shader from source code and returns the shader ID or an error if compilation fails.
func ShaderCompile(ctx api.IContext, id string, source string, shaderType uint32) (uint32, error) {
	shader := ctx.CreateShader(shaderType)
	cSources, free := ctx.Strs(source + "\x00")
	ctx.ShaderSource(shader, 1, cSources, nil)
	free()
	ctx.CompileShader(shader)

	var status int32
	ctx.GetShaderiv(shader, api.COMPILE_STATUS, &status)
	if status == api.FALSE {
		var logLength int32
		ctx.GetShaderiv(shader, api.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		ctx.GetShaderInfoLog(shader, logLength, nil, ctx.Str(log))
		return 0, fmt.Errorf("failed to compile shader %s: %v", id, log)
	}
	return shader, nil
}

// ShaderCreateProgram links a vertex and fragment shader into a shader program, validates it, and returns the program ID.
func ShaderCreateProgram(ctx api.IContext, id string, vertexShader uint32, fragmentShader uint32) (uint32, error) {
	//fmt.Printf(
	//	"ShaderCreateProgram: id=%s vertexShader=%d fragmentShader=%d\n",
	//	id,
	//	vertexShader,
	//	fragmentShader,
	//)

	shaderProgram := ctx.CreateProgram()
	ctx.AttachShader(shaderProgram, vertexShader)
	ctx.AttachShader(shaderProgram, fragmentShader)
	ctx.LinkProgram(shaderProgram)
	var status int32
	ctx.GetProgramiv(shaderProgram, api.LINK_STATUS, &status)
	if status == api.FALSE {
		return 0, fmt.Errorf("failed to link shader prg: %s", id)
	}
	ctx.UseProgram(shaderProgram)
	ctx.DeleteShader(fragmentShader)
	ctx.DeleteShader(vertexShader)

	return shaderProgram, nil
}
