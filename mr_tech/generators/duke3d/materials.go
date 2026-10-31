package duke3d

import (
	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/generators/duke3d/lumps"
)

// ConfigureMaterial assigns physically differentiated specular and detail texture properties
// based on Duke Nukem 3D tile IDs and surface flags.
func ConfigureMaterial(mat *config.Material, picNum int, isSky bool) {
	if mat == nil {
		return
	}
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(picNum, isSky, false)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureWallMaterial assigns physical properties to a wall material based on its tile and CStat flags.
func ConfigureWallMaterial(mat *config.Material, wall *lumps.Wall) {
	if mat == nil || wall == nil {
		return
	}
	ConfigureWallMaterialWithTile(mat, wall, int(wall.PicNum))
}

// ConfigureWallMaterialWithTile assigns physical properties to a wall material using a specific tile ID and CStat flags.
func ConfigureWallMaterialWithTile(mat *config.Material, wall *lumps.Wall, tilePic int) {
	if mat == nil || wall == nil {
		return
	}
	isTranslucent := wall.IsTranslucent()
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(tilePic, false, isTranslucent)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// ConfigureSpriteMaterial assigns physical properties to sprite geometry based on sprite tile and properties.
func ConfigureSpriteMaterial(mat *config.Material, sp *lumps.Sprite) {
	if mat == nil || sp == nil {
		return
	}
	picNum := int(sp.PicNum)

	// Blood pools, footprints, dirt decals
	if picNum >= 1111 && picNum <= 1119 {
		mat.SpecularBoost = 0.50
		mat.Shininess = 65.0
		mat.DetailScale = 16.0
		mat.DetailFactor = 0.10
		return
	}

	// Explosive canister / metallic barrels
	if picNum >= 1220 && picNum <= 1250 {
		mat.SpecularBoost = 0.70
		mat.Shininess = 75.0
		mat.DetailScale = 20.0
		mat.DetailFactor = 0.0
		return
	}

	isTrans := sp.IsTransparent()
	spec, shine, dScale, dFactor := HeuristicMaterialProperties(picNum, false, isTrans)
	mat.SpecularBoost = spec
	mat.Shininess = shine
	mat.DetailScale = dScale
	mat.DetailFactor = dFactor
}

// HeuristicMaterialProperties deduces realistic physical specular and detail texture properties
// based on Duke Nukem 3D tile indexing and properties.
func HeuristicMaterialProperties(picNum int, isSky bool, isTranslucent bool) (float64, float64, float32, float32) {
	// Sky & Parallaxing surfaces
	if isSky || (picNum >= 80 && picNum <= 99) {
		return 0.0, 0.0, 1.0, 0.0
	}

	// Mirrors
	if picNum == 560 {
		return 0.95, 128.0, 32.0, 0.04
	}

	// Glass, Windows, Aquarium, Translucent walls
	if isTranslucent || (picNum >= 501 && picNum <= 505) || (picNum >= 510 && picNum <= 540) ||
		(picNum >= 1106 && picNum <= 1110) {
		return 0.90, 120.0, 24.0, 0.08
	}

	// Liquids (Water, Nukage/Acid, Slime, Blood pool)
	if (picNum >= 330 && picNum <= 335) || (picNum >= 1082 && picNum <= 1105) ||
		(picNum >= 200 && picNum <= 205) {
		return 0.80, 95.0, 1.0, 0.0
	}

	// Computer terminals, TVs, Security monitors, Arcade screens, Neon billboards
	if (picNum >= 160 && picNum <= 199) || (picNum >= 490 && picNum <= 500) ||
		(picNum >= 550 && picNum <= 559) || (picNum >= 700 && picNum <= 750) ||
		(picNum >= 900 && picNum <= 950) {
		return 0.82, 105.0, 30.0, 0.12
	}

	// Metals, Steel plates, Elevator doors, Vents, Catwalks, Grates, Girders, Spaceship hulls
	if (picNum >= 206 && picNum <= 249) || (picNum >= 280 && picNum <= 329) ||
		(picNum >= 650 && picNum <= 699) || (picNum >= 1000 && picNum <= 1050) {
		return 0.68, 78.0, 22.0, 0.22
	}

	// Polished Tiles, Bank Marble, Red Casino/Cinema Carpets, Hotel Bathrooms
	if (picNum >= 300 && picNum <= 349) || (picNum >= 800 && picNum <= 840) {
		return 0.42, 55.0, 20.0, 0.16
	}

	// Alien Organic Walls, Egg Pods, Slime Tendrils
	if picNum >= 1120 && picNum <= 1180 {
		return 0.35, 32.0, 14.0, 0.28
	}

	// Wood, Planks, Crates, Cardboard Boxes, Furniture, Wooden Doors
	if (picNum >= 350 && picNum <= 379) || (picNum >= 780 && picNum <= 799) ||
		(picNum >= 850 && picNum <= 870) {
		return 0.12, 16.0, 16.0, 0.26
	}

	// Urban Concrete, City Bricks, Asphalt, Sidewalks, Canyons, Rocks, Plaster
	if (picNum >= 1 && picNum <= 79) || (picNum >= 100 && picNum <= 159) ||
		(picNum >= 250 && picNum <= 279) {
		return 0.05, 10.0, 16.0, 0.38
	}

	// Default fallback
	return 0.15, 25.0, 16.0, 0.25
}
