//go:build !js

package main

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/desktop"
)

func getContext(width, height int) api.IContext {
	return desktop.NewContext(width, height)
}

func waitExit() {
	// Su desktop render.Start() blocca già l'esecuzione tramite glfw (w.th.Start),
	// quindi qui non dobbiamo fare nulla.
}
