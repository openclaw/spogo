package spotify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/steipete/spogo/internal/cookies"
)

type playlistRouteStub struct {
	API
	err   error
	calls int
}

func (s *playlistRouteStub) FollowPlaylist(context.Context, string) error {
	s.calls++
	return s.err
}

func (s *playlistRouteStub) UnfollowPlaylist(context.Context, string) error {
	s.calls++
	return s.err
}

func (s *playlistRouteStub) IsFollowingPlaylist(context.Context, string) (bool, error) {
	s.calls++
	return s.err == nil, s.err
}

var playlistOperations = []struct {
	name   string
	invoke func(API) error
}{
	{"follow", func(api API) error { return api.FollowPlaylist(context.Background(), "p1") }},
	{"unfollow", func(api API) error { return api.UnfollowPlaylist(context.Background(), "p1") }},
	{"following", func(api API) error {
		following, err := api.IsFollowingPlaylist(context.Background(), "p1")
		if err == nil && !following {
			return errors.New("lost membership result")
		}
		return err
	}},
}

func TestPlaylistWebAndConnectRouting(t *testing.T) {
	for _, engine := range []string{"web", "connect", "auto"} {
		t.Run(engine, func(t *testing.T) {
			var methods []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				wantPath := "/me/library"
				if r.Method == http.MethodGet {
					wantPath += "/contains"
				}
				if r.URL.Path != wantPath || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Method == http.MethodGet {
					_, _ = w.Write([]byte(`[true]`))
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()
			web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			var client API = web
			switch engine {
			case "connect":
				client = &ConnectClient{web: web}
			case "auto":
				client = NewAutoClient(&ConnectClient{web: web}, web)
			}
			for _, op := range playlistOperations {
				if err := op.invoke(client); err != nil {
					t.Fatalf("%s: %v", op.name, err)
				}
			}
			if len(methods) != 3 || methods[0] != "PUT" || methods[1] != "DELETE" || methods[2] != "GET" {
				t.Fatalf("requests: %v", methods)
			}
		})
	}
}

func TestAutoPlaylistRouting(t *testing.T) {
	failure := errors.New("upstream unavailable")
	for _, providerErr := range []error{nil, ErrUnsupported, APIError{Status: 429}, APIError{Status: 403}, cookies.ErrNoCookies, ErrOAuthAuthentication, failure} {
		for _, secondaryAvailable := range []bool{true, false} {
			for _, op := range playlistOperations {
				primary := &playlistRouteStub{err: providerErr}
				web := &playlistRouteStub{err: providerErr}
				local := &playlistRouteStub{}
				var secondary API
				if secondaryAvailable {
					secondary = web
				}
				err := op.invoke(NewAutoClient(primary, secondary, local))
				if !errors.Is(err, providerErr) {
					t.Fatalf("%s: error=%v want=%v", op.name, err, providerErr)
				}
				wantPrimary, wantWeb := 1, 0
				if secondaryAvailable {
					wantPrimary, wantWeb = 0, 1
				}
				if primary.calls != wantPrimary || web.calls != wantWeb || local.calls != 0 {
					t.Fatalf("%s: calls primary=%d web=%d local=%d", op.name, primary.calls, web.calls, local.calls)
				}
			}
		}
	}
}

func TestPlaylistWebErrorsDoNotRetryThroughConnect(t *testing.T) {
	for _, apiErr := range []error{APIError{Status: 429}, cookies.ErrNoCookies, ErrOAuthAuthentication} {
		for _, op := range playlistOperations {
			web := &playlistRouteStub{err: apiErr}
			connect := &playlistRouteStub{}
			if err := op.invoke(NewPlaybackFallbackClient(web, connect)); !errors.Is(err, apiErr) {
				t.Fatalf("%s: %v", op.name, err)
			}
			if web.calls != 1 || connect.calls != 0 {
				t.Fatalf("%s: web=%d connect=%d", op.name, web.calls, connect.calls)
			}
		}
	}
}

func TestPlaylistClientHTTPFailures(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		for _, op := range playlistOperations {
			t.Run(http.StatusText(status)+"/"+op.name, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Retry-After", "17")
					http.Error(w, "fixture error", status)
				}))
				defer srv.Close()
				web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
				if err != nil {
					t.Fatal(err)
				}
				var apiErr APIError
				if err := op.invoke(&ConnectClient{web: web}); !errors.As(err, &apiErr) || apiErr.Status != status {
					t.Fatalf("error: %v", err)
				}
			})
		}
	}
}

func TestPlaylistContainsRejectsInvalidCardinalityAndNull(t *testing.T) {
	for _, body := range []string{`null`, `[null]`, `[true,false]`} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer srv.Close()
			web, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := web.IsFollowingPlaylist(context.Background(), "p1"); err == nil {
				t.Fatal("accepted invalid response")
			}
		})
	}
}
