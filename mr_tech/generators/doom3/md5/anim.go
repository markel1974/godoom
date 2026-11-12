package md5

import (
	"bufio"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/physics/aabb"
)

// AnimHierarchyJoint represents a bone in the hierarchy section of an .md5anim file.
type AnimHierarchyJoint struct {
	name       string
	parent     int
	flags      int
	startIndex int
}

// AnimBaseFrameJoint represents a joint's default local position and orientation in an .md5anim file.
type AnimBaseFrameJoint struct {
	pos    geometry.XYZ
	orient [4]float64
}

// Anim holds the parsed animation data of an .md5anim file.
type Anim struct {
	numFrames int
	frameRate int
	hierarchy []AnimHierarchyJoint
	baseFrame []AnimBaseFrameJoint
	Frames    [][]float64
}

// NewAnim initializes and returns a new Anim object for parsing and managing .md5anim animation data.
func NewAnim() *Anim {
	return &Anim{}
}

// NumJoints returns the number of joints in the animation's hierarchy.
func (anim *Anim) NumJoints() int {
	return len(anim.hierarchy)
}

// Parse parses an .md5anim file.
func (anim *Anim) Parse(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	inHierarchy := false
	inBaseFrame := false
	inFrame := false
	currentFrameIdx := -1
	var currentFrameVals []float64

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "//") {
			continue
		}

		if commentIdx := strings.Index(line, "//"); commentIdx >= 0 {
			line = strings.TrimSpace(line[:commentIdx])
		}

		tokens := strings.Fields(line)
		if len(tokens) == 0 {
			continue
		}

		if tokens[0] == "numFrames" && len(tokens) > 1 {
			anim.numFrames, _ = strconv.Atoi(tokens[1])
			continue
		}
		if tokens[0] == "frameRate" && len(tokens) > 1 {
			anim.frameRate, _ = strconv.Atoi(tokens[1])
			continue
		}
		if tokens[0] == "hierarchy" && len(tokens) > 1 && tokens[1] == "{" {
			inHierarchy = true
			continue
		}
		if tokens[0] == "baseframe" && len(tokens) > 1 && tokens[1] == "{" {
			inBaseFrame = true
			continue
		}
		if tokens[0] == "frame" && len(tokens) > 2 && tokens[2] == "{" {
			inFrame = true
			currentFrameIdx, _ = strconv.Atoi(tokens[1])
			currentFrameVals = make([]float64, 0, 64)
			_ = currentFrameIdx
			continue
		}

		if tokens[0] == "}" {
			if inHierarchy {
				inHierarchy = false
			} else if inBaseFrame {
				inBaseFrame = false
			} else if inFrame {
				anim.Frames = append(anim.Frames, currentFrameVals)
				inFrame = false
			}
			continue
		}

		if inHierarchy {
			quoteStart := strings.Index(line, "\"")
			quoteEnd := strings.LastIndex(line, "\"")
			if quoteStart < 0 || quoteEnd <= quoteStart {
				continue
			}
			name := line[quoteStart+1 : quoteEnd]
			rest := strings.TrimSpace(line[quoteEnd+1:])
			subTokens := strings.Fields(rest)
			if len(subTokens) >= 3 {
				parent, _ := strconv.Atoi(subTokens[0])
				flags, _ := strconv.Atoi(subTokens[1])
				sIdx, _ := strconv.Atoi(subTokens[2])
				anim.hierarchy = append(anim.hierarchy, AnimHierarchyJoint{
					name:       name,
					parent:     parent,
					flags:      flags,
					startIndex: sIdx,
				})
			}
			continue
		}

		if inBaseFrame {
			pStart := strings.Index(line, "(")
			pEnd := strings.Index(line, ")")
			qStart := strings.LastIndex(line, "(")
			qEnd := strings.LastIndex(line, ")")
			if pStart >= 0 && pEnd > pStart && qStart >= 0 && qEnd > qStart && qStart > pEnd {
				pFields := strings.Fields(line[pStart+1 : pEnd])
				qFields := strings.Fields(line[qStart+1 : qEnd])
				if len(pFields) >= 3 && len(qFields) >= 3 {
					px, _ := strconv.ParseFloat(pFields[0], 64)
					py, _ := strconv.ParseFloat(pFields[1], 64)
					pz, _ := strconv.ParseFloat(pFields[2], 64)

					qx, _ := strconv.ParseFloat(qFields[0], 64)
					qy, _ := strconv.ParseFloat(qFields[1], 64)
					qz, _ := strconv.ParseFloat(qFields[2], 64)

					anim.baseFrame = append(anim.baseFrame, AnimBaseFrameJoint{
						pos:    geometry.XYZ{X: px, Y: py, Z: pz},
						orient: [4]float64{qx, qy, qz, 0},
					})
				}
			}
			continue
		}

		if inFrame {
			for _, tStr := range tokens {
				val, err := strconv.ParseFloat(tStr, 64)
				if err == nil {
					currentFrameVals = append(currentFrameVals, val)
				}
			}
			continue
		}
	}

	return nil
}

// BuildFrameSkeleton constructs the animated skeleton joints in model space for a specific frame index.
func (anim *Anim) BuildFrameSkeleton(frameIdx int) []*Joint {
	skel := make([]*Joint, len(anim.hierarchy))
	if frameIdx < 0 || frameIdx >= len(anim.Frames) {
		return skel
	}
	frameData := anim.Frames[frameIdx]

	for i, h := range anim.hierarchy {
		base := anim.baseFrame[i]
		pos := base.pos
		orient := base.orient
		k := 0

		if h.flags&1 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.X = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&2 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.Y = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&4 != 0 {
			if h.startIndex+k < len(frameData) {
				pos.Z = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&8 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[0] = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&16 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[1] = frameData[h.startIndex+k]
				k++
			}
		}
		if h.flags&32 != 0 {
			if h.startIndex+k < len(frameData) {
				orient[2] = frameData[h.startIndex+k]
				k++
			}
		}

		// Recompute qw with strict negative convention for Doom 3
		t := 1.0 - (orient[0]*orient[0] + orient[1]*orient[1] + orient[2]*orient[2])
		if t > 0.0 {
			orient[3] = -math.Sqrt(t)
		} else {
			orient[3] = 0.0
		}

		if h.parent < 0 {
			// Strip root motion translation so the animation plays in-place while physics drives world movement.
			pos = geometry.XYZ{}
			skel[i] = NewJoint(h.name, h.parent, pos, aabb.NewQuaternion(orient))
		} else {
			parentJoint := skel[h.parent]
			rPosX, rPosY, rPosZ := parentJoint.Orient.Rotate(pos.X, pos.Y, pos.Z)
			pk := geometry.XYZ{X: parentJoint.Pos.X + rPosX, Y: parentJoint.Pos.Y + rPosY, Z: parentJoint.Pos.Z + rPosZ}

			orientQuat := aabb.NewQuaternion(orient)
			parentJoint.Orient.Multiplies(orientQuat, orientQuat)
			orientQuat.Normalize()

			skel[i] = NewJoint(h.name, h.parent, pk, orientQuat)
		}
	}

	return skel
}
