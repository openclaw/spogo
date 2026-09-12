//go:build darwin

package spotify

import (
	"errors"
	"testing"
)

func TestAppleScriptPlaylistDelegation(t *testing.T) {
	failure := errors.New("web request failed")
	for _, op := range playlistOperations {
		if err := op.invoke(&AppleScriptClient{}); !errors.Is(err, ErrUnsupported) {
			t.Fatalf("%s without fallback: %v", op.name, err)
		}
		for _, wantErr := range []error{nil, failure} {
			web := &playlistRouteStub{err: wantErr}
			if err := op.invoke(&AppleScriptClient{fallback: web}); !errors.Is(err, wantErr) {
				t.Fatalf("%s: %v", op.name, err)
			}
			if web.calls != 1 {
				t.Fatalf("%s: calls=%d", op.name, web.calls)
			}
		}
	}
}
