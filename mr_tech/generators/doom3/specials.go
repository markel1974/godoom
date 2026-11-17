package doom3

import (
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/common"
	"github.com/markel1974/godoom/mr_tech/generators/doom3/proc"
	"github.com/markel1974/godoom/mr_tech/geometry"
)

type TeleportDest struct {
	pos   geometry.XYZ
	angle float64
}

// collectTeleportDestinations indexes teleport destination entities by their Doom 3 entity "name".
func collectTeleportDestinations(entities []*proc.MapEntity) map[string]*TeleportDest {
	destinations := make(map[string]*TeleportDest)
	for _, ent := range entities {
		classname, _ := ent.GetProperty("classname")
		if classname != "info_player_teleport" &&
			classname != "info_teleport_destination" &&
			classname != "target_teleporter" &&
			classname != "misc_teleporter_dest" {
			continue
		}
		name, ok := ent.GetProperty("name")
		if !ok || name == "" {
			continue
		}
		origin, err := ent.GetOrigin()
		if err != nil {
			continue
		}
		destinations[name] = &TeleportDest{
			pos:   origin,
			angle: ent.GetAngle(),
		}
	}
	return destinations
}

// buildDoorOrLift converts a Doom 3 func_door, func_plat, or func_elevator entity (using either
// an inline .proc submodel or an external .lwo model) into a dynamic ThingDoorDef.
func buildDoorOrLift(ent *proc.MapEntity, procModels map[string]*proc.Model, materials *proc.Materials, arc *proc.Pk4, thingsManager *Things) (*config.Thing, bool) {
	if ent.IsDormant() {
		return nil, false
	}
	classname, _ := ent.GetProperty("classname")
	if classname != "func_door" && classname != "func_plat" && classname != "func_elevator" {
		return nil, false
	}

	var cModel *config.Model3DEntry
	modelProp, _ := ent.GetProperty("model")
	nameProp, _ := ent.GetProperty("name")

	var pModel *proc.Model
	if m, ok := procModels[modelProp]; ok {
		pModel = m
	} else if m, ok = procModels[nameProp]; ok {
		pModel = m
	}

	if pModel != nil {
		cModel = bakeProcModel(pModel, ent, materials, arc)
	} else if strings.HasSuffix(strings.ToLower(modelProp), ".lwo") && thingsManager != nil {
		if loaded, err := thingsManager.loadModel(modelProp); err == nil {
			cModel = loaded
		}
	}
	if cModel == nil || len(cModel.Frames) == 0 || len(cModel.Frames[0].Triangles) == 0 {
		return nil, false
	}

	center, size := cModel.CenterWithBounds([]string{"default"})
	worldPos := center
	if pModel == nil {
		if origin, err := ent.GetOrigin(); err == nil {
			worldPos = geometry.XYZ{
				X: origin.X + center.X,
				Y: origin.Y + center.Y,
				Z: origin.Z + center.Z,
			}
		}
	}

	id := nameProp
	if id == "" {
		id = classname
	}
	const heavyMass = 1000.0
	radius := math.Max(size.X, size.Y) * 0.5
	thingCfg := config.NewConfigThing(id, worldPos, 0, config.ThingDoorDef, heavyMass, radius, size.Z, 0)
	thingCfg.GForce = 0
	thingCfg.Model3DEntry = cModel
	_ = common.NewItem(thingCfg)

	if classname == "func_door" {
		thingCfg.Door = buildDoorSpec(ent, size)
	} else {
		thingCfg.Door = buildLiftSpec(ent, size)
	}
	return thingCfg, true
}

