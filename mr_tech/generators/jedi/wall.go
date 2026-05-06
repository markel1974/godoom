package jedi

import (
	"fmt"
	"strings"
)

// Wall represents a wall in a level, including its geometry, textures, and related properties.
type Wall struct {
	Id          string
	LeftVertex  int
	RightVertex int
	Adjoin      int
	MidTexture  int
	TopTexture  int
	BotTexture  int
	SignTexture int
	Flags       int
	Light       int
	V1          int
	V2          int
	Overlay     int
	DAdjoin     int
	DMirror     int
	OffsetX     float64
	OffsetY     float64
}

// NewWall initializes a new Wall instance with default values and returns a pointer to it.
func NewWall() *Wall {
	return &Wall{
		LeftVertex:  -1,
		RightVertex: -1,
		Adjoin:      -1,
		MidTexture:  -1,
		TopTexture:  -1,
		BotTexture:  -1,
		SignTexture: -1,
		V1:          -1,
		V2:          -1,
		Flags:       0,
		Light:       0,
		Overlay:     -1,
		DAdjoin:     -1,
		DMirror:     -1,
	}
}

// Parse processes a list of tokens to populate the fields of a Wall instance based on recognized attributes.
func (w *Wall) Parse(tokens []string) {
	for i := 0; i < len(tokens); i++ {
		key := strings.ToUpper(strings.TrimSpace(tokens[i]))
		if !strings.Contains(key, ":") {
			continue
		}
		switch key {
		case "NAME:":
		case "WALL:":
			w.Id, _ = GetTokenStringAt(tokens, 1)
		case "WALK:":
		case "MIRROR:":
		case "LEFT:":
			i++
			leftVertex, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LEFT invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.LeftVertex = leftVertex
			}
		case "RIGHT:":
			i++
			rightVertex, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: RIGHT invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.RightVertex = rightVertex
			}
		case "ADJOIN:":
			i++
			adjoin, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: ADJOIN invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.Adjoin = adjoin
			}
		case "MID:":
			i++
			midTexture, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: MID invalid token id at %d: %s\n", i, err.Error())
			} else {
				//if midTexture == 0 {
				//	midTexture = -1
				//}
				w.MidTexture = midTexture
			}
		case "TOP:":
			i++
			topTexture, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: TOP invalid token id at %d: %s\n", i, err.Error())
			} else {
				//if topTexture == 0 {
				//	topTexture = -1
				//}
				w.TopTexture = topTexture
			}
		case "BOT:":
			i++
			botTexture, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: BOT invalid token id at %d: %s\n", i, err.Error())
			} else {
				//if botTexture == 0 {
				//	botTexture = -1
				//}
				w.BotTexture = botTexture
			}
		case "SIGN:":
			i++
			signTexture, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: SIGN invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.SignTexture = signTexture
			}
		case "FLAGS:":
			i++
			flags, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: FLAGS invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.Flags = flags
			}
		case "LIGHT:":
			i++
			light, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: LIGHT invalid token id at %d: %s\n", i, err.Error())
			} else {
				w.Light = light
			}
		case "V1:":
			i++
			v1, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: V1 invalid token at %d: %s\n", i, err.Error())
			} else {
				w.V1 = v1
				w.LeftVertex = w.V1
			}
		case "V2:":
			i++
			v2, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: V2 invalid token at %d: %s\n", i, err.Error())
			} else {
				w.V2 = v2
				w.RightVertex = w.V2
			}
		case "OVERLAY:":
			i++
			overlay, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: OVERLAY invalid token at %d: %s\n", i, err.Error())
			} else {
				w.Overlay = overlay
			}
		case "DADJOIN:":
			i++
			dAdjoin, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: DADJOIN invalid token at %d: %s\n", i, err.Error())
			} else {
				w.DAdjoin = dAdjoin
			}
		case "DMIRROR:":
			i++
			dMirror, err := GetTokenIntAt(tokens, i)
			if err != nil {
				fmt.Printf("doWall: DADJOIN invalid token at %d: %s\n", i, err.Error())
			} else {
				w.DMirror = dMirror
			}
		default:
			fmt.Println("doWall: Unknown wall attribute: ", key)
		}
	}
}
