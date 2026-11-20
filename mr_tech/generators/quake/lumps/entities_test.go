package lumps

import "testing"

func TestEntityIsLevelAvailable(t *testing.T) {
	tests := []struct {
		name  string
		props map[string]string
		level int
		want  bool
	}{
		{name: "no spawnflags", props: map[string]string{}, level: 1, want: true},
		{name: "not_easy on easy", props: map[string]string{"spawnflags": "256"}, level: 1, want: false},
		{name: "not_easy on medium", props: map[string]string{"spawnflags": "256"}, level: 3, want: true},
		{name: "hard only on easy", props: map[string]string{"spawnflags": "768"}, level: 1, want: false},
		{name: "hard only on medium", props: map[string]string{"spawnflags": "768"}, level: 3, want: false},
		{name: "hard only on hard", props: map[string]string{"spawnflags": "768"}, level: 4, want: true},
		{name: "dm only hidden on easy", props: map[string]string{"spawnflags": "1792"}, level: 1, want: false},
		{name: "sp only visible on easy", props: map[string]string{"spawnflags": "2048"}, level: 1, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ent := &Entity{Properties: tc.props}
			if got := ent.IsLevelAvailable(tc.level); got != tc.want {
				t.Fatalf("IsLevelAvailable(%d) = %v, want %v (spawnflags=%q)", tc.level, got, tc.want, tc.props["spawnflags"])
			}
		})
	}
}
