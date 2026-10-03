//go:build js && wasm

package main

import (
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/api"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl/web"
)

func getContext(width, height int) api.IContext {
	return web.NewContext(width, height)
}

func waitExit() {
	// In Wasm requestAnimationFrame non è bloccante,
	// dobbiamo fermare l'uscita prematura del programma Go.
	<-make(chan struct{})
}
