package main

import (
	"fmt"
	"os"

	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/script"
	"github.com/markel1974/godoom/mr_tech/generators/wad"
	"github.com/markel1974/godoom/mr_tech/generators/world"
	"github.com/markel1974/godoom/mr_tech/model"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl"
)

const (
	_W        = 640 //1024 //1980 / 2
	_H        = 480 //768  //1080 / 2
	_MaxQueue = 32
)

//TODO MANCA L'UTILIZZO DELL'AABBtree (definito in engine) in Player -> MoveApply

func main() {
	var cfg *model.ConfigRoot
	var err error

	//MODE Define World Mode [0 = legacy, 1 = Generate, 2 = Doom]
	m := 2
	switch m {
	case 0:
		cfg, err = script.ParseScriptData(script.StubOld2)
	case 1:
		cfg, err = world.Generate()
	case 2:
		const levelNumber = 3
		wadFile := "resources" + string(os.PathSeparator) + "wad" + string(os.PathSeparator) + "DOOM.WAD"
		wb := wad.NewBuilder()
		cfg, err = wb.Setup(wadFile, levelNumber)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	en := engine.NewEngine(_W, _H, _MaxQueue)
	if err = en.Setup(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//render := software.NewSoftwareRender()
	render := open_gl.NewOpenGLRender()
	if err = render.Setup(en); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	render.Start()
}
