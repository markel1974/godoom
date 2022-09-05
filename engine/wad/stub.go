package wad

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type FooBspElement struct {
	Sector    uint16
	SubSector uint16
	StartX    int16
	StartY    int16
	EndX      int16
	EndY      int16
	Result    uint16
	Tag       string
}


func NewFooBspElement(sector string, subSector string, startX int16, startY int16, endX int16, endY int16, result uint16, tag string) *FooBspElement {
	sectorId, _ := strconv.Atoi(sector)
	subSectorId, _ := strconv.Atoi(subSector)
	return &FooBspElement{
		Sector:    uint16(sectorId),
		SubSector: uint16(subSectorId),
		StartX:    startX,
		StartY:    startY,
		EndX:      endX,
		EndY:      endY,
		Result:    result,
		Tag:       tag,
	}
}


type FooBsp struct {
	Container []*FooBspElement
}

func NewFooBsp() * FooBsp{
	return &FooBsp{}
}

func (f * FooBsp) Add(sector string, subSector string, startX int16, startY int16, endX int16, endY int16, result uint16, tag string) {
	f.Container = append(f.Container, NewFooBspElement(sector, subSector, startX, startY, endX, endY, result, tag))
}

func (f * FooBsp) Print() {
	out, _ := json.MarshalIndent(f, "", " ")
	fmt.Println(string(out))
}

/*
func (f * FooBsp) Verify(level * Level, bsp *BSP) {
	_ = json.Unmarshal([]byte(FooBspStub), f)
	printHeader := func(sector uint16, subSector uint16, tag string, result uint16, length float64) {
		fmt.Println("--------------------------------", subSector, "(", sector, ")")
		fmt.Println("TAG:", tag)
		fmt.Println("LENGTH:", length)
		fmt.Println("Expected:", result)
	}
	checkUnknownError := 0
	checkSameError := 0
	checkMultiError := 0
	unexpected := 0
	for _, c := range f.Container {
		x1 := c.StartX; x2 := c.EndX; y1 := c.StartY; y2 := c.EndY
		length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
		//x := (c.StartX + c.EndX) / 2
		//y := (c.StartY + c.EndY) / 2

		checkSubSector, state, _ := bsp.FindOppositeSubSectorByPoints(c.SubSector, c)
		printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
		fmt.Println(checkSubSector, "==>", state)


		//if c.Result == checkSubSector {
		//	//printHeader(c.Sector, c.SubSector, c.Tag, c.Result)
		//	//fmt.Println("Check Ok:", checkSubSector, "(", checkSector, ")")
		//} else {
		//	if state == -1 {
		//		//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
		//		//fmt.Println("ERROR UNKNOWN Check (segment is unknown you have to remove)")
		//		checkUnknownError++
		//	} else if state == - 2 {
		//		//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
		//		//fmt.Println("ERROR SAME Check (is inside the same sector you have to remove)")
		//		checkSameError++
		//	} else if state == - 3 {
		//		//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
		//		//fmt.Println("ERROR MULTI Check")
		//		checkMultiError++
		//	} else {
		//		printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
		//		out := bsp.TraverseBsp(c.StartX, c.StartY, false)
		//		//outOpposite := bsp.TraverseBsp(c.StartX, c.StartY, true)
		//		fmt.Println("----> Traverse", out)
		//		fmt.Println("ERROR Unexpected (may be correct...)", unexpected)
		//		unexpected++
		//	}
		//}
		// //fmt.Println("Computed:", out, "Computed Opposite:", outOpposite)
	}
	fmt.Println()
	fmt.Println()
	totalError := checkUnknownError + checkSameError + checkMultiError + unexpected
	fmt.Println("TOTAL:", len( f.Container))
	fmt.Println("TOTAL CHECK UNKNOWN ERROR:", checkUnknownError)
	fmt.Println("TOTAL CHECK SAME ERROR:", checkSameError)
	fmt.Println("TOTAL CHECK MULTI ERROR:", checkMultiError)
	fmt.Println("TOTAL UNEXPECTED:", unexpected)
	fmt.Println("TOTAL GOOD:", len( f.Container) - totalError)
	fmt.Println("TOTAL ERROR:", totalError)
}

 */

/*
func (f * FooBsp) VerifyOld(level * Level, bsp *BSP) {
	_ = json.Unmarshal([]byte(FooBspStub), f)

	printHeader := func(sector uint16, subSector uint16, tag string, result uint16, length float64) {
		fmt.Println("--------------------------------", subSector, "(", sector, ")")
		fmt.Println("TAG:", tag)
		fmt.Println("LENGTH:", length)
		fmt.Println("Expected:", result)
	}

	checkUnknownError := 0
	checkSameError := 0
	checkMultiError := 0
	unexpected := 0
	for _, c := range f.Container {
		if c.SubSector == c.Result {
			continue
		}
		x1 := c.StartX; x2 := c.EndX; y1 := c.StartY; y2 := c.EndY
		length := math.Sqrt((float64(x2 - x1) * float64(x2 - x1)) + (float64(y2 - y1) * float64(y2 - y1)))
		//x := (c.StartX + c.EndX) / 2
		//y := (c.StartY + c.EndY) / 2
		//out := bsp.TraverseBsp(x, y, false)
		//outOpposite := bsp.TraverseBsp(x, y, true)

		_, checkSubSector, state := bsp.FindOppositeSubSectorByLine(c.SubSector, c.StartX, c.StartY, c.EndX, c.EndY)
		if c.Result == checkSubSector {
			//printHeader(c.Sector, c.SubSector, c.Tag, c.Result)
			//fmt.Println("Check Ok:", checkSubSector, "(", checkSector, ")")
		} else {
			if state == -1 {
				//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
				//fmt.Println("ERROR UNKNOWN Check (segment is unknown you have to remove)")
				checkUnknownError++
			} else if state == - 2 {
				//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
				//fmt.Println("ERROR SAME Check (is inside the same sector you have to remove)")
				checkSameError++
			} else if state == - 3 {
				//printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
				//fmt.Println("ERROR MULTI Check")
				checkMultiError++
			} else {
				printHeader(c.Sector, c.SubSector, c.Tag, c.Result, length)
				fmt.Println("ERROR Unexpected (may be correct...)", unexpected)
				unexpected++
			}
		}
		//fmt.Println("Computed:", out, "Computed Opposite:", outOpposite)
	}
	fmt.Println()
	fmt.Println()
	totalError := checkUnknownError + checkSameError + checkMultiError + unexpected
	fmt.Println("TOTAL:", len( f.Container))
	fmt.Println("TOTAL CHECK UNKNOWN ERROR:", checkUnknownError)
	fmt.Println("TOTAL CHECK SAME ERROR:", checkSameError)
	fmt.Println("TOTAL CHECK MULTI ERROR:", checkMultiError)
	fmt.Println("TOTAL UNEXPECTED:", unexpected)
	fmt.Println("TOTAL GOOD:", len( f.Container) - totalError)
	fmt.Println("TOTAL ERROR:", totalError)
}

 */


