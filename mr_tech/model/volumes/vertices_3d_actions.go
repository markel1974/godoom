package volumes

import (
	"fmt"

	"github.com/markel1974/godoom/mr_tech/config"
)

// Vertices3DActions stores per-part mappings from composite model actions to each part's animation actions.
type Vertices3DActions struct {
	actions [][]int
}

// NewVertices3DActions creates an empty Vertices3DActions instance.
func NewVertices3DActions() *Vertices3DActions {
	return &Vertices3DActions{}
}

// Setup initializes per-part action maps from the model config, validating entries and defaulting invalid mappings to 0.
func (v *Vertices3DActions) Setup(cfg *config.Model3D, elements []*Vertices3DEntry) error {
	numParts := len(elements)
	v.actions = make([][]int, numParts)

	// Validate and normalize ActionMaps
	rawActions := cfg.ActionMaps
	if rawActions == nil || len(rawActions) == 0 {
		fmt.Printf("[warning] Vertices3D: Model3D has no ActionMaps defined\n")
	} else if len(rawActions) != numParts {
		fmt.Printf("[warning] Vertices3D: ActionMaps count (%d) does not match parts count (%d)\n", len(rawActions), numParts)
	}

	expectedActions := 0
	if len(cfg.ActionDefinitions) > 0 {
		expectedActions = len(cfg.ActionDefinitions)
	} else if len(rawActions) > 0 && len(rawActions[0]) > 0 {
		expectedActions = len(rawActions[0])
	} else if len(elements) > 0 && elements[0] != nil {
		expectedActions = len(elements[0].actions)
	}

	for i := 0; i < numParts; i++ {
		var action []int
		if i < len(rawActions) && rawActions[i] != nil {
			action = make([]int, len(rawActions[i]))
			copy(action, rawActions[i])
			if expectedActions > 0 && len(action) < expectedActions {
				fmt.Printf("[warning] Vertices3D: part %d action map has %d entries, expected at least %d, falling back to action 0 for missing entries\n", i, len(action), expectedActions)
				padded := make([]int, expectedActions)
				copy(padded, action)
				action = padded
			}
		} else {
			fmt.Printf("[warning] Vertices3D missing action map for part %d, falling back to action 0\n", i)
			action = make([]int, expectedActions)
		}

		if elements[i] != nil {
			numActions := len(elements[i].actions)
			for actionIdx, targetIdx := range action {
				if targetIdx < 0 || (numActions > 0 && targetIdx >= numActions) {
					fmt.Printf("[warning] Vertices3D: part %d action %d maps to invalid sub-action %d (max %d), falling back to action 0\n", i, actionIdx, targetIdx, numActions-1)
					action[actionIdx] = 0
				}
			}
		}
		v.actions[i] = action
	}
	return nil
}
