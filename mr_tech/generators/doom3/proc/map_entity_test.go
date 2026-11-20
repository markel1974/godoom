package proc

import (
	"math"
	"testing"

	"github.com/markel1974/godoom/mr_tech/geometry"
)

func TestMapEntityIsDormant(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]string
		want  bool
	}{
		{name: "active empty", props: map[string]string{"classname": "monster_demon_imp"}, want: false},
		{name: "zero flags", props: map[string]string{"teleport": "0", "hide": "0", "trigger": "0", "num_cinematics": "0", "spawner": "0"}, want: false},
		{name: "teleport", props: map[string]string{"teleport": "1"}, want: true},
		{name: "hide 1", props: map[string]string{"hide": "1"}, want: true},
		{name: "hide 2", props: map[string]string{"hide": "2"}, want: true},
		{name: "hide true", props: map[string]string{"hide": "true"}, want: true},
		{name: "trigger", props: map[string]string{"trigger": "1"}, want: true},
		{name: "bind", props: map[string]string{"bind": "mover_1"}, want: true},
		{name: "trigger_anim", props: map[string]string{"trigger_anim": "16hole"}, want: true},
		{name: "num_cinematics", props: map[string]string{"num_cinematics": "2"}, want: true},
		{name: "spawner", props: map[string]string{"spawner": "3"}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ent := &MapEntity{Properties: tc.props}
			if got := ent.IsDormant(); got != tc.want {
				t.Fatalf("IsDormant() = %v, want %v (props=%v)", got, tc.want, tc.props)
			}
		})
	}
}

func TestMapEntityIsLevelAvailable(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]string
		level int
		want  bool
	}{
		{name: "default spawns everywhere", props: map[string]string{}, level: 1, want: true},
		{name: "not_easy hidden on 1", props: map[string]string{"not_easy": "1"}, level: 1, want: false},
		{name: "not_easy spawns on 3", props: map[string]string{"not_easy": "1"}, level: 3, want: true},
		{name: "not_medium hidden on 3", props: map[string]string{"not_medium": "1"}, level: 3, want: false},
		{name: "not_hard hidden on 4", props: map[string]string{"not_hard": "1"}, level: 4, want: false},
		{name: "not_nightmare hidden on 5", props: map[string]string{"not_nightmare": "1"}, level: 5, want: false},
		{name: "not_easy=0 still spawns", props: map[string]string{"not_easy": "0"}, level: 1, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ent := &MapEntity{Properties: tc.props}
			if got := ent.IsLevelAvailable(tc.level); got != tc.want {
				t.Fatalf("IsLevelAvailable(%d) = %v, want %v", tc.level, got, tc.want)
			}
		})
	}
}

func TestMapEntityOrientation(t *testing.T) {
	// Test angle
	ent1 := &MapEntity{Properties: map[string]string{"angle": "90"}}
	if got := ent1.GetAngle(); got != 90 {
		t.Fatalf("GetAngle() with angle=90 got %f, want 90", got)
	}

	// Test angles (pitch yaw roll)
	ent2 := &MapEntity{Properties: map[string]string{"angles": "0 180 0"}}
	if got := ent2.GetAngle(); got != 180 {
		t.Fatalf("GetAngle() with angles='0 180 0' got %f, want 180", got)
	}

	// Test rotation matrix (90 degrees around Z: m00=0, m01=1, m02=0, m10=-1, m11=0, m12=0, m20=0, m21=0, m22=1)
	ent3 := &MapEntity{Properties: map[string]string{"rotation": "0 1 0 -1 0 0 0 0 1"}}
	if got := ent3.GetAngle(); got != 90 {
		t.Fatalf("GetAngle() with rotation got %f, want 90", got)
	}

	// Test TransformPoint with angles
	ent4 := &MapEntity{Properties: map[string]string{
		"origin": "100 200 50",
		"angles": "0 90 0",
	}}
	p := ent4.TransformPoint(geometry.XYZ{X: 10, Y: 0, Z: 0})
	// 10 units forward (+X) rotated 90 deg counter-clockwise should land at +Y: (100, 210, 50)
	if math.Abs(p.X-100.0) > 1e-3 || math.Abs(p.Y-210.0) > 1e-3 || math.Abs(p.Z-50.0) > 1e-3 {
		t.Fatalf("TransformPoint got (%f, %f, %f), want (100, 210, 50)", p.X, p.Y, p.Z)
	}
}