// bakeProcModel converts an inline Doom 3 .proc submodel into a world-transformed Model3DEntry.
func bakeProcModel(model *proc.Model, ent *proc.MapEntity, materials *proc.Materials, arc *proc.Pk4) *config.Model3DEntry {
	var allTriangles []config.Model3DEntryTriangle
	for _, surface := range model.Surfaces {
		matName := surface.Material
		texNameLC := strings.ToLower(matName)
		if strings.HasPrefix(texNameLC, "textures/decals/") || strings.HasSuffix(texNameLC, "_shadow") {
			continue
		}
		if materials != nil {
			if def := materials.Resolve(matName); def != nil && def.IsDecal {
				continue
			}
		}

		var material *config.Material
		if materials != nil {
			if matDef := materials.Resolve(matName); matDef != nil {
				if guiMat, _ := ResolveGUIMaterial(arc, materials, matDef, ent); guiMat != nil {
					material = guiMat
				} else {
					material = ResolveConfigMaterial(matDef)
				}
			}
		}
		if material == nil {
			material = config.NewConfigMaterial(matName, []string{texNameLC}, ToMaterial(texNameLC))
			ConfigureMaterial(material, texNameLC, nil)
		}

		for i := 0; i+2 < len(surface.Indices); i += 3 {
			i0, i1, i2 := surface.Indices[i], surface.Indices[i+1], surface.Indices[i+2]
			if i0 >= len(surface.Vertices) || i1 >= len(surface.Vertices) || i2 >= len(surface.Vertices) {
				continue
			}
			v0, v1, v2 := surface.Vertices[i0], surface.Vertices[i1], surface.Vertices[i2]
			tri := config.NewModel3DEntryTriangle(material)
			tri.Vertices[0] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v0.Pos), U: float32(v0.U), V: float32(v0.V)}
			tri.Vertices[1] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v1.Pos), U: float32(v1.U), V: float32(v1.V)}
			tri.Vertices[2] = config.Model3DEntryVertex{Pos: ent.TransformPoint(v2.Pos), U: float32(v2.U), V: float32(v2.V)}
			allTriangles = append(allTriangles, tri)
		}
	}
	model3d := config.NewModel3DEntry(1, []string{"default"}, []string{"death", "die"})
	model3d.Frames[0] = config.NewModel3DEntryFrame(allTriangles)
	return model3d
}

func buildDoorSpec(ent *proc.MapEntity, size geometry.XYZ) *config.DoorSpec {
	var dirX, dirY, dirZ float64
	moveDir := 0.0
	if raw, ok := ent.GetProperty("movedir"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			moveDir = v
		}
	}
	switch moveDir {
	case -1:
		dirZ = 1.0
	case -2:
		dirZ = -1.0
	default:
		rad := moveDir * math.Pi / 180.0
		dirX = math.Cos(rad)
		dirY = math.Sin(rad)
	}

	lip := 8.0
	if raw, ok := ent.GetProperty("lip"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			lip = v
		}
	}
	travel := math.Abs(dirX)*size.X + math.Abs(dirY)*size.Y + math.Abs(dirZ)*size.Z - lip
	if travel <= 0 {
		travel = math.Max(size.X, math.Max(size.Y, size.Z))
	}

	speed := 150.0
	if raw, ok := ent.GetProperty("speed"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			speed = v
		}
	}

	waitSeconds := 3.0
	if raw, ok := ent.GetProperty("wait"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			waitSeconds = v
		}
	}

	startOpen := false
	if raw, ok := ent.GetProperty("start_open"); ok && strings.TrimSpace(raw) == "1" {
		startOpen = true
	}

	requiredKey, _ := ent.GetProperty("requires")
	if requiredKey == "" {
		if locked, ok := ent.GetProperty("locked"); ok && strings.TrimSpace(locked) == "1" {
			requiredKey, _ = ent.GetProperty("key")
		}
	}

	targetName, _ := ent.GetProperty("name")
	target, _ := ent.GetProperty("target")

	return &config.DoorSpec{
		RequiredKey: requiredKey,
		OpenOffset:  geometry.XYZ{X: dirX * travel, Y: dirY * travel, Z: dirZ * travel},
		Speed:       speed,
		WaitSeconds: waitSeconds,
		StartOpen:   startOpen,
		TargetName:  targetName,
		Target:      target,
		Activation:  config.DoorActivateTouch,
	}
}

