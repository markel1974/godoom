package config

// KeySpec defines key identity and optional target metadata for a ThingKeyDef.
type KeySpec struct {
	KeyId string `json:"keyId"`
	//Target string `json:"target,omitempty"`
}

func NewKeySpec(keyId string) *KeySpec {
	return &KeySpec{
		KeyId: keyId,
		//Target: target,
	}
}