const FooBspStub = `
{
 "Container": [
  {
   "Sector": 47,
   "SubSector": 135,
   "StartX": 2240,
   "StartY": -3776,
   "EndX": 2240,
   "EndY": -3648,
   "Result": 137,
   "Tag": ""
  },
  {
   "Sector": 47,
   "SubSector": 135,
   "StartX": 2240,
   "StartY": -3648,
   "EndX": 2736,
   "EndY": -3648,
   "Result": 134,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged,notOnMap,alreadyOnMap)[---]"
  },
  {
   "Sector": 47,
   "SubSector": 135,
   "StartX": 2240,
   "StartY": -3776,
   "EndX": 2240,
   "EndY": -3648,
   "Result": 137,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 47,
   "SubSector": 135,
   "StartX": 2240,
   "StartY": -3648,
   "EndX": 2736,
   "EndY": -3648,
   "Result": 134,
   "Tag": ""
  },
  {
   "Sector": 61,
   "SubSector": 167,
   "StartX": 2752,
   "StartY": -3048,
   "EndX": 2976,
   "EndY": -3072,
   "Result": 168,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 61,
   "SubSector": 167,
   "StartX": 2752,
   "StartY": -3048,
   "EndX": 2976,
   "EndY": -3072,
   "Result": 168,
   "Tag": ""
  },
  {
   "Sector": 30,
   "SubSector": 72,
   "StartX": -336,
   "StartY": -3168,
   "EndX": -336,
   "EndY": -3296,
   "Result": 73,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 58,
   "SubSector": 194,
   "StartX": 3353,
   "StartY": -3601,
   "EndX": 3360,
   "EndY": -3648,
   "Result": 194,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 97,
   "StartX": 928,
   "StartY": -3392,
   "EndX": 1184,
   "EndY": -3392,
   "Result": 96,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 97,
   "StartX": 1184,
   "StartY": -3392,
   "EndX": 928,
   "EndY": -3392,
   "Result": 97,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 107,
   "StartX": 896,
   "StartY": -3104,
   "EndX": 896,
   "EndY": -3360,
   "Result": 106,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 107,
   "StartX": 704,
   "StartY": -3360,
   "EndX": 704,
   "EndY": -3104,
   "Result": 117,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 41,
   "SubSector": 118,
   "StartX": 496,
   "StartY": -3304,
   "EndX": 496,
   "EndY": -3160,
   "Result": 119,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 41,
   "SubSector": 118,
   "StartX": 496,
   "StartY": -3304,
   "EndX": 496,
   "EndY": -3160,
   "Result": 119,
   "Tag": ""
  },
  {
   "Sector": 72,
   "SubSector": 211,
   "StartX": 2888,
   "StartY": -4320,
   "EndX": 2888,
   "EndY": -4352,
   "Result": 210,
   "Tag": "--(twoSided,impassible)[-BRNBIGR-]"
  },
  {
   "Sector": 72,
   "SubSector": 211,
   "StartX": 2888,
   "StartY": -4192,
   "EndX": 2888,
   "EndY": -4320,
   "Result": 210,
   "Tag": "--(twoSided,impassible)[-BRNBIGC-]"
  },
  {
   "Sector": 72,
   "SubSector": 211,
   "StartX": 2888,
   "StartY": -4160,
   "EndX": 2888,
   "EndY": -4192,
   "Result": 210,
   "Tag": "--(twoSided,impassible)[-BRNBIGL-]"
  },
  {
   "Sector": 72,
   "SubSector": 211,
   "StartX": 2856,
   "StartY": -4352,
   "EndX": 2856,
   "EndY": -4160,
   "Result": 208,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 73,
   "SubSector": 223,
   "StartX": 2944,
   "StartY": -4032,
   "EndX": 2912,
   "EndY": -4128,
   "Result": 223,
   "Tag": ""
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -208,
   "StartY": -3264,
   "EndX": -240,
   "EndY": -3264,
   "Result": 64,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -192,
   "StartY": -3248,
   "EndX": -208,
   "EndY": -3264,
   "Result": 65,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -192,
   "StartY": -3216,
   "EndX": -192,
   "EndY": -3248,
   "Result": 66,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -208,
   "StartY": -3200,
   "EndX": -192,
   "EndY": -3216,
   "Result": 61,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -240,
   "StartY": -3200,
   "EndX": -208,
   "EndY": -3200,
   "Result": 67,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -256,
   "StartY": -3216,
   "EndX": -240,
   "EndY": -3200,
   "Result": 68,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -256,
   "StartY": -3248,
   "EndX": -256,
   "EndY": -3216,
   "Result": 71,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 32,
   "SubSector": 70,
   "StartX": -240,
   "StartY": -3264,
   "EndX": -256,
   "EndY": -3248,
   "Result": 69,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 52,
   "SubSector": 147,
   "StartX": 2208,
   "StartY": -2560,
   "EndX": 2208,
   "EndY": -2304,
   "Result": 146,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 60,
   "SubSector": 173,
   "StartX": 3112,
   "StartY": -3360,
   "EndX": 2816,
   "EndY": -3232,
   "Result": 169,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 60,
   "SubSector": 173,
   "StartX": 2984,
   "StartY": -3200,
   "EndX": 3280,
   "EndY": -3320,
   "Result": 171,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 76,
   "SubSector": 219,
   "StartX": 2944,
   "StartY": -4016,
   "EndX": 3072,
   "EndY": -4016,
   "Result": 217,
   "Tag": ""
  },
  {
   "Sector": 76,
   "SubSector": 219,
   "StartX": 2944,
   "StartY": -4016,
   "EndX": 3072,
   "EndY": -4016,
   "Result": 217,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 76,
   "SubSector": 219,
   "StartX": 3072,
   "StartY": -4032,
   "EndX": 2944,
   "EndY": -4032,
   "Result": 218,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 24,
   "SubSector": 94,
   "StartX": 64,
   "StartY": -3392,
   "EndX": 128,
   "EndY": -3264,
   "Result": 95,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARG3-STARG3]"
  },
  {
   "Sector": 38,
   "SubSector": 102,
   "StartX": 1216,
   "StartY": -3392,
   "EndX": 1344,
   "EndY": -3360,
   "Result": 101,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 37,
   "SubSector": 108,
   "StartX": 928,
   "StartY": -3072,
   "EndX": 1184,
   "EndY": -3072,
   "Result": 109,
   "Tag": ""
  },
  {
   "Sector": 37,
   "SubSector": 108,
   "StartX": 1184,
   "StartY": -3104,
   "EndX": 928,
   "EndY": -3104,
   "Result": 105,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 37,
   "SubSector": 108,
   "StartX": 928,
   "StartY": -3072,
   "EndX": 1184,
   "EndY": -3072,
   "Result": 109,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-COMPUTE2]"
  },
  {
   "Sector": 82,
   "SubSector": 233,
   "StartX": 2992,
   "StartY": -4848,
   "EndX": 3024,
   "EndY": -4848,
   "Result": 235,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 21,
   "SubSector": 50,
   "StartX": 2176,
   "StartY": -3808,
   "EndX": 2048,
   "EndY": -3808,
   "Result": 47,
   "Tag": ""
  },
  {
   "Sector": 21,
   "SubSector": 50,
   "StartX": 2048,
   "StartY": -3776,
   "EndX": 2176,
   "EndY": -3776,
   "Result": 53,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 21,
   "SubSector": 50,
   "StartX": 2176,
   "StartY": -3808,
   "EndX": 2048,
   "EndY": -3808,
   "Result": 47,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 78,
   "SubSector": 226,
   "StartX": 3024,
   "StartY": -4592,
   "EndX": 2992,
   "EndY": -4592,
   "Result": 228,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 30,
   "SubSector": 62,
   "StartX": -128,
   "StartY": -3120,
   "EndX": -256,
   "EndY": -3120,
   "Result": 63,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 42,
   "SubSector": 122,
   "StartX": 320,
   "StartY": -3264,
   "EndX": 288,
   "EndY": -3264,
   "Result": 120,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 42,
   "SubSector": 122,
   "StartX": 320,
   "StartY": -3200,
   "EndX": 320,
   "EndY": -3264,
   "Result": 119,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 42,
   "SubSector": 122,
   "StartX": 288,
   "StartY": -3200,
   "EndX": 320,
   "EndY": -3200,
   "Result": 121,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 42,
   "SubSector": 122,
   "StartX": 288,
   "StartY": -3264,
   "EndX": 288,
   "EndY": -3200,
   "Result": 123,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 16,
   "SubSector": 141,
   "StartX": 2240,
   "StartY": -3968,
   "EndX": 2176,
   "EndY": -3920,
   "Result": 141,
   "Tag": ""
  },
  {
   "Sector": 7,
   "SubSector": 144,
   "StartX": 2496,
   "StartY": -2112,
   "EndX": 2496,
   "EndY": -2496,
   "Result": 154,
   "Tag": "--(twoSided,upperUnpegged)[COMPSPAN-COMPTALL]"
  },
  {
   "Sector": 48,
   "SubSector": 222,
   "StartX": 2368,
   "StartY": -4096,
   "EndX": 2368,
   "EndY": -3968,
   "Result": 220,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 60,
   "SubSector": 168,
   "StartX": 2976,
   "StartY": -3072,
   "EndX": 2752,
   "EndY": -3048,
   "Result": 167,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 72,
   "SubSector": 214,
   "StartX": 3128,
   "StartY": -4320,
   "EndX": 3128,
   "EndY": -4352,
   "Result": 215,
   "Tag": "--(twoSided,impassible)[-BRNBIGR-]"
  },
  {
   "Sector": 72,
   "SubSector": 214,
   "StartX": 3128,
   "StartY": -4160,
   "EndX": 3128,
   "EndY": -4192,
   "Result": 215,
   "Tag": "--(twoSided,impassible)[-BRNBIGL-]"
  },
  {
   "Sector": 72,
   "SubSector": 214,
   "StartX": 3128,
   "StartY": -4192,
   "EndX": 3128,
   "EndY": -4320,
   "Result": 215,
   "Tag": "--(twoSided,impassible)[-BRNBIGC-]"
  },
  {
   "Sector": 72,
   "SubSector": 214,
   "StartX": 3128,
   "StartY": -4320,
   "EndX": 3128,
   "EndY": -4352,
   "Result": 215,
   "Tag": ""
  },
  {
   "Sector": 5,
   "SubSector": 31,
   "StartX": 2040,
   "StartY": -3144,
   "EndX": 1896,
   "EndY": -3104,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 63,
   "SubSector": 186,
   "StartX": 2784,
   "StartY": -3776,
   "EndX": 2784,
   "EndY": -3904,
   "Result": 187,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 63,
   "SubSector": 186,
   "StartX": 2752,
   "StartY": -3904,
   "EndX": 2752,
   "EndY": -3776,
   "Result": 143,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 63,
   "SubSector": 186,
   "StartX": 2784,
   "StartY": -3776,
   "EndX": 2784,
   "EndY": -3904,
   "Result": 187,
   "Tag": ""
  },
  {
   "Sector": 56,
   "SubSector": 190,
   "StartX": 2944,
   "StartY": -3904,
   "EndX": 2944,
   "EndY": -3776,
   "Result": 192,
   "Tag": "--(twoSided,secret)[BROWN96-BROWN96]"
  },
  {
   "Sector": 73,
   "SubSector": 206,
   "StartX": 3104,
   "StartY": -4160,
   "EndX": 2912,
   "EndY": -4160,
   "Result": 205,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 11,
   "SubSector": 26,
   "StartX": 1984,
   "StartY": -2560,
   "EndX": 1792,
   "EndY": -2560,
   "Result": 29,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 1,
   "SubSector": 154,
   "StartX": 2496,
   "StartY": -2496,
   "EndX": 2496,
   "EndY": -2112,
   "Result": 144,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 24,
   "SubSector": 90,
   "StartX": 128,
   "StartY": -3264,
   "EndX": 160,
   "EndY": -3264,
   "Result": 92,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 59,
   "SubSector": 163,
   "StartX": 3312,
   "StartY": -3496,
   "EndX": 3408,
   "EndY": -3432,
   "Result": 162,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 59,
   "SubSector": 163,
   "StartX": 3352,
   "StartY": -3568,
   "EndX": 3312,
   "EndY": -3496,
   "Result": 159,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 59,
   "SubSector": 163,
   "StartX": 3448,
   "StartY": -3520,
   "EndX": 3352,
   "EndY": -3568,
   "Result": 164,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 59,
   "SubSector": 163,
   "StartX": 3352,
   "StartY": -3568,
   "EndX": 3312,
   "EndY": -3496,
   "Result": 159,
   "Tag": ""
  },
  {
   "Sector": 65,
   "SubSector": 188,
   "StartX": 2848,
   "StartY": -3776,
   "EndX": 2848,
   "EndY": -3904,
   "Result": 189,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 65,
   "SubSector": 188,
   "StartX": 2816,
   "StartY": -3904,
   "EndX": 2816,
   "EndY": -3776,
   "Result": 187,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 65,
   "SubSector": 188,
   "StartX": 2848,
   "StartY": -3776,
   "EndX": 2848,
   "EndY": -3904,
   "Result": 189,
   "Tag": ""
  },
  {
   "Sector": 72,
   "SubSector": 210,
   "StartX": 2888,
   "StartY": -4352,
   "EndX": 2888,
   "EndY": -4320,
   "Result": 211,
   "Tag": "--(twoSided,impassible)[-BRNBIGL-]"
  },
  {
   "Sector": 72,
   "SubSector": 210,
   "StartX": 2888,
   "StartY": -4320,
   "EndX": 2888,
   "EndY": -4192,
   "Result": 211,
   "Tag": "--(twoSided,impassible)[-BRNBIGC-]"
  },
  {
   "Sector": 72,
   "SubSector": 210,
   "StartX": 2888,
   "StartY": -4192,
   "EndX": 2888,
   "EndY": -4160,
   "Result": 211,
   "Tag": "--(twoSided,impassible)[-BRNBIGR-]"
  },
  {
   "Sector": 77,
   "SubSector": 220,
   "StartX": 2240,
   "StartY": -4096,
   "EndX": 2240,
   "EndY": -3968,
   "Result": 221,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 77,
   "SubSector": 220,
   "StartX": 2368,
   "StartY": -3968,
   "EndX": 2368,
   "EndY": -4096,
   "Result": 222,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 77,
   "SubSector": 220,
   "StartX": 2240,
   "StartY": -4096,
   "EndX": 2240,
   "EndY": -3968,
   "Result": 221,
   "Tag": ""
  },
  {
   "Sector": 4,
   "SubSector": 4,
   "StartX": 1536,
   "StartY": -2560,
   "EndX": 1536,
   "EndY": -2432,
   "Result": 3,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 4,
   "SubSector": 4,
   "StartX": 1552,
   "StartY": -2432,
   "EndX": 1552,
   "EndY": -2560,
   "Result": 0,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 4,
   "SubSector": 4,
   "StartX": 1552,
   "StartY": -2432,
   "EndX": 1552,
   "EndY": -2560,
   "Result": 0,
   "Tag": ""
  },
  {
   "Sector": 29,
   "SubSector": 74,
   "StartX": -256,
   "StartY": -3136,
   "EndX": -320,
   "EndY": -3168,
   "Result": 74,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 110,
   "StartX": 1184,
   "StartY": -3072,
   "EndX": 1216,
   "EndY": -3072,
   "Result": 110,
   "Tag": ""
  },
  {
   "Sector": 72,
   "SubSector": 205,
   "StartX": 2912,
   "StartY": -4160,
   "EndX": 3104,
   "EndY": -4160,
   "Result": 206,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 72,
   "SubSector": 205,
   "StartX": 3104,
   "StartY": -4352,
   "EndX": 2912,
   "EndY": -4352,
   "Result": 207,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 72,
   "SubSector": 215,
   "StartX": 3128,
   "StartY": -4352,
   "EndX": 3128,
   "EndY": -4320,
   "Result": 214,
   "Tag": "--(twoSided,impassible)[-BRNBIGL-]"
  },
  {
   "Sector": 72,
   "SubSector": 215,
   "StartX": 3128,
   "StartY": -4320,
   "EndX": 3128,
   "EndY": -4192,
   "Result": 214,
   "Tag": ""
  },
  {
   "Sector": 72,
   "SubSector": 215,
   "StartX": 3128,
   "StartY": -4192,
   "EndX": 3128,
   "EndY": -4160,
   "Result": 214,
   "Tag": "--(twoSided,impassible)[-BRNBIGR-]"
  },
  {
   "Sector": 72,
   "SubSector": 215,
   "StartX": 3128,
   "StartY": -4320,
   "EndX": 3128,
   "EndY": -4192,
   "Result": 214,
   "Tag": "--(twoSided,impassible)[-BRNBIGC-]"
  },
  {
   "Sector": 72,
   "SubSector": 215,
   "StartX": 3160,
   "StartY": -4160,
   "EndX": 3160,
   "EndY": -4352,
   "Result": 212,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 2,
   "SubSector": 8,
   "StartX": 1384,
   "StartY": -2592,
   "EndX": 1472,
   "EndY": -2560,
   "Result": 8,
   "Tag": ""
  },
  {
   "Sector": 9,
   "SubSector": 20,
   "StartX": 1792,
   "StartY": -2240,
   "EndX": 1792,
   "EndY": -2304,
   "Result": 22,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 84,
   "SubSector": 236,
   "StartX": 3040,
   "StartY": -4672,
   "EndX": 2976,
   "EndY": -4672,
   "Result": 231,
   "Tag": ""
  },
  {
   "Sector": 84,
   "SubSector": 236,
   "StartX": 2976,
   "StartY": -4648,
   "EndX": 3040,
   "EndY": -4648,
   "Result": 230,
   "Tag": "--(twoSided)[--EXITDOOR]"
  },
  {
   "Sector": 84,
   "SubSector": 236,
   "StartX": 3040,
   "StartY": -4672,
   "EndX": 2976,
   "EndY": -4672,
   "Result": 231,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 24,
   "SubSector": 120,
   "StartX": 256,
   "StartY": -3264,
   "EndX": 288,
   "EndY": -3264,
   "Result": 123,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 120,
   "StartX": 288,
   "StartY": -3264,
   "EndX": 320,
   "EndY": -3264,
   "Result": 122,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 120,
   "StartX": 320,
   "StartY": -3328,
   "EndX": 256,
   "EndY": -3328,
   "Result": 126,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 61,
   "SubSector": 169,
   "StartX": 3112,
   "StartY": -3360,
   "EndX": 2944,
   "EndY": -3536,
   "Result": 170,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 61,
   "SubSector": 169,
   "StartX": 2816,
   "StartY": -3232,
   "EndX": 3112,
   "EndY": -3360,
   "Result": 173,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 61,
   "SubSector": 169,
   "StartX": 3112,
   "StartY": -3360,
   "EndX": 2944,
   "EndY": -3536,
   "Result": 170,
   "Tag": ""
  },
  {
   "Sector": 61,
   "SubSector": 169,
   "StartX": 2944,
   "StartY": -3536,
   "EndX": 2752,
   "EndY": -3360,
   "Result": 155,
   "Tag": "--(twoSided,notOnMap,alreadyOnMap)[---]"
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 256,
   "StartY": -3264,
   "EndX": 224,
   "EndY": -3264,
   "Result": 54,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 224,
   "StartY": -3264,
   "EndX": 224,
   "EndY": -3200,
   "Result": 57,
   "Tag": ""
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 224,
   "StartY": -3200,
   "EndX": 256,
   "EndY": -3200,
   "Result": 55,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 224,
   "StartY": -3264,
   "EndX": 224,
   "EndY": -3200,
   "Result": 57,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 224,
   "StartY": -3200,
   "EndX": 256,
   "EndY": -3200,
   "Result": 55,
   "Tag": ""
  },
  {
   "Sector": 25,
   "SubSector": 56,
   "StartX": 256,
   "StartY": -3200,
   "EndX": 256,
   "EndY": -3264,
   "Result": 123,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 82,
   "SubSector": 234,
   "StartX": 2992,
   "StartY": -4840,
   "EndX": 2992,
   "EndY": -4848,
   "Result": 235,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 23,
   "SubSector": 53,
   "StartX": 2048,
   "StartY": -3680,
   "EndX": 2176,
   "EndY": -3680,
   "Result": 51,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 23,
   "SubSector": 53,
   "StartX": 2176,
   "StartY": -3776,
   "EndX": 2048,
   "EndY": -3776,
   "Result": 50,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 23,
   "SubSector": 53,
   "StartX": 2176,
   "StartY": -3776,
   "EndX": 2048,
   "EndY": -3776,
   "Result": 50,
   "Tag": ""
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 192,
   "StartY": -3264,
   "EndX": 160,
   "EndY": -3264,
   "Result": 54,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 160,
   "StartY": -3264,
   "EndX": 160,
   "EndY": -3200,
   "Result": 92,
   "Tag": ""
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 160,
   "StartY": -3200,
   "EndX": 192,
   "EndY": -3200,
   "Result": 55,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 160,
   "StartY": -3264,
   "EndX": 160,
   "EndY": -3200,
   "Result": 92,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 160,
   "StartY": -3200,
   "EndX": 192,
   "EndY": -3200,
   "Result": 55,
   "Tag": ""
  },
  {
   "Sector": 27,
   "SubSector": 58,
   "StartX": 192,
   "StartY": -3200,
   "EndX": 192,
   "EndY": -3264,
   "Result": 57,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 74,
   "SubSector": 207,
   "StartX": 2912,
   "StartY": -4352,
   "EndX": 3104,
   "EndY": -4352,
   "Result": 205,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 73,
   "SubSector": 218,
   "StartX": 2944,
   "StartY": -4032,
   "EndX": 3072,
   "EndY": -4032,
   "Result": 219,
   "Tag": "--(twoSided)[--BIGDOOR4]"
  },
  {
   "Sector": 8,
   "SubSector": 10,
   "StartX": 2176,
   "StartY": -2304,
   "EndX": 2176,
   "EndY": -2560,
   "Result": 146,
   "Tag": ""
  },
  {
   "Sector": 8,
   "SubSector": 10,
   "StartX": 2144,
   "StartY": -2560,
   "EndX": 2144,
   "EndY": -2304,
   "Result": 9,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 8,
   "SubSector": 10,
   "StartX": 2176,
   "StartY": -2304,
   "EndX": 2176,
   "EndY": -2560,
   "Result": 146,
   "Tag": "--(twoSided,upperUnpegged)[STEP1--]"
  },
  {
   "Sector": 5,
   "SubSector": 35,
   "StartX": 1544,
   "StartY": -3384,
   "EndX": 1784,
   "EndY": -3448,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 35,
   "StartX": 1784,
   "StartY": -3448,
   "EndX": 1544,
   "EndY": -3384,
   "Result": 35,
   "Tag": ""
  },
  {
   "Sector": 7,
   "SubSector": 9,
   "StartX": 2144,
   "StartY": -2304,
   "EndX": 2144,
   "EndY": -2560,
   "Result": 10,
   "Tag": ""
  },
  {
   "Sector": 7,
   "SubSector": 9,
   "StartX": 2144,
   "StartY": -2304,
   "EndX": 2144,
   "EndY": -2560,
   "Result": 10,
   "Tag": "--(twoSided,upperUnpegged)[STEP1-COMPTILE]"
  },
  {
   "Sector": 31,
   "SubSector": 63,
   "StartX": -128,
   "StartY": -3136,
   "EndX": -256,
   "EndY": -3136,
   "Result": 61,
   "Tag": ""
  },
  {
   "Sector": 31,
   "SubSector": 63,
   "StartX": -128,
   "StartY": -3136,
   "EndX": -256,
   "EndY": -3136,
   "Result": 61,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 31,
   "SubSector": 63,
   "StartX": -256,
   "StartY": -3120,
   "EndX": -128,
   "EndY": -3120,
   "Result": 62,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 288,
   "StartY": -3264,
   "EndX": 256,
   "EndY": -3264,
   "Result": 120,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 256,
   "StartY": -3264,
   "EndX": 256,
   "EndY": -3200,
   "Result": 56,
   "Tag": ""
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 256,
   "StartY": -3200,
   "EndX": 288,
   "EndY": -3200,
   "Result": 121,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 256,
   "StartY": -3264,
   "EndX": 256,
   "EndY": -3200,
   "Result": 56,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 256,
   "StartY": -3200,
   "EndX": 288,
   "EndY": -3200,
   "Result": 121,
   "Tag": ""
  },
  {
   "Sector": 43,
   "SubSector": 123,
   "StartX": 288,
   "StartY": -3200,
   "EndX": 288,
   "EndY": -3264,
   "Result": 122,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 55,
   "SubSector": 155,
   "StartX": 2752,
   "StartY": -3360,
   "EndX": 2944,
   "EndY": -3536,
   "Result": 169,
   "Tag": "--(twoSided,notOnMap,alreadyOnMap)[NUKE24--]"
  },
  {
   "Sector": 55,
   "SubSector": 155,
   "StartX": 2944,
   "StartY": -3536,
   "EndX": 2752,
   "EndY": -3584,
   "Result": 156,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 74,
   "SubSector": 212,
   "StartX": 3160,
   "StartY": -4352,
   "EndX": 3160,
   "EndY": -4160,
   "Result": 215,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 9,
   "SubSector": 21,
   "StartX": 1792,
   "StartY": -2304,
   "EndX": 1984,
   "EndY": -2304,
   "Result": 22,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 10,
   "SubSector": 22,
   "StartX": 1984,
   "StartY": -2240,
   "EndX": 1984,
   "EndY": -2304,
   "Result": 18,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 10,
   "SubSector": 22,
   "StartX": 1792,
   "StartY": -2240,
   "EndX": 1984,
   "EndY": -2240,
   "Result": 19,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 10,
   "SubSector": 22,
   "StartX": 1792,
   "StartY": -2304,
   "EndX": 1792,
   "EndY": -2240,
   "Result": 20,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 10,
   "SubSector": 22,
   "StartX": 1984,
   "StartY": -2304,
   "EndX": 1792,
   "EndY": -2304,
   "Result": 21,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 134,
   "StartX": 2736,
   "StartY": -3648,
   "EndX": 2240,
   "EndY": -3648,
   "Result": 135,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged,notOnMap,alreadyOnMap)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 58,
   "SubSector": 193,
   "StartX": 3328,
   "StartY": -3968,
   "EndX": 3328,
   "EndY": -3744,
   "Result": 195,
   "Tag": ""
  },
  {
   "Sector": 58,
   "SubSector": 193,
   "StartX": 3328,
   "StartY": -3968,
   "EndX": 3328,
   "EndY": -3744,
   "Result": 195,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 58,
   "SubSector": 193,
   "StartX": 3520,
   "StartY": -3840,
   "EndX": 3520,
   "EndY": -3904,
   "Result": 196,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 75,
   "SubSector": 217,
   "StartX": 2944,
   "StartY": -4000,
   "EndX": 3072,
   "EndY": -4000,
   "Result": 216,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 75,
   "SubSector": 217,
   "StartX": 2944,
   "StartY": -4000,
   "EndX": 3072,
   "EndY": -4000,
   "Result": 216,
   "Tag": ""
  },
  {
   "Sector": 75,
   "SubSector": 217,
   "StartX": 3072,
   "StartY": -4016,
   "EndX": 2944,
   "EndY": -4016,
   "Result": 219,
   "Tag": "--(twoSided)[--BIGDOOR4]"
  },
  {
   "Sector": 82,
   "SubSector": 231,
   "StartX": 2976,
   "StartY": -4672,
   "EndX": 3040,
   "EndY": -4672,
   "Result": 236,
   "Tag": "--(twoSided,upperUnpegged)[--STARTAN1]"
  },
  {
   "Sector": 82,
   "SubSector": 231,
   "StartX": 3024,
   "StartY": -4840,
   "EndX": 2992,
   "EndY": -4840,
   "Result": 235,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 6,
   "SubSector": 6,
   "StartX": 1664,
   "StartY": -2624,
   "EndX": 1664,
   "EndY": -2752,
   "Result": 23,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 12,
   "SubSector": 29,
   "StartX": 1984,
   "StartY": -2560,
   "EndX": 1984,
   "EndY": -2624,
   "Result": 25,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 12,
   "SubSector": 29,
   "StartX": 1792,
   "StartY": -2560,
   "EndX": 1984,
   "EndY": -2560,
   "Result": 26,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 12,
   "SubSector": 29,
   "StartX": 1792,
   "StartY": -2624,
   "EndX": 1792,
   "EndY": -2560,
   "Result": 27,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 12,
   "SubSector": 29,
   "StartX": 1984,
   "StartY": -2624,
   "EndX": 1792,
   "EndY": -2624,
   "Result": 28,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 114,
   "StartX": 1344,
   "StartY": -3104,
   "EndX": 1216,
   "EndY": -3072,
   "Result": 111,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 114,
   "StartX": 1216,
   "StartY": -2880,
   "EndX": 1344,
   "EndY": -2880,
   "Result": 116,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 132,
   "StartX": 2432,
   "StartY": -3112,
   "EndX": 2272,
   "EndY": -3008,
   "Result": 132,
   "Tag": ""
  },
  {
   "Sector": 29,
   "SubSector": 64,
   "StartX": -240,
   "StartY": -3264,
   "EndX": -208,
   "EndY": -3264,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 64,
   "StartX": -208,
   "StartY": -3264,
   "EndX": -240,
   "EndY": -3264,
   "Result": 64,
   "Tag": ""
  },
  {
   "Sector": 2,
   "SubSector": 116,
   "StartX": 1344,
   "StartY": -2880,
   "EndX": 1216,
   "EndY": -2880,
   "Result": 114,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 45,
   "SubSector": 129,
   "StartX": 320,
   "StartY": -3136,
   "EndX": 256,
   "EndY": -3136,
   "Result": 121,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 45,
   "SubSector": 129,
   "StartX": 320,
   "StartY": -3072,
   "EndX": 320,
   "EndY": -3136,
   "Result": 127,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 45,
   "SubSector": 129,
   "StartX": 256,
   "StartY": -3072,
   "EndX": 320,
   "EndY": -3072,
   "Result": 128,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 45,
   "SubSector": 129,
   "StartX": 256,
   "StartY": -3136,
   "EndX": 256,
   "EndY": -3072,
   "Result": 55,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 35,
   "SubSector": 84,
   "StartX": -64,
   "StartY": -3328,
   "EndX": -64,
   "EndY": -3136,
   "Result": 83,
   "Tag": ""
  },
  {
   "Sector": 35,
   "SubSector": 84,
   "StartX": -64,
   "StartY": -3328,
   "EndX": -64,
   "EndY": -3136,
   "Result": 83,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 2992,
   "StartY": -4592,
   "EndX": 3024,
   "EndY": -4592,
   "Result": 226,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 3024,
   "StartY": -4592,
   "EndX": 3024,
   "EndY": -4600,
   "Result": 225,
   "Tag": ""
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 3024,
   "StartY": -4600,
   "EndX": 2992,
   "EndY": -4600,
   "Result": 224,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 3024,
   "StartY": -4592,
   "EndX": 3024,
   "EndY": -4600,
   "Result": 225,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 3024,
   "StartY": -4600,
   "EndX": 2992,
   "EndY": -4600,
   "Result": 224,
   "Tag": ""
  },
  {
   "Sector": 79,
   "SubSector": 228,
   "StartX": 2992,
   "StartY": -4600,
   "EndX": 2992,
   "EndY": -4592,
   "Result": 227,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 29,
   "SubSector": 83,
   "StartX": -64,
   "StartY": -3136,
   "EndX": -64,
   "EndY": -3328,
   "Result": 84,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARG3-STARG3]"
  },
  {
   "Sector": 29,
   "SubSector": 83,
   "StartX": -64,
   "StartY": -3328,
   "EndX": -64,
   "EndY": -3136,
   "Result": 83,
   "Tag": ""
  },
  {
   "Sector": 60,
   "SubSector": 177,
   "StartX": 3304,
   "StartY": -3040,
   "EndX": 3136,
   "EndY": -3072,
   "Result": 174,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 69,
   "SubSector": 195,
   "StartX": 3328,
   "StartY": -3744,
   "EndX": 3328,
   "EndY": -3968,
   "Result": 193,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 71,
   "SubSector": 203,
   "StartX": 2880,
   "StartY": -2880,
   "EndX": 2880,
   "EndY": -2912,
   "Result": 203,
   "Tag": ""
  },
  {
   "Sector": 80,
   "SubSector": 229,
   "StartX": 2976,
   "StartY": -4608,
   "EndX": 3040,
   "EndY": -4608,
   "Result": 224,
   "Tag": ""
  },
  {
   "Sector": 80,
   "SubSector": 229,
   "StartX": 3040,
   "StartY": -4632,
   "EndX": 2976,
   "EndY": -4632,
   "Result": 230,
   "Tag": "--(twoSided)[--EXITDOOR]"
  },
  {
   "Sector": 80,
   "SubSector": 229,
   "StartX": 2976,
   "StartY": -4608,
   "EndX": 3040,
   "EndY": -4608,
   "Result": 224,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 29,
   "SubSector": 66,
   "StartX": -192,
   "StartY": -3248,
   "EndX": -192,
   "EndY": -3216,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 66,
   "StartX": -192,
   "StartY": -3216,
   "EndX": -192,
   "EndY": -3248,
   "Result": 66,
   "Tag": ""
  },
  {
   "Sector": 5,
   "SubSector": 37,
   "StartX": 1672,
   "StartY": -3104,
   "EndX": 1520,
   "EndY": -3168,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 37,
   "StartX": 1520,
   "StartY": -3168,
   "EndX": 1672,
   "EndY": -3104,
   "Result": 37,
   "Tag": ""
  },
  {
   "Sector": 5,
   "SubSector": 39,
   "StartX": 1376,
   "StartY": -3200,
   "EndX": 1376,
   "EndY": -3104,
   "Result": 43,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 5,
   "SubSector": 39,
   "StartX": 1376,
   "StartY": -3360,
   "EndX": 1376,
   "EndY": -3264,
   "Result": 42,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 5,
   "SubSector": 39,
   "StartX": 1520,
   "StartY": -3168,
   "EndX": 1544,
   "EndY": -3384,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 50,
   "SubSector": 143,
   "StartX": 2752,
   "StartY": -3776,
   "EndX": 2752,
   "EndY": -3904,
   "Result": 186,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 50,
   "SubSector": 143,
   "StartX": 2720,
   "StartY": -3904,
   "EndX": 2688,
   "EndY": -3776,
   "Result": 142,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 50,
   "SubSector": 143,
   "StartX": 2752,
   "StartY": -3776,
   "EndX": 2752,
   "EndY": -3904,
   "Result": 186,
   "Tag": ""
  },
  {
   "Sector": 58,
   "SubSector": 164,
   "StartX": 3352,
   "StartY": -3568,
   "EndX": 3448,
   "EndY": -3520,
   "Result": 163,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN--]"
  },
  {
   "Sector": 58,
   "SubSector": 164,
   "StartX": 3448,
   "StartY": -3520,
   "EndX": 3352,
   "EndY": -3568,
   "Result": 164,
   "Tag": ""
  },
  {
   "Sector": 62,
   "SubSector": 180,
   "StartX": 3345,
   "StartY": -2939,
   "EndX": 3320,
   "EndY": -3040,
   "Result": 180,
   "Tag": ""
  },
  {
   "Sector": 2,
   "SubSector": 2,
   "StartX": 1472,
   "StartY": -2432,
   "EndX": 1472,
   "EndY": -2560,
   "Result": 3,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 224,
   "StartY": -3264,
   "EndX": 192,
   "EndY": -3264,
   "Result": 54,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 192,
   "StartY": -3264,
   "EndX": 192,
   "EndY": -3200,
   "Result": 58,
   "Tag": ""
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 192,
   "StartY": -3200,
   "EndX": 224,
   "EndY": -3200,
   "Result": 55,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 192,
   "StartY": -3264,
   "EndX": 192,
   "EndY": -3200,
   "Result": 58,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 192,
   "StartY": -3200,
   "EndX": 224,
   "EndY": -3200,
   "Result": 55,
   "Tag": ""
  },
  {
   "Sector": 26,
   "SubSector": 57,
   "StartX": 224,
   "StartY": -3200,
   "EndX": 224,
   "EndY": -3264,
   "Result": 56,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 29,
   "SubSector": 67,
   "StartX": -208,
   "StartY": -3200,
   "EndX": -240,
   "EndY": -3200,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 67,
   "StartX": -240,
   "StartY": -3200,
   "EndX": -208,
   "EndY": -3200,
   "Result": 67,
   "Tag": ""
  },
  {
   "Sector": 29,
   "SubSector": 80,
   "StartX": -320,
   "StartY": -3296,
   "EndX": -256,
   "EndY": -3328,
   "Result": 80,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 99,
   "StartX": 928,
   "StartY": -3392,
   "EndX": 896,
   "EndY": -3392,
   "Result": 99,
   "Tag": ""
  },
  {
   "Sector": 39,
   "SubSector": 105,
   "StartX": 1344,
   "StartY": -3264,
   "EndX": 1344,
   "EndY": -3360,
   "Result": 42,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 39,
   "SubSector": 105,
   "StartX": 1344,
   "StartY": -3104,
   "EndX": 1344,
   "EndY": -3200,
   "Result": 43,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 39,
   "SubSector": 105,
   "StartX": 928,
   "StartY": -3104,
   "EndX": 1184,
   "EndY": -3104,
   "Result": 108,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 39,
   "SubSector": 105,
   "StartX": 1184,
   "StartY": -3360,
   "EndX": 928,
   "EndY": -3360,
   "Result": 96,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 39,
   "SubSector": 105,
   "StartX": 928,
   "StartY": -3360,
   "EndX": 928,
   "EndY": -3104,
   "Result": 106,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 49,
   "SubSector": 142,
   "StartX": 2688,
   "StartY": -3776,
   "EndX": 2720,
   "EndY": -3904,
   "Result": 143,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 49,
   "SubSector": 142,
   "StartX": 2688,
   "StartY": -3920,
   "EndX": 2632,
   "EndY": -3792,
   "Result": 139,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged)[--BROWN1]"
  },
  {
   "Sector": 49,
   "SubSector": 142,
   "StartX": 2688,
   "StartY": -3776,
   "EndX": 2720,
   "EndY": -3904,
   "Result": 143,
   "Tag": ""
  },
  {
   "Sector": 5,
   "SubSector": 131,
   "StartX": 2736,
   "StartY": -3112,
   "EndX": 2736,
   "EndY": -3360,
   "Result": 133,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 78,
   "SubSector": 224,
   "StartX": 3040,
   "StartY": -4608,
   "EndX": 2976,
   "EndY": -4608,
   "Result": 229,
   "Tag": ""
  },
  {
   "Sector": 78,
   "SubSector": 224,
   "StartX": 2992,
   "StartY": -4600,
   "EndX": 3024,
   "EndY": -4600,
   "Result": 228,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 78,
   "SubSector": 224,
   "StartX": 3040,
   "StartY": -4608,
   "EndX": 2976,
   "EndY": -4608,
   "Result": 229,
   "Tag": "--(twoSided,upperUnpegged)[--STARTAN1]"
  },
  {
   "Sector": 62,
   "SubSector": 175,
   "StartX": 3400,
   "StartY": -3152,
   "EndX": 3304,
   "EndY": -3040,
   "Result": 174,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged,lowerUnpegged)[--BROWNGRN]"
  },
  {
   "Sector": 62,
   "SubSector": 175,
   "StartX": 3304,
   "StartY": -3040,
   "EndX": 3400,
   "EndY": -3152,
   "Result": 175,
   "Tag": ""
  },
  {
   "Sector": 71,
   "SubSector": 201,
   "StartX": 2752,
   "StartY": -2784,
   "EndX": 2944,
   "EndY": -2656,
   "Result": 204,
   "Tag": "--(twoSided)[BROWN1-BROWN1]"
  },
  {
   "Sector": 71,
   "SubSector": 201,
   "StartX": 2752,
   "StartY": -2784,
   "EndX": 2944,
   "EndY": -2656,
   "Result": 204,
   "Tag": ""
  },
  {
   "Sector": 15,
   "SubSector": 43,
   "StartX": 1344,
   "StartY": -3200,
   "EndX": 1344,
   "EndY": -3104,
   "Result": 105,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 15,
   "SubSector": 43,
   "StartX": 1376,
   "StartY": -3104,
   "EndX": 1376,
   "EndY": -3200,
   "Result": 39,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 15,
   "SubSector": 43,
   "StartX": 1344,
   "StartY": -3200,
   "EndX": 1344,
   "EndY": -3104,
   "Result": 105,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 109,
   "StartX": 1184,
   "StartY": -3072,
   "EndX": 928,
   "EndY": -3072,
   "Result": 108,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 109,
   "StartX": 928,
   "StartY": -3072,
   "EndX": 1184,
   "EndY": -3072,
   "Result": 109,
   "Tag": ""
  },
  {
   "Sector": 57,
   "SubSector": 171,
   "StartX": 3280,
   "StartY": -3320,
   "EndX": 2984,
   "EndY": -3200,
   "Result": 173,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 57,
   "SubSector": 171,
   "StartX": 2984,
   "StartY": -3200,
   "EndX": 3136,
   "EndY": -3072,
   "Result": 172,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 62,
   "SubSector": 197,
   "StartX": 3584,
   "StartY": -3904,
   "EndX": 3584,
   "EndY": -3840,
   "Result": 196,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 34,
   "SubSector": 79,
   "StartX": -256,
   "StartY": -3328,
   "EndX": -128,
   "EndY": -3328,
   "Result": 78,
   "Tag": ""
  },
  {
   "Sector": 34,
   "SubSector": 79,
   "StartX": -256,
   "StartY": -3328,
   "EndX": -128,
   "EndY": -3328,
   "Result": 78,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 34,
   "SubSector": 79,
   "StartX": -128,
   "StartY": -3344,
   "EndX": -256,
   "EndY": -3344,
   "Result": 77,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 24,
   "SubSector": 121,
   "StartX": 320,
   "StartY": -3200,
   "EndX": 288,
   "EndY": -3200,
   "Result": 122,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 121,
   "StartX": 288,
   "StartY": -3200,
   "EndX": 256,
   "EndY": -3200,
   "Result": 123,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 121,
   "StartX": 256,
   "StartY": -3136,
   "EndX": 320,
   "EndY": -3136,
   "Result": 129,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 53,
   "SubSector": 151,
   "StartX": 2624,
   "StartY": -2784,
   "EndX": 2752,
   "EndY": -2560,
   "Result": 152,
   "Tag": "--(twoSided)[BROWN1-BROWN1]"
  },
  {
   "Sector": 9,
   "SubSector": 19,
   "StartX": 1984,
   "StartY": -2240,
   "EndX": 1792,
   "EndY": -2240,
   "Result": 22,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 29,
   "SubSector": 71,
   "StartX": -320,
   "StartY": -3296,
   "EndX": -320,
   "EndY": -3168,
   "Result": 73,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 29,
   "SubSector": 71,
   "StartX": -256,
   "StartY": -3216,
   "EndX": -256,
   "EndY": -3248,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 24,
   "SubSector": 93,
   "StartX": 128,
   "StartY": -3200,
   "EndX": 64,
   "EndY": -3072,
   "Result": 95,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARG3-STARG3]"
  },
  {
   "Sector": 37,
   "SubSector": 96,
   "StartX": 928,
   "StartY": -3360,
   "EndX": 1184,
   "EndY": -3360,
   "Result": 105,
   "Tag": ""
  },
  {
   "Sector": 37,
   "SubSector": 96,
   "StartX": 1184,
   "StartY": -3392,
   "EndX": 928,
   "EndY": -3392,
   "Result": 97,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-COMPUTE2]"
  },
  {
   "Sector": 37,
   "SubSector": 96,
   "StartX": 928,
   "StartY": -3360,
   "EndX": 1184,
   "EndY": -3360,
   "Result": 105,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 16,
   "SubSector": 221,
   "StartX": 2240,
   "StartY": -3968,
   "EndX": 2240,
   "EndY": -4096,
   "Result": 220,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 62,
   "SubSector": 198,
   "StartX": 3616,
   "StartY": -3776,
   "EndX": 3584,
   "EndY": -3840,
   "Result": 198,
   "Tag": ""
  },
  {
   "Sector": 0,
   "SubSector": 0,
   "StartX": 1552,
   "StartY": -2560,
   "EndX": 1552,
   "EndY": -2432,
   "Result": 4,
   "Tag": "--(twoSided)[--BIGDOOR2]"
  },
  {
   "Sector": 0,
   "SubSector": 0,
   "StartX": 1552,
   "StartY": -2560,
   "EndX": 1552,
   "EndY": -2432,
   "Result": 4,
   "Tag": ""
  },
  {
   "Sector": 24,
   "SubSector": 125,
   "StartX": 320,
   "StartY": -3392,
   "EndX": 320,
   "EndY": -3328,
   "Result": 126,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 24,
   "SubSector": 127,
   "StartX": 320,
   "StartY": -3136,
   "EndX": 320,
   "EndY": -3072,
   "Result": 129,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 22,
   "SubSector": 137,
   "StartX": 2240,
   "StartY": -3648,
   "EndX": 2240,
   "EndY": -3776,
   "Result": 135,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 82,
   "SubSector": 232,
   "StartX": 3024,
   "StartY": -4848,
   "EndX": 3024,
   "EndY": -4840,
   "Result": 235,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 39,
   "SubSector": 101,
   "StartX": 1344,
   "StartY": -3360,
   "EndX": 1216,
   "EndY": -3392,
   "Result": 102,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 44,
   "SubSector": 126,
   "StartX": 320,
   "StartY": -3392,
   "EndX": 256,
   "EndY": -3392,
   "Result": 124,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 44,
   "SubSector": 126,
   "StartX": 320,
   "StartY": -3328,
   "EndX": 320,
   "EndY": -3392,
   "Result": 125,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 44,
   "SubSector": 126,
   "StartX": 256,
   "StartY": -3328,
   "EndX": 320,
   "EndY": -3328,
   "Result": 120,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 44,
   "SubSector": 126,
   "StartX": 256,
   "StartY": -3392,
   "EndX": 256,
   "EndY": -3328,
   "Result": 54,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 57,
   "SubSector": 159,
   "StartX": 3264,
   "StartY": -3616,
   "EndX": 3104,
   "EndY": -3552,
   "Result": 166,
   "Tag": "--(twoSided)[NUKE24--]"
  },
  {
   "Sector": 57,
   "SubSector": 159,
   "StartX": 3312,
   "StartY": -3496,
   "EndX": 3352,
   "EndY": -3568,
   "Result": 163,
   "Tag": "--(twoSided)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 53,
   "SubSector": 204,
   "StartX": 2944,
   "StartY": -2656,
   "EndX": 2752,
   "EndY": -2784,
   "Result": 201,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 7,
   "SubSector": 23,
   "StartX": 1664,
   "StartY": -2752,
   "EndX": 1664,
   "EndY": -2624,
   "Result": 6,
   "Tag": "--(twoSided,upperUnpegged)[COMPSPAN-COMPTALL]"
  },
  {
   "Sector": 37,
   "SubSector": 106,
   "StartX": 896,
   "StartY": -3360,
   "EndX": 896,
   "EndY": -3104,
   "Result": 107,
   "Tag": ""
  },
  {
   "Sector": 37,
   "SubSector": 106,
   "StartX": 928,
   "StartY": -3104,
   "EndX": 928,
   "EndY": -3360,
   "Result": 105,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 37,
   "SubSector": 106,
   "StartX": 896,
   "StartY": -3360,
   "EndX": 896,
   "EndY": -3104,
   "Result": 107,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-COMPUTE2]"
  },
  {
   "Sector": 5,
   "SubSector": 41,
   "StartX": 1896,
   "StartY": -3104,
   "EndX": 1672,
   "EndY": -3104,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 18,
   "SubSector": 46,
   "StartX": 2048,
   "StartY": -3840,
   "EndX": 2176,
   "EndY": -3840,
   "Result": 47,
   "Tag": ""
  },
  {
   "Sector": 18,
   "SubSector": 46,
   "StartX": 2048,
   "StartY": -3840,
   "EndX": 2176,
   "EndY": -3840,
   "Result": 47,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 18,
   "SubSector": 46,
   "StartX": 2176,
   "StartY": -3872,
   "EndX": 2048,
   "EndY": -3872,
   "Result": 45,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 33,
   "SubSector": 73,
   "StartX": -336,
   "StartY": -3296,
   "EndX": -336,
   "EndY": -3168,
   "Result": 72,
   "Tag": ""
  },
  {
   "Sector": 33,
   "SubSector": 73,
   "StartX": -320,
   "StartY": -3168,
   "EndX": -320,
   "EndY": -3296,
   "Result": 71,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 33,
   "SubSector": 73,
   "StartX": -336,
   "StartY": -3296,
   "EndX": -336,
   "EndY": -3168,
   "Result": 72,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 160,
   "StartY": -3264,
   "EndX": 128,
   "EndY": -3264,
   "Result": 90,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 128,
   "StartY": -3264,
   "EndX": 128,
   "EndY": -3200,
   "Result": 95,
   "Tag": ""
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 128,
   "StartY": -3200,
   "EndX": 160,
   "EndY": -3200,
   "Result": 91,
   "Tag": "--(twoSided,lowerUnpegged)[---]"
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 160,
   "StartY": -3200,
   "EndX": 160,
   "EndY": -3264,
   "Result": 58,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 160,
   "StartY": -3264,
   "EndX": 128,
   "EndY": -3264,
   "Result": 90,
   "Tag": ""
  },
  {
   "Sector": 36,
   "SubSector": 92,
   "StartX": 128,
   "StartY": -3264,
   "EndX": 128,
   "EndY": -3200,
   "Result": 95,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARG3-STARG3]"
  },
  {
   "Sector": 58,
   "SubSector": 161,
   "StartX": 3352,
   "StartY": -3592,
   "EndX": 3353,
   "EndY": -3601,
   "Result": 161,
   "Tag": ""
  },
  {
   "Sector": 57,
   "SubSector": 174,
   "StartX": 3136,
   "StartY": -3072,
   "EndX": 3304,
   "EndY": -3040,
   "Result": 177,
   "Tag": "--(twoSided,blockMonsters)[NUKE24--]"
  },
  {
   "Sector": 57,
   "SubSector": 174,
   "StartX": 3472,
   "StartY": -3432,
   "EndX": 3408,
   "EndY": -3432,
   "Result": 176,
   "Tag": "--(twoSided)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 57,
   "SubSector": 174,
   "StartX": 3304,
   "StartY": -3040,
   "EndX": 3400,
   "EndY": -3152,
   "Result": 175,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged,lowerUnpegged)[BROWNGRNBROWNGRNBROWNGRN]"
  },
  {
   "Sector": 22,
   "SubSector": 51,
   "StartX": 2176,
   "StartY": -3680,
   "EndX": 2048,
   "EndY": -3680,
   "Result": 53,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 22,
   "SubSector": 51,
   "StartX": 1984,
   "StartY": -3648,
   "EndX": 2176,
   "EndY": -3648,
   "Result": 38,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 56,
   "SubSector": 216,
   "StartX": 3072,
   "StartY": -4000,
   "EndX": 2944,
   "EndY": -4000,
   "Result": 217,
   "Tag": ""
  },
  {
   "Sector": 56,
   "SubSector": 216,
   "StartX": 3072,
   "StartY": -4000,
   "EndX": 2944,
   "EndY": -4000,
   "Result": 217,
   "Tag": "--(twoSided,upperUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 1520,
   "StartY": -3168,
   "EndX": 1672,
   "EndY": -3104,
   "Result": 37,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 1672,
   "StartY": -3104,
   "EndX": 1896,
   "EndY": -3104,
   "Result": 41,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 1896,
   "StartY": -3104,
   "EndX": 2040,
   "EndY": -3144,
   "Result": 31,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 2040,
   "StartY": -3144,
   "EndX": 2128,
   "EndY": -3272,
   "Result": 40,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 2128,
   "StartY": -3272,
   "EndX": 2064,
   "EndY": -3408,
   "Result": 36,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 2064,
   "StartY": -3408,
   "EndX": 1784,
   "EndY": -3448,
   "Result": 38,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 1784,
   "StartY": -3448,
   "EndX": 1544,
   "EndY": -3384,
   "Result": 35,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 13,
   "SubSector": 34,
   "StartX": 1544,
   "StartY": -3384,
   "EndX": 1520,
   "EndY": -3168,
   "Result": 39,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[BROWN144--]"
  },
  {
   "Sector": 14,
   "SubSector": 42,
   "StartX": 1344,
   "StartY": -3360,
   "EndX": 1344,
   "EndY": -3264,
   "Result": 105,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 14,
   "SubSector": 42,
   "StartX": 1376,
   "StartY": -3264,
   "EndX": 1376,
   "EndY": -3360,
   "Result": 39,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 14,
   "SubSector": 42,
   "StartX": 1376,
   "StartY": -3264,
   "EndX": 1376,
   "EndY": -3360,
   "Result": 39,
   "Tag": ""
  },
  {
   "Sector": 24,
   "SubSector": 54,
   "StartX": 160,
   "StartY": -3264,
   "EndX": 192,
   "EndY": -3264,
   "Result": 58,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 54,
   "StartX": 192,
   "StartY": -3264,
   "EndX": 224,
   "EndY": -3264,
   "Result": 57,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 54,
   "StartX": 224,
   "StartY": -3264,
   "EndX": 256,
   "EndY": -3264,
   "Result": 56,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 54,
   "StartX": 256,
   "StartY": -3328,
   "EndX": 256,
   "EndY": -3392,
   "Result": 126,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 5,
   "SubSector": 40,
   "StartX": 2128,
   "StartY": -3272,
   "EndX": 2040,
   "EndY": -3144,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 40,
   "StartX": 2040,
   "StartY": -3144,
   "EndX": 2128,
   "EndY": -3272,
   "Result": 40,
   "Tag": ""
  },
  {
   "Sector": 29,
   "SubSector": 69,
   "StartX": -256,
   "StartY": -3248,
   "EndX": -240,
   "EndY": -3264,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 69,
   "StartX": -240,
   "StartY": -3264,
   "EndX": -256,
   "EndY": -3248,
   "Result": 69,
   "Tag": ""
  },
  {
   "Sector": 60,
   "SubSector": 166,
   "StartX": 3104,
   "StartY": -3552,
   "EndX": 3264,
   "EndY": -3616,
   "Result": 159,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 62,
   "SubSector": 179,
   "StartX": 3320,
   "StartY": -3040,
   "EndX": 3304,
   "EndY": -3040,
   "Result": 179,
   "Tag": ""
  },
  {
   "Sector": 2,
   "SubSector": 7,
   "StartX": 1344,
   "StartY": -2880,
   "EndX": 1384,
   "EndY": -2592,
   "Result": 7,
   "Tag": ""
  },
  {
   "Sector": 39,
   "SubSector": 111,
   "StartX": 1216,
   "StartY": -3072,
   "EndX": 1344,
   "EndY": -3104,
   "Result": 114,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 64,
   "SubSector": 187,
   "StartX": 2816,
   "StartY": -3776,
   "EndX": 2816,
   "EndY": -3904,
   "Result": 188,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 64,
   "SubSector": 187,
   "StartX": 2784,
   "StartY": -3904,
   "EndX": 2784,
   "EndY": -3776,
   "Result": 186,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 64,
   "SubSector": 187,
   "StartX": 2816,
   "StartY": -3776,
   "EndX": 2816,
   "EndY": -3904,
   "Result": 188,
   "Tag": ""
  },
  {
   "Sector": 78,
   "SubSector": 225,
   "StartX": 3024,
   "StartY": -4600,
   "EndX": 3024,
   "EndY": -4592,
   "Result": 228,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 11,
   "SubSector": 25,
   "StartX": 1984,
   "StartY": -2624,
   "EndX": 1984,
   "EndY": -2560,
   "Result": 29,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 24,
   "SubSector": 91,
   "StartX": 160,
   "StartY": -3200,
   "EndX": 128,
   "EndY": -3200,
   "Result": 92,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 38,
   "SubSector": 113,
   "StartX": 896,
   "StartY": -3072,
   "EndX": 928,
   "EndY": -3072,
   "Result": 113,
   "Tag": ""
  },
  {
   "Sector": 17,
   "SubSector": 45,
   "StartX": 2048,
   "StartY": -3872,
   "EndX": 2176,
   "EndY": -3872,
   "Result": 46,
   "Tag": ""
  },
  {
   "Sector": 17,
   "SubSector": 45,
   "StartX": 2048,
   "StartY": -3872,
   "EndX": 2176,
   "EndY": -3872,
   "Result": 46,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 17,
   "SubSector": 45,
   "StartX": 2176,
   "StartY": -3904,
   "EndX": 2048,
   "EndY": -3904,
   "Result": 44,
   "Tag": "--(twoSided,upperUnpegged)[--BROWN1]"
  },
  {
   "Sector": 70,
   "SubSector": 196,
   "StartX": 3520,
   "StartY": -3904,
   "EndX": 3520,
   "EndY": -3840,
   "Result": 193,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged)[--BROWNGRN]"
  },
  {
   "Sector": 70,
   "SubSector": 196,
   "StartX": 3584,
   "StartY": -3840,
   "EndX": 3584,
   "EndY": -3904,
   "Result": 197,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged,lowerUnpegged)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 70,
   "SubSector": 196,
   "StartX": 3584,
   "StartY": -3840,
   "EndX": 3584,
   "EndY": -3904,
   "Result": 197,
   "Tag": ""
  },
  {
   "Sector": 29,
   "SubSector": 78,
   "StartX": -128,
   "StartY": -3328,
   "EndX": -256,
   "EndY": -3328,
   "Result": 79,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 29,
   "SubSector": 78,
   "StartX": -256,
   "StartY": -3328,
   "EndX": -128,
   "EndY": -3328,
   "Result": 78,
   "Tag": ""
  },
  {
   "Sector": 40,
   "SubSector": 104,
   "StartX": 1024,
   "StartY": -3648,
   "EndX": 1088,
   "EndY": -3648,
   "Result": 103,
   "Tag": ""
  },
  {
   "Sector": 40,
   "SubSector": 104,
   "StartX": 1024,
   "StartY": -3648,
   "EndX": 1088,
   "EndY": -3648,
   "Result": 103,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 56,
   "SubSector": 158,
   "StartX": 3072,
   "StartY": -3648,
   "EndX": 2944,
   "EndY": -3536,
   "Result": 158,
   "Tag": ""
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 2992,
   "StartY": -4840,
   "EndX": 3024,
   "EndY": -4840,
   "Result": 231,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 3024,
   "StartY": -4840,
   "EndX": 3024,
   "EndY": -4848,
   "Result": 232,
   "Tag": ""
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 3024,
   "StartY": -4848,
   "EndX": 2992,
   "EndY": -4848,
   "Result": 233,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 3024,
   "StartY": -4840,
   "EndX": 3024,
   "EndY": -4848,
   "Result": 232,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 3024,
   "StartY": -4848,
   "EndX": 2992,
   "EndY": -4848,
   "Result": 233,
   "Tag": ""
  },
  {
   "Sector": 83,
   "SubSector": 235,
   "StartX": 2992,
   "StartY": -4848,
   "EndX": 2992,
   "EndY": -4840,
   "Result": 234,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 59,
   "SubSector": 176,
   "StartX": 3408,
   "StartY": -3432,
   "EndX": 3472,
   "EndY": -3432,
   "Result": 174,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 36,
   "StartX": 2064,
   "StartY": -3408,
   "EndX": 2128,
   "EndY": -3272,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 36,
   "StartX": 2128,
   "StartY": -3272,
   "EndX": 2064,
   "EndY": -3408,
   "Result": 36,
   "Tag": ""
  },
  {
   "Sector": 35,
   "SubSector": 95,
   "StartX": 128,
   "StartY": -3200,
   "EndX": 128,
   "EndY": -3264,
   "Result": 92,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 35,
   "SubSector": 95,
   "StartX": 64,
   "StartY": -3072,
   "EndX": 128,
   "EndY": -3200,
   "Result": 93,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 35,
   "SubSector": 95,
   "StartX": 128,
   "StartY": -3200,
   "EndX": 128,
   "EndY": -3264,
   "Result": 92,
   "Tag": ""
  },
  {
   "Sector": 35,
   "SubSector": 95,
   "StartX": 128,
   "StartY": -3264,
   "EndX": 64,
   "EndY": -3392,
   "Result": 94,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 29,
   "SubSector": 61,
   "StartX": -256,
   "StartY": -3136,
   "EndX": -128,
   "EndY": -3136,
   "Result": 63,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 29,
   "SubSector": 61,
   "StartX": -192,
   "StartY": -3216,
   "EndX": -208,
   "EndY": -3200,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 38,
   "SubSector": 103,
   "StartX": 1088,
   "StartY": -3648,
   "EndX": 1024,
   "EndY": -3648,
   "Result": 104,
   "Tag": ""
  },
  {
   "Sector": 38,
   "SubSector": 103,
   "StartX": 1088,
   "StartY": -3648,
   "EndX": 1024,
   "EndY": -3648,
   "Result": 104,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 5,
   "SubSector": 136,
   "StartX": 2176,
   "StartY": -3648,
   "EndX": 2240,
   "EndY": -3648,
   "Result": 136,
   "Tag": ""
  },
  {
   "Sector": 56,
   "SubSector": 157,
   "StartX": 2944,
   "StartY": -3648,
   "EndX": 2944,
   "EndY": -3669,
   "Result": 157,
   "Tag": ""
  },
  {
   "Sector": 60,
   "SubSector": 172,
   "StartX": 3136,
   "StartY": -3072,
   "EndX": 2984,
   "EndY": -3200,
   "Result": 171,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 68,
   "SubSector": 192,
   "StartX": 2912,
   "StartY": -3904,
   "EndX": 2912,
   "EndY": -3776,
   "Result": 191,
   "Tag": ""
  },
  {
   "Sector": 68,
   "SubSector": 192,
   "StartX": 2944,
   "StartY": -3776,
   "EndX": 2944,
   "EndY": -3904,
   "Result": 190,
   "Tag": "--(twoSided,secret)[---]"
  },
  {
   "Sector": 68,
   "SubSector": 192,
   "StartX": 2912,
   "StartY": -3904,
   "EndX": 2912,
   "EndY": -3776,
   "Result": 191,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 11,
   "SubSector": 27,
   "StartX": 1792,
   "StartY": -2560,
   "EndX": 1792,
   "EndY": -2624,
   "Result": 29,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 16,
   "SubSector": 44,
   "StartX": 2048,
   "StartY": -3904,
   "EndX": 2176,
   "EndY": -3904,
   "Result": 45,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 54,
   "SubSector": 152,
   "StartX": 2752,
   "StartY": -2560,
   "EndX": 2624,
   "EndY": -2784,
   "Result": 151,
   "Tag": ""
  },
  {
   "Sector": 54,
   "SubSector": 152,
   "StartX": 2752,
   "StartY": -2560,
   "EndX": 2624,
   "EndY": -2784,
   "Result": 151,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 57,
   "SubSector": 162,
   "StartX": 3408,
   "StartY": -3432,
   "EndX": 3312,
   "EndY": -3496,
   "Result": 163,
   "Tag": "--(twoSided)[BROWNGRN-BROWNGRN]"
  },
  {
   "Sector": 81,
   "SubSector": 230,
   "StartX": 3040,
   "StartY": -4648,
   "EndX": 2976,
   "EndY": -4648,
   "Result": 236,
   "Tag": ""
  },
  {
   "Sector": 81,
   "SubSector": 230,
   "StartX": 3040,
   "StartY": -4648,
   "EndX": 2976,
   "EndY": -4648,
   "Result": 236,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 81,
   "SubSector": 230,
   "StartX": 2976,
   "StartY": -4632,
   "EndX": 3040,
   "EndY": -4632,
   "Result": 229,
   "Tag": "--(twoSided)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 100,
   "StartX": 1216,
   "StartY": -3392,
   "EndX": 1184,
   "EndY": -3392,
   "Result": 100,
   "Tag": ""
  },
  {
   "Sector": 24,
   "SubSector": 119,
   "StartX": 496,
   "StartY": -3160,
   "EndX": 496,
   "EndY": -3304,
   "Result": 118,
   "Tag": "--(twoSided)[--STARG3]"
  },
  {
   "Sector": 24,
   "SubSector": 119,
   "StartX": 320,
   "StartY": -3264,
   "EndX": 320,
   "EndY": -3200,
   "Result": 122,
   "Tag": "--(twoSided)[STEP6-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 124,
   "StartX": 256,
   "StartY": -3392,
   "EndX": 320,
   "EndY": -3392,
   "Result": 126,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 48,
   "SubSector": 139,
   "StartX": 2632,
   "StartY": -3792,
   "EndX": 2688,
   "EndY": -3920,
   "Result": 142,
   "Tag": "--(twoSided,blockMonsters,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 74,
   "SubSector": 208,
   "StartX": 2856,
   "StartY": -4160,
   "EndX": 2856,
   "EndY": -4352,
   "Result": 211,
   "Tag": ""
  },
  {
   "Sector": 74,
   "SubSector": 208,
   "StartX": 2856,
   "StartY": -4160,
   "EndX": 2856,
   "EndY": -4352,
   "Result": 211,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 38,
   "SubSector": 115,
   "StartX": 832,
   "StartY": -2944,
   "EndX": 704,
   "EndY": -2944,
   "Result": 115,
   "Tag": ""
  },
  {
   "Sector": 24,
   "SubSector": 128,
   "StartX": 320,
   "StartY": -3072,
   "EndX": 256,
   "EndY": -3072,
   "Result": 129,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 56,
   "SubSector": 156,
   "StartX": 2752,
   "StartY": -3584,
   "EndX": 2944,
   "EndY": -3536,
   "Result": 155,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 67,
   "SubSector": 191,
   "StartX": 2880,
   "StartY": -3904,
   "EndX": 2880,
   "EndY": -3776,
   "Result": 189,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 67,
   "SubSector": 191,
   "StartX": 2912,
   "StartY": -3776,
   "EndX": 2912,
   "EndY": -3904,
   "Result": 192,
   "Tag": ""
  },
  {
   "Sector": 67,
   "SubSector": 191,
   "StartX": 2912,
   "StartY": -3776,
   "EndX": 2912,
   "EndY": -3904,
   "Result": 192,
   "Tag": "--(twoSided)[BROWN96-BROWN96]"
  },
  {
   "Sector": 9,
   "SubSector": 18,
   "StartX": 1984,
   "StartY": -2304,
   "EndX": 1984,
   "EndY": -2240,
   "Result": 22,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 29,
   "SubSector": 65,
   "StartX": -208,
   "StartY": -3264,
   "EndX": -192,
   "EndY": -3248,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 65,
   "StartX": -192,
   "StartY": -3248,
   "EndX": -208,
   "EndY": -3264,
   "Result": 65,
   "Tag": ""
  },
  {
   "Sector": 41,
   "SubSector": 117,
   "StartX": 704,
   "StartY": -3104,
   "EndX": 704,
   "EndY": -3360,
   "Result": 107,
   "Tag": "--(twoSided,upperUnpegged)[STEP6-STARTAN3]"
  },
  {
   "Sector": 41,
   "SubSector": 117,
   "StartX": 704,
   "StartY": -3104,
   "EndX": 704,
   "EndY": -3360,
   "Result": 107,
   "Tag": ""
  },
  {
   "Sector": 58,
   "SubSector": 160,
   "StartX": 3352,
   "StartY": -3568,
   "EndX": 3352,
   "EndY": -3592,
   "Result": 160,
   "Tag": ""
  },
  {
   "Sector": 19,
   "SubSector": 47,
   "StartX": 2176,
   "StartY": -3840,
   "EndX": 2048,
   "EndY": -3840,
   "Result": 46,
   "Tag": ""
  },
  {
   "Sector": 19,
   "SubSector": 47,
   "StartX": 2048,
   "StartY": -3808,
   "EndX": 2176,
   "EndY": -3808,
   "Result": 50,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 19,
   "SubSector": 47,
   "StartX": 2176,
   "StartY": -3840,
   "EndX": 2048,
   "EndY": -3840,
   "Result": 46,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 60,
   "SubSector": 170,
   "StartX": 2944,
   "StartY": -3536,
   "EndX": 3112,
   "EndY": -3360,
   "Result": 169,
   "Tag": "--(twoSided,blockMonsters)[---]"
  },
  {
   "Sector": 51,
   "SubSector": 146,
   "StartX": 2208,
   "StartY": -2304,
   "EndX": 2208,
   "EndY": -2560,
   "Result": 147,
   "Tag": ""
  },
  {
   "Sector": 51,
   "SubSector": 146,
   "StartX": 2176,
   "StartY": -2560,
   "EndX": 2176,
   "EndY": -2304,
   "Result": 10,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 51,
   "SubSector": 146,
   "StartX": 2208,
   "StartY": -2304,
   "EndX": 2208,
   "EndY": -2560,
   "Result": 147,
   "Tag": "--(twoSided,upperUnpegged)[STEP1--]"
  },
  {
   "Sector": 78,
   "SubSector": 227,
   "StartX": 2992,
   "StartY": -4592,
   "EndX": 2992,
   "EndY": -4600,
   "Result": 228,
   "Tag": "--(twoSided)[--EXITSIGN]"
  },
  {
   "Sector": 66,
   "SubSector": 189,
   "StartX": 2880,
   "StartY": -3776,
   "EndX": 2880,
   "EndY": -3904,
   "Result": 191,
   "Tag": "--(twoSided,upperUnpegged)[BROWN1-BROWN1]"
  },
  {
   "Sector": 66,
   "SubSector": 189,
   "StartX": 2848,
   "StartY": -3904,
   "EndX": 2848,
   "EndY": -3776,
   "Result": 188,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 66,
   "SubSector": 189,
   "StartX": 2880,
   "StartY": -3776,
   "EndX": 2880,
   "EndY": -3904,
   "Result": 191,
   "Tag": ""
  },
  {
   "Sector": 3,
   "SubSector": 3,
   "StartX": 1472,
   "StartY": -2560,
   "EndX": 1472,
   "EndY": -2432,
   "Result": 2,
   "Tag": "--(twoSided,upperUnpegged)[---]"
  },
  {
   "Sector": 3,
   "SubSector": 3,
   "StartX": 1536,
   "StartY": -2432,
   "EndX": 1536,
   "EndY": -2560,
   "Result": 4,
   "Tag": "--(twoSided)[--BIGDOOR2]"
  },
  {
   "Sector": 3,
   "SubSector": 3,
   "StartX": 1472,
   "StartY": -2560,
   "EndX": 1472,
   "EndY": -2432,
   "Result": 2,
   "Tag": ""
  },
  {
   "Sector": 11,
   "SubSector": 28,
   "StartX": 1792,
   "StartY": -2624,
   "EndX": 1984,
   "EndY": -2624,
   "Result": 29,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARGR1-PLANET1]"
  },
  {
   "Sector": 5,
   "SubSector": 38,
   "StartX": 1784,
   "StartY": -3448,
   "EndX": 2064,
   "EndY": -3408,
   "Result": 34,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[---]"
  },
  {
   "Sector": 5,
   "SubSector": 38,
   "StartX": 2176,
   "StartY": -3648,
   "EndX": 1984,
   "EndY": -3648,
   "Result": 51,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 55,
   "StartX": 256,
   "StartY": -3200,
   "EndX": 224,
   "EndY": -3200,
   "Result": 56,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 55,
   "StartX": 224,
   "StartY": -3200,
   "EndX": 192,
   "EndY": -3200,
   "Result": 57,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 55,
   "StartX": 192,
   "StartY": -3200,
   "EndX": 160,
   "EndY": -3200,
   "Result": 58,
   "Tag": "--(twoSided,lowerUnpegged)[SLADWALL-STARTAN3]"
  },
  {
   "Sector": 24,
   "SubSector": 55,
   "StartX": 256,
   "StartY": -3072,
   "EndX": 256,
   "EndY": -3136,
   "Result": 129,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL4-TEKWALL4]"
  },
  {
   "Sector": 30,
   "SubSector": 77,
   "StartX": -256,
   "StartY": -3344,
   "EndX": -128,
   "EndY": -3344,
   "Result": 79,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[STARTAN3-STARTAN3]"
  },
  {
   "Sector": 29,
   "SubSector": 68,
   "StartX": -240,
   "StartY": -3200,
   "EndX": -256,
   "EndY": -3216,
   "Result": 70,
   "Tag": "--(twoSided,upperUnpegged,lowerUnpegged)[TEKWALL1-TEKWALL1]"
  },
  {
   "Sector": 29,
   "SubSector": 68,
   "StartX": -256,
   "StartY": -3216,
   "EndX": -240,
   "EndY": -3200,
   "Result": 68,
   "Tag": ""
  },
  {
   "Sector": 46,
   "SubSector": 133,
   "StartX": 2736,
   "StartY": -3360,
   "EndX": 2736,
   "EndY": -3112,
   "Result": 131,
   "Tag": ""
  },
  {
   "Sector": 46,
   "SubSector": 133,
   "StartX": 2736,
   "StartY": -3360,
   "EndX": 2736,
   "EndY": -3112,
   "Result": 131,
   "Tag": "--(twoSided,impassible,upperUnpegged,lowerUnpegged)[---]"
  }
 ]
}
`