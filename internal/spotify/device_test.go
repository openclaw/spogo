package spotify

import "testing"

func TestConnectDeviceIDsAreCaseSensitive(t *testing.T) {
	state := connectState{devices: map[string]any{"OPAQUE-ID": map[string]any{"name": "Desk"}}}
	if got := resolveConnectTargetDeviceID(state, "opaque-id"); got != "" {
		t.Fatalf("target = %q, want no match for different ID", got)
	}
}

func TestFindDevice(t *testing.T) {
	devices := []Device{{ID: "name-match", Name: "opaque-id"}, {ID: "opaque-id", Name: "Desk"}, {Name: "Unavailable"}}
	for _, test := range []struct {
		selector, wantID string
		found            bool
	}{
		{"opaque-id", "opaque-id", true},
		{"desk", "opaque-id", true},
		{"DESK", "opaque-id", true},
		{"Unavailable", "", true},
		{"", "", false},
		{"Missing", "", false},
	} {
		t.Run(test.selector, func(t *testing.T) {
			got, found := FindDevice(devices, test.selector)
			if found != test.found || got.ID != test.wantID {
				t.Fatalf("device = %q, found = %v; want %q, %v", got.ID, found, test.wantID, test.found)
			}
		})
	}
}

func TestConnectPreservesOpaqueSelectorWhitespace(t *testing.T) {
	state := connectState{devices: map[string]any{
		"id":   map[string]any{"name": "Desk"},
		" id ": map[string]any{"name": "Kitchen"},
	}}
	if got := resolveConnectTargetDeviceID(state, " id "); got != " id " {
		t.Fatalf("target = %q, want exact unmodified ID", got)
	}
}
