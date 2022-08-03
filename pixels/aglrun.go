package pixels

import (
	"errors"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/markel1974/godoom/pixels/executor"
)

func GLRun(run func()) {
	err := glfw.Init()
	if err != nil {
		panic(errors.New("failed to initialize glfw"))
	}
	defer glfw.Terminate()
	executor.Thread.Run(run)
}

func GLGetTime() float64 {
	return glfw.GetTime()
}
