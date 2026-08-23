package spotify

import (
	"reflect"
	"testing"
)

func TestParseMapLiteral(t *testing.T) {
	raw := `{1:"foo",2:"bar"}`
	m, err := parseMapLiteral(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if m[1] != "foo" || m[2] != "bar" {
		t.Fatalf("unexpected map: %#v", m)
	}
}

func TestParseWebpackMaps(t *testing.T) {
	js := `var a={1:"alpha-beta",2:"beta-gamma"};var b={1:"a1b2c3d4",2:"e5f6a7b8"};`
	nameMap, hashMap, err := parseWebpackMaps(js)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if nameMap[1] != "alpha-beta" && nameMap[2] != "beta-gamma" {
		t.Fatalf("unexpected name map")
	}
	if hashMap[1] != "a1b2c3d4" && hashMap[2] != "e5f6a7b8" {
		t.Fatalf("unexpected hash map")
	}
}

func TestFindOperationHashes(t *testing.T) {
	body := `foo searchDesktop bar sha256Hash":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`
	found := findOperationHashes(body, []string{"searchDesktop"})
	if found["searchDesktop"] == "" {
		t.Fatalf("expected hash")
	}
}

func TestFindOperationHashesAltPattern(t *testing.T) {
	body := `"searchDesktop","query","0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"`
	found := findOperationHashes(body, []string{"searchDesktop"})
	if found["searchDesktop"] == "" {
		t.Fatalf("expected hash")
	}
}

func TestScoreMapsEmpty(t *testing.T) {
	if scoreHashMap(nil) != 0 {
		t.Fatalf("expected 0 for empty hash map")
	}
	if scoreNameMap(nil) != 0 {
		t.Fatalf("expected 0 for empty name map")
	}
}

func TestPrioritizeOperationChunks(t *testing.T) {
	tests := []struct {
		name       string
		operations []string
		want       []string
	}{
		{
			name:       "search route before incidental search chunks",
			operations: []string{"searchDesktop"},
			want:       []string{"xpui-routes-search.abc.js", "xpui-routes-recent-searches.def.js", "xpui-routes-album.ghi.js", "xpui-home.jkl.js"},
		},
		{
			name:       "album route before unrelated chunks",
			operations: []string{"getAlbum"},
			want:       []string{"xpui-routes-album.ghi.js", "xpui-routes-recent-searches.def.js", "xpui-home.jkl.js", "xpui-routes-search.abc.js"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			chunks := []string{"xpui-routes-recent-searches.def.js", "xpui-routes-album.ghi.js", "xpui-home.jkl.js", "xpui-routes-search.abc.js"}
			if got := prioritizeOperationChunks(chunks, test.operations); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("prioritized chunks = %#v, want %#v", got, test.want)
			}
		})
	}
}
