package config

import "github.com/markel1974/godoom/mr_tech/geometry"

// KeySpec describes a key identifier associated with a key item or lock interaction.
type KeySpec struct {
	KeyId string `json:"keyId"`
}

// NewKeyDefault creates a KeySpec with default values.
func NewKeyDefault(key string) *KeySpec {
	return &KeySpec{
		KeyId: key,
	}
}

// NewConfigKey creates a key Thing with the specified identity, transform, mass, and key specification ID.
func NewConfigKey(id string, pos geometry.XYZ, angle float64, mass float64, keyId string) *Thing {
	c := NewConfigThing(id, pos, angle, ThingKeyDef, mass, 0, 0, 0)
	c.Key = NewKeyDefault(keyId)
	return c
}
