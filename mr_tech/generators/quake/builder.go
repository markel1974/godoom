package quake

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/quake/lumps"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

const gForce = 9.8 * 14

//MODEL IMPORTAL
//model is Z up
//model is CCW

// Builder manages the construction and handling of graphical assets, leveraging a Textures manager for texture operations.
type Builder struct {
}

// NewBuilder initializes and returns a pointer to a new Builder instance with a default Textures manager.
func NewBuilder() *Builder {
	return &Builder{}
}

// Setup initializes the game environment by loading and processing BSP data, textures, entities, and lights from a .pak file.
func (p *Builder) Setup(pakPath string, lev int) (*config.Root, error) {
	if lev < 1 {
		lev = 1
	}
	levelIndex := lev - 1
	//bpsPath := "maps" + lumps.PakSeparator + "e1m" + strconv.Itoa(level) + ".bsp"

	arc, aErr := lumps.NewArchive(pakPath)
	if aErr != nil {
		return nil, aErr
	}

	if err := arc.Setup(pakPath); err != nil {
		return nil, err
	}
	maps, _ := arc.ReadDirFilter("maps", "^e.+\\.bsp")
	if len(maps) == 0 {
		maps, _ = arc.ReadDirFilter("maps", "\\.bsp$") // Fallback for Q2/Q3
	}
	if levelIndex >= len(maps) {
		return nil, fmt.Errorf("level %d out of range for available maps", levelIndex)
	}
	bpsPath := "maps" + lumps.PakSeparator + maps[levelIndex]

	reader, bErr := lumps.NewBSPReader(arc, bpsPath)
	if bErr != nil {
		return nil, bErr
	}
	if err := reader.Setup(); err != nil {
		return nil, err
	}

	texManager := reader.GetTextures()

	cal := config.NewConfigCalibration(0, 0, 0, 0, 0, 0, true)
	//cal.Auto = false
	//cal.OrthoSize = 32092
	//cal.LightCamY = 8000
	//cal.ZNearRoom = 0.1
	//cal.ZFarRoom = 16000

	cal.AspectRatio = 1.0

	scaleFactor := geometry.XYZ{X: 1, Y: 1, Z: 1}
	root := config.NewConfigRoot(cal, nil, nil, nil, scaleFactor, texManager)

	if err := reader.Build(root); err != nil {
		return nil, err
	}

	playerAngle, playerPos := reader.GetPlayerInfo()

	root.Player = config.NewConfigPlayer(playerPos, playerAngle, 100, 1200, 15, 40)
	playerLogic := common.NewPlayer()
	root.Player.OnCollision = playerLogic.OnCollision
	root.Player.OnImpact = playerLogic.OnImpact
	root.Player.GForce = gForce
	root.Player.JumpForce = 1000

	root.Player.Flash.ZFar = 8192
	root.Player.Flash.Factor = 0.02
	root.Player.Flash.Falloff = 2000
	root.Player.Flash.OffsetX = 0.2
	root.Player.Flash.OffsetY = 0.1
	root.Player.Bobbing.SwayScale = 2.0
	root.Player.Bobbing.SwayOffsetX = 50
	root.Player.Bobbing.SwayOffsetY = -0.9
	root.Player.Bobbing.MaxAmplitudeX = 5.0 // MAXIMUM EXCURSION: 12 units (approx. 20% of player height)
	root.Player.Bobbing.MaxAmplitudeY = 5.5
	root.Player.Bobbing.StrideLength = 0.0015 // FREQUENCY: 1000 * 0.0007 = 0.7 rad/frame.
	root.Player.Bobbing.IdleAmpX = 0.9        // Breathing
	root.Player.Bobbing.IdleAmpY = 0.9
	root.Player.Bobbing.IdleDrift = 0.01
	root.Player.Bobbing.SpeedLerp = 0.30 // Instant reactivity to speed
	root.Player.Bobbing.AmpLerp = 0.20
	root.Player.Bobbing.ImpactMax = 1000.0
	root.Player.Bobbing.ImpactScale = 0.02   // LANDING: 1000 * 0.02 = 20 units of vertical shake
	root.Player.Bobbing.SpringTension = 0.20 // Stiffer spring (faster return)
	root.Player.Bobbing.SpringDamping = 0.80
	root.Player.Bobbing.TiltAmp = 0.05

	//fmt.Println("TODO REACTIVATE ROOT THINGS!")
	//root.Things = nil

	return root, nil
}
