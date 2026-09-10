package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFollowPlaylist(t *testing.T) {
	for _, public := range []bool{true, false} {
		t.Run(fmt.Sprint(public), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut || r.URL.Path != "/me/library" || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				body, _ := io.ReadAll(r.Body)
				if len(body) != 0 {
					t.Errorf("unexpected body: %s", body)
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer srv.Close()
			client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
			if err != nil {
				t.Fatal(err)
			}
			if err := client.FollowPlaylist(context.Background(), "p1", public); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientUnfollowPlaylist(t *testing.T) {
	var gotMethod string
	mux := http.NewServeMux()
	mux.HandleFunc("/me/library", func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected query: %s", r.URL.RawQuery)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	if err := client.UnfollowPlaylist(context.Background(), "p1"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("method = %s, want DELETE", gotMethod)
	}
}

func TestClientIsFollowingPlaylistTrue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me/library/contains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		_ = json.NewEncoder(w).Encode([]bool{true})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	following, err := client.IsFollowingPlaylist(context.Background(), "p1")
	if err != nil {
		t.Fatalf("is following: %v", err)
	}
	if !following {
		t.Fatalf("expected following = true")
	}
}

func TestClientIsFollowingPlaylistFalse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me/library/contains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		_ = json.NewEncoder(w).Encode([]bool{false})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	following, err := client.IsFollowingPlaylist(context.Background(), "p1")
	if err != nil {
		t.Fatalf("is following: %v", err)
	}
	if following {
		t.Fatalf("expected following = false")
	}
}

func TestClientIsFollowingPlaylistEmptyResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me/library/contains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		_ = json.NewEncoder(w).Encode([]bool{})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	if _, err := client.IsFollowingPlaylist(context.Background(), "p1"); err == nil {
		t.Fatalf("expected error for empty contains response")
	}
}

func TestClientIsFollowingPlaylistMalformedResponse(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/me/library/contains", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Query().Encode() != "uris=spotify%3Aplaylist%3Ap1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		_, _ = w.Write([]byte(`{"not":"an array"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Options{TokenProvider: staticTokenProvider{}, BaseURL: srv.URL})
	if err != nil {
		t.Fatalf("client: %v", err)
	}

	if _, err := client.IsFollowingPlaylist(context.Background(), "p1"); err == nil {
		t.Fatalf("expected error for malformed contains response")
	}
}