func buildLiftSpec(ent *proc.MapEntity, size geometry.XYZ) *config.DoorSpec {
	lip := 8.0
	if raw, ok := ent.GetProperty("lip"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			lip = v
		}
	}
	travel := size.Z - lip
	if raw, ok := ent.GetProperty("height"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			travel = v
		}
	}
	if travel <= 0 {
		travel = size.Z
	}
	speed := 150.0
	if raw, ok := ent.GetProperty("speed"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil && v > 0 {
			speed = v
		}
	}
	waitSeconds := 3.0
	if raw, ok := ent.GetProperty("wait"); ok {
		if v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64); err == nil {
			waitSeconds = v
		}
	}
	targetName, _ := ent.GetProperty("name")
	target, _ := ent.GetProperty("target")
	return &config.DoorSpec{
		OpenOffset:  geometry.XYZ{Z: -travel},
		Speed:       speed,
		WaitSeconds: waitSeconds,
		TargetName:  targetName,
		Target:      target,
		Activation:  config.DoorActivateTouch,
	}
}

// buildTeleport creates a ThingTeleportDef from a trigger_teleport (or trigger targeting a teleport dest).
func buildTeleportSpec(ent *proc.MapEntity, procModels map[string]*proc.Model, destinations map[string]*TeleportDest) *config.Thing {
	if ent.IsDormant() {
		return nil
	}
	classname, _ := ent.GetProperty("classname")
	target, ok := ent.GetProperty("target")
	if !ok || strings.TrimSpace(target) == "" {
		return nil
	}
	dest, ok := destinations[target]
	if !ok {
		return nil
	}

	modelProp, _ := ent.GetProperty("model")
	nameProp, _ := ent.GetProperty("name")
	var pModel *proc.Model
	if m, ok := procModels[modelProp]; ok {
		pModel = m
	} else if m, ok := procModels[nameProp]; ok {
		pModel = m
	}

	var pos geometry.XYZ
	radius, height := 32.0, 64.0

	if pModel != nil {
		minPt := geometry.XYZ{X: math.MaxFloat64, Y: math.MaxFloat64, Z: math.MaxFloat64}
		maxPt := geometry.XYZ{X: -math.MaxFloat64, Y: -math.MaxFloat64, Z: -math.MaxFloat64}
		hasVerts := false
		for _, surf := range pModel.Surfaces {
			for _, v := range surf.Vertices {
				p := ent.TransformPoint(v.Pos)
				hasVerts = true
				if p.X < minPt.X {
					minPt.X = p.X
				}
				if p.Y < minPt.Y {
					minPt.Y = p.Y
				}
				if p.Z < minPt.Z {
					minPt.Z = p.Z
				}
				if p.X > maxPt.X {
					maxPt.X = p.X
				}
				if p.Y > maxPt.Y {
					maxPt.Y = p.Y
				}
				if p.Z > maxPt.Z {
					maxPt.Z = p.Z
				}
			}
		}
		if hasVerts {
			pos = geometry.XYZ{
				X: (minPt.X + maxPt.X) * 0.5,
				Y: (minPt.Y + maxPt.Y) * 0.5,
				Z: minPt.Z,
			}
			if r := math.Max((maxPt.X-minPt.X)*0.5, (maxPt.Y-minPt.Y)*0.5); r > 16.0 {
				radius = r
			}
			if h := maxPt.Z - minPt.Z; h > 32.0 {
				height = h
			}
		}
	} else if origin, err := ent.GetOrigin(); err == nil {
		pos = origin
	} else {
		return nil
	}

	id := nameProp
	if id == "" {
		id = classname
	}
	tpCfg := config.NewConfigThing(id, pos, 0, config.ThingTeleportDef, 1000.0, radius, height, 0)
	tpCfg.GForce = 0
	_ = common.NewItem(tpCfg)
	tpCfg.Teleport = &config.TeleportSpec{
		Destination:   dest.pos,
		TargetAngle:   dest.angle,
		ResetVelocity: true,
		Cooldown:      0.5,
	}
	return tpCfg
}
