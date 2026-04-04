package wolfstein

import "github.com/markel1974/godoom/mr_tech/model/config"

// GetOriginalMapData restituisce un layout 16x16 per il WolfParser.
// 0 = cella navigabile, >0 = muro solido, 90/91 = porte.
func GetOriginalMapData() (int, int, []uint16) {
	width := 16
	height := 16
	mapData := []uint16{
		01, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		1, 0, 0, 2, 2, 0, 90, 0, 0, 3, 3, 3, 3, 0, 0, 1, // Porta 90 tra la stanza sx e dx
		1, 0, 0, 2, 2, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 1,
		1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 3, 0, 0, 1,
		1, 1, 1, 91, 1, 1, 1, 1, 1, 1, 90, 1, 3, 1, 1, 1, // Porta orizzontale 91, porta verticale 90
		1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1,
		1, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 1,
		1, 0, 4, 4, 4, 0, 0, 5, 0, 1, 0, 0, 6, 6, 0, 1,
		1, 0, 0, 0, 4, 0, 0, 5, 0, 90, 0, 0, 6, 6, 0, 1, // Porta 90
		1, 0, 0, 0, 4, 0, 0, 5, 0, 1, 0, 0, 0, 0, 0, 1,
		1, 1, 1, 1, 1, 91, 1, 1, 1, 1, 1, 1, 1, 0, 0, 1, // Porta 91
		1, 0, 0, 0, 0, 0, 0, 5, 0, 0, 0, 0, 0, 0, 0, 1,
		1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
		1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	}
	return width, height, mapData
}

func CreateLevel(level int) (*config.ConfigRoot, error) {
	w, h, data := GetOriginalMapData()
	wp := NewParser(8, 15, true)
	return wp.Parse(w, h, data)
}
