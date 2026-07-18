package cli

import "testing"

func TestResolveVersion(t *testing.T) {
	for _, test := range []struct {
		name   string
		linked string
		want   string
	}{
		{name: "release linker override", linked: "0.10.3", want: "0.10.3"},
		{name: "release linker override with prefix", linked: " v0.10.3 ", want: "0.10.3"},
		{name: "local build", linked: "dev", want: "dev"},
		{name: "missing linker value", want: "dev"},
	} {
		t.Run(test.name, func(t *testing.T) {
			original := version
			version = test.linked
			t.Cleanup(func() { version = original })
			if got := currentVersion(); got != test.want {
				t.Fatalf("currentVersion() = %q, want %q", got, test.want)
			}
		})
	}
}
