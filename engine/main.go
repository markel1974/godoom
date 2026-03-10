package main

import (
	"fmt"
	"os"

	"github.com/markel1974/godoom/engine/generators/script"
	"github.com/markel1974/godoom/engine/generators/wad"
	"github.com/markel1974/godoom/engine/generators/world"
	"github.com/markel1974/godoom/engine/model"
	"github.com/markel1974/godoom/engine/portal"
	"github.com/markel1974/godoom/engine/renderers/open_gl"
)

const (
	_W        = 640 //1024 //1980 / 2
	_H        = 480 //768  //1080 / 2
	_MaxQueue = 32
)

func main() {
	var cfg *model.ConfigRoot
	var err error

	//MODE Define World Mode [0 = legacy, 1 = Generate, 2 = Doom]
	m := 2
	switch m {
	case 0:
		cfg, err = script.ParseScriptData(script.StubOld2)
	case 1:
		cfg, err = world.Generate(16, 16)
	case 2:
		const levelNumber = 7
		wadFile := "resources" + string(os.PathSeparator) + "wad" + string(os.PathSeparator) + "DOOM.WAD"
		wb := wad.NewBuilder()
		cfg, err = wb.Setup(wadFile, levelNumber)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	compiler := model.NewCompiler()
	if err = compiler.Setup(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	p := portal.NewPortal(_W, _H, _MaxQueue)
	if err = p.Setup(compiler.GetSectors(), compiler.GetMaxHeight()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	playerSector, err := compiler.Get(cfg.Player.Sector)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	player := model.NewPlayer(cfg.Player, playerSector, false)

	//render := software.NewSoftwareRender()
	render := open_gl.NewOpenGLRender()
	if err = render.Setup(p, player, cfg.Textures); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	render.Start()
}
