package things

import (
	"strings"
	"sync/atomic"

	"github.com/markel1974/godoom/mr_tech/config"
	"github.com/markel1974/godoom/mr_tech/geometry"
	"github.com/markel1974/godoom/mr_tech/model/ithing"
	"github.com/markel1974/godoom/mr_tech/model/volumes"
	"github.com/markel1974/godoom/mr_tech/physics/collisions"
)

// IKeyHolder is implemented by actors (such as Player) that can collect and carry keys.
type IKeyHolder interface {
	GiveKey(keyId string)
	HasKey(keyId string) bool
}

// Key is a collectible key entity that grants a key ID on collision with an IKeyHolder and removes itself.
type Key struct {
	*Base
	keyId     string
	collected atomic.Bool
}

// NewKey creates and initializes a Key Thing with collision-based pickup behavior.
func NewKey(things IThingHandler, cfg *config.Thing, volume *volumes.Volume) *Key {
	keyId := cfg.Id
	if cfg.Key != nil && strings.TrimSpace(cfg.Key.KeyId) != "" {
		keyId = cfg.Key.KeyId
	}
	key := &Key{
		Base:  NewBase(things, cfg),
		keyId: strings.ToLower(strings.TrimSpace(keyId)),
	}
	if err := key.Prepare(key, volume); err != nil {
		panic(err)
	}
	key.GetSolver().SetElasticCollision(key.onKeyCollision)
	cfg.OnBind(key)
	return key
}

// IsEqual reports whether other is the same Key instance.
func (k *Key) IsEqual(other ithing.IThing) bool {
	if v, ok := other.(*Key); ok && k == v {
		return true
	}
	return false
}

// KeyId returns the normalized key identifier carried by this Key.
func (k *Key) KeyId() string {
	return k.keyId
}

// Jump is unsupported for static pickup keys.
func (k *Key) Jump(leapX float64, leapY float64, zFactor float64) bool {
	return false
}

// FireHitscan is unsupported for pickup keys.
func (k *Key) FireHitscan(id string, pos geometry.XYZ, force float64, dirX, dirY, dirZ float64) {
}

// LaunchObject is unsupported for pickup keys.
func (k *Key) LaunchObject(throwableIndex int, onCollision config.CollisionFunc, onImpact config.ImpactFunc, pos geometry.XYZ, angle, pitch, speed float64) {
}

// PostMessage sends a ThingEvent to the Key's inbox channel.
func (k *Key) PostMessage(ec *ithing.ThingEvent) {
	k.inbox <- ec
}

// StartLoop starts the goroutine processing physics stages for the Key.
func (k *Key) StartLoop() {
	go func() {
		for {
			select {
			case evt := <-k.inbox:
				switch evt.GetKind() {
				case ithing.StageThinking:
					k.StageThinking(evt.GetCoords())
				case ithing.StageResolve:
					k.StageResolve()
				case ithing.StageApply:
					k.StageApply()
				}
				evt.Done()
			case <-k.done:
				return
			}
		}
	}()
}

// HasThinking reports false because pickup keys are passive until touched.
func (k *Key) HasThinking() bool {
	return false
}

// StageThinking is a no-op for passive pickup keys.
func (k *Key) StageThinking(playerX float64, playerY float64, playerZ float64) {
}

func (k *Key) onKeyCollision(other collisions.IObject, energy float64) {
	if holder, ok := other.(IKeyHolder); ok {
		if k.collected.CompareAndSwap(false, true) {
			holder.GiveKey(k.keyId)
			k.SetActive(false)
		}
	}
	k.onElasticCollision(other, energy)
}
