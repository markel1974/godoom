package shaders

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/metrics"
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

type fogUniformLocs struct {
	enabled      int32
	color        int32
	sunColor     int32
	params       int32
	heightParams int32
}

func resolveFogUniformLocs(ctx api.IContext, prg uint32) fogUniformLocs {
	return fogUniformLocs{
		enabled:      ctx.GetUniformLocation(prg, ctx.Str("u_fogEnabled\x00")),
		color:        ctx.GetUniformLocation(prg, ctx.Str("u_fogColor\x00")),
		sunColor:     ctx.GetUniformLocation(prg, ctx.Str("u_fogSunColor\x00")),
		params:       ctx.GetUniformLocation(prg, ctx.Str("u_fogParams\x00")),
		heightParams: ctx.GetUniformLocation(prg, ctx.Str("u_fogHeightParams\x00")),
	}
}

func bindFogUniforms(ctx api.IContext, fog *metrics.AtmosphereFog, locs fogUniformLocs) {
	if fog == nil {
		if locs.enabled >= 0 {
			ctx.Uniform1i(locs.enabled, 0)
		}
		return
	}
	fogEn, fogCol, fogSunCol, fogParams, fogHeight := fog.GetShaderParams()
	if locs.enabled >= 0 {
		ctx.Uniform1i(locs.enabled, fogEn)
	}
	if fogEn == 1 {
		bindFogValueUniforms(ctx, locs, fogCol, fogSunCol, fogParams, fogHeight)
	}
}

func bindFogValueUniforms(ctx api.IContext, locs fogUniformLocs, fogCol, fogSunCol [3]float32, fogParams [4]float32, fogHeight [3]float32) {
	if locs.color >= 0 {
		ctx.Uniform3f(locs.color, fogCol[0], fogCol[1], fogCol[2])
	}
	if locs.sunColor >= 0 {
		ctx.Uniform3f(locs.sunColor, fogSunCol[0], fogSunCol[1], fogSunCol[2])
	}
	if locs.params >= 0 {
		ctx.Uniform4f(locs.params, fogParams[0], fogParams[1], fogParams[2], fogParams[3])
	}
	if locs.heightParams >= 0 {
		ctx.Uniform3f(locs.heightParams, fogHeight[0], fogHeight[1], fogHeight[2])
	}
}
