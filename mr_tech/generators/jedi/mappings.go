package jedi

import (
	"fmt"
	"strings"

	"github.com/markel1974/godoom/mr_tech/config"
)

// _aliases maps specific string keys to their corresponding alias strings used for standardizing naming conventions.
var _aliases = map[string]string{
	"GAMOBXS":  "GAMBOXS",
	"GAMOBXB":  "GAMBOXB",
	"GAMBOXP":  "GAMBOXB",
	"GBAR":     "GGBAR",
	"GSAK":     "GGSAK",
	"HORSE2":   "SHORSE2",
	"LANTERN1": "LANTERN",
	"NAMBOXT":  "NAMBOXT1",
}

// GetAlias checks if the given name has a corresponding alias in the alias map and returns it; otherwise, it returns the name.
func GetAlias(name string) string {
	if v, ok := _aliases[name]; ok {
		return v
	}
	return name
}

// ConvertAnimation maps animations to enemy states and adds them to EnemyAnimations with their respective intervals and durations.
func ConvertAnimation(r *config.EnemyAnimations, animations []string, intervals [][2]int) {
	for idx, animation := range animations {
		animation = strings.TrimSpace(strings.ToLower(animation))
		if strings.HasPrefix(animation, "stand") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "hover") {
			r.Add(config.EnemyStateIdle, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "walk") {
			r.Add(config.EnemyStateWalk, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "run") {
			r.Add(config.EnemyStateRun, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "prowl") {
			r.Add(config.EnemyStateProwl, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "leap") {
			r.Add(config.EnemyStateJump, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "attack") {
			r.Add(config.EnemyStateMelee, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "shoot") {
			r.Add(config.EnemyStateShoot, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "smash") {
			r.Add(config.EnemyStateShoot, idx, intervals[idx][1]-intervals[idx][0]) // TODO IMPLEMENT
		} else if strings.HasPrefix(animation, "load") {
			r.Add(config.EnemyStateReload, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "pain") {
			r.Add(config.EnemyStatePain, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "death") {
			r.Add(config.EnemyStateDead, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "bdeath") {
			r.Add(config.EnemyStateDead, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "pull") {
			r.Add(config.EnemyStatePull, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "swing") {
			r.Add(config.EnemyStateSwing, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "fly") {
			r.Add(config.EnemyStateFly, idx, intervals[idx][1]-intervals[idx][0])
		} else if strings.HasPrefix(animation, "kneel") {
			r.Add(config.EnemyStateKneel, idx, intervals[idx][1]-intervals[idx][0])
		} else {
			fmt.Printf("Unknown animation: %s\n", animation)
		}
	}
}
