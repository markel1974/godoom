package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/engine"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d"
	"github.com/markel1974/godoom/mr_tech/generators/dungeon"
	"github.com/markel1974/godoom/mr_tech/generators/jedi"
	"github.com/markel1974/godoom/mr_tech/generators/quake"
	"github.com/markel1974/godoom/mr_tech/generators/script"
	"github.com/markel1974/godoom/mr_tech/generators/wad"
	"github.com/markel1974/godoom/mr_tech/generators/wolfstein"
	"github.com/markel1974/godoom/mr_tech/model/textures"
	"github.com/markel1974/godoom/mr_tech/physics"
	"github.com/markel1974/godoom/mr_tech/renderers/open_gl"
	"github.com/markel1974/godoom/mr_tech/version"
)

type ModeDef struct {
	Name    string
	Target  string
	Zip     bool
	Builder func(res common.IResources, target string, level int) (*config.Root, error)
}

func IntToBool(i int) bool {
	return i != 0
}

var _availableModes = []ModeDef{
	{
		Name:   "script",
		Target: "resources" + string(os.PathSeparator) + "dungeon" + string(os.PathSeparator) + "textures.zip",
		Zip:    true,
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			p := script.NewBuilder()
			return p.Build(res, script.StubOld2)
		},
	},
	{
		Name:   "dungeon",
		Target: "resources" + string(os.PathSeparator) + "dungeon" + string(os.PathSeparator) + "textures.zip",
		Zip:    true,
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			db := dungeon.NewBuilder()
			return db.Build(res, level)
		},
	},
	{
		Name:   "wolfstein",
		Target: "",
		Zip:    false,
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			p := wolfstein.NewBuilder()
			return p.Build(level)
		},
	},
	{
		Name:   "doom",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "wad" + string(os.PathSeparator) + "DOOM.WAD",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			wb := wad.NewBuilder()
			return wb.Build(res, target, level)
		},
	},
	{
		Name:   "jedi",
		Zip:    true,
		Target: "resources" + string(os.PathSeparator) + "jedi" + string(os.PathSeparator) + "jedi.zip",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			jf := jedi.NewBuilder()
			return jf.Build(res, 0, target, level)
		},
	},
	{
		Name:   "outlaws",
		Zip:    true,
		Target: "resources" + string(os.PathSeparator) + "outlaws" + string(os.PathSeparator) + "outlaws2.zip",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			jf := jedi.NewBuilder()
			return jf.Build(res, 1, target, level)
		},
	},
	{
		Name:   "quake",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "quake" + string(os.PathSeparator) + "pak0.pak",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "quake2",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "quake2" + string(os.PathSeparator) + "pak0.pak",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "quake3",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "quake3arena" + string(os.PathSeparator) + "pak0.pk3",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			qb := quake.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "doom3",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "doom3" + string(os.PathSeparator) + "pak002.pk4",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			qb := doom3.NewBuilder()
			return qb.Setup(res, target, level)
		},
	},
	{
		Name:   "duke3d",
		Zip:    false,
		Target: "resources" + string(os.PathSeparator) + "duke3d" + string(os.PathSeparator) + "DUKE3D.GRP",
		Builder: func(res common.IResources, target string, level int) (*config.Root, error) {
			b := duke3d.NewBuilder()
			return b.Build(res, target, level)
		},
	},
}

func main() {
	var showHelp bool
	var showVersion bool
	var mode int
	var level int
	var width int
	var height int
	var fps int
	var enableShadows int
	var enableShadowLights int

	var helpModes string
	for idx, v := range _availableModes {
		helpModes += fmt.Sprintf("[%d = %s]", idx, v.Name)
	}

	flag.BoolVar(&showHelp, "h", false, "show this help")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.IntVar(&mode, "m", 2, helpModes)
	flag.IntVar(&level, "l", 1, "level number")
	flag.IntVar(&fps, "fps", 120, "fps")
	flag.IntVar(&width, "width", 640, "width")
	flag.IntVar(&height, "height", 480, "height")
	flag.IntVar(&enableShadows, "shadows", 1, "enable shadows")
	flag.IntVar(&enableShadowLights, "shadowLights", 1, "enable shadows lights")

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version.AppName, version.AppVersion)
		return
	}

	if mode < 0 || mode >= len(_availableModes) {
		fmt.Println("Invalid mode")
		return
	}

	gen := _availableModes[mode]

	app := NewApp()
	if err := app.Setup(gen.Zip, gen.Target, width, height, fps); err != nil {
		fmt.Println("Error setting up application:", err)
		return
	}

	textures.SetFps(uint64(fps))
	physics.SetFps(uint64(fps))

	res := app.GetResources()
	target := app.GetPath()
	cfg, err := gen.Builder(res, target, level)
	if err != nil {
		fmt.Println(err)
		return
	}

	en := engine.NewEngine()
	if err = en.Setup(cfg); err != nil {
		fmt.Println(err)
		return
	}

	cal := en.GetCalibration()
	cal.EnableShadows = IntToBool(enableShadows)
	cal.EnableShadowLights = IntToBool(enableShadowLights)

	ctx := app.GetContext(cfg.Player)
	render := open_gl.NewRender(ctx)
	if err = render.Setup(en); err != nil {
		fmt.Println(err)
		return
	}
	render.Start()
	app.WaitExit()
}
