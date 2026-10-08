package model

/*
import (
	"time"
	"github.com/markel1974/godoom/mr_tech/config"
)

// ThingEffect represents a temporary visual effect like an explosion or a flash.
// It uses the generic IThing system but manages its own lifecycle.
type ThingEffect struct {
	*ThingBase
	spawnTime time.Time
	duration  time.Duration
}

// NewThingEffect creates a new ThingEffect instance.
func NewThingEffect(things *Things, cfg *config.Thing, volume *Volume) *ThingEffect {
	thing := &ThingEffect{
		spawnTime: time.Now(),
		duration:  0,
	}
	thing.ThingBase = NewThingBase(thing, things, cfg, volume)

	// Set properties from config if available
	if cfg.Effect != nil {
		thing.duration = time.Duration(cfg.Effect.Duration * float64(time.Millisecond))
		// Optional: Apply StartScale if needed, or modify Light Intensity
	} else {
		// Fallback duration if not defined
		thing.duration = 500 * time.Millisecond
	}

	// Un effect è intangibile (nessuna collisione solida)
	thing.GetEntity().SetMass(0)

	return thing
}

// PostMessage sends an ThingEvent instance to the ThingEffect's inbox channel for processing.
func (t *ThingEffect) PostMessage(ec *ThingEvent) {
	t.inbox <- ec
}

// StartLoop begins a goroutine that processes incoming events.
func (t *ThingEffect) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-t.inbox:
				switch evt.GetKind() {
				case StageThinking:
					t.StageThinking(evt.GetCoords())
				case StageCompute:
				case StageResolve:
				case StageApply:
				}
				evt.Done()
			case <-t.done:
				return
			}
		}
	}()
}

// StageThinking checks if the effect has outlived its duration and marks it for removal.
func (t *ThingEffect) StageThinking(playerX float64, playerY float64, playerZ float64) {
	if time.Since(t.spawnTime) > t.duration {
		t.SetActive(false) // Segna l'entità per essere rimossa dal mondo
	}
}

*/
