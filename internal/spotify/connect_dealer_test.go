package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestGetConnectionID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		data, _ := json.Marshal(map[string]any{"headers": map[string]any{"Spotify-Connection-Id": "conn-id"}})
		if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	defer srv.Close()

	prev := dealerURL
	dealerURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Cleanup(func() { dealerURL = prev })

	id, err := getConnectionID(context.Background(), "token", 0)
	if err != nil {
		t.Fatalf("getConnectionID: %v", err)
	}
	if id != "conn-id" {
		t.Fatalf("unexpected id: %s", id)
	}
}

func TestDealerConnectionContextTimeout(t *testing.T) {
	tests := []struct {
		name         string
		timeout      time.Duration
		wantTimeout  time.Duration
		wantDeadline bool
	}{
		{name: "zero uses default", timeout: 0, wantTimeout: defaultHTTPClientTimeout, wantDeadline: true},
		{name: "positive uses configured timeout", timeout: 1500 * time.Millisecond, wantTimeout: 1500 * time.Millisecond, wantDeadline: true},
		{name: "negative is unlimited", timeout: -time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := time.Now()
			ctx, cancel := dealerConnectionContext(context.Background(), tt.timeout)
			t.Cleanup(cancel)
			after := time.Now()

			deadline, ok := ctx.Deadline()
			if ok != tt.wantDeadline {
				t.Fatalf("deadline present = %t, want %t", ok, tt.wantDeadline)
			}
			if ok && (deadline.Before(before.Add(tt.wantTimeout)) || deadline.After(after.Add(tt.wantTimeout))) {
				t.Fatalf("deadline = %s, want timeout %s", deadline, tt.wantTimeout)
			}
		})
	}
}

func TestEnsureConnectDeviceRegisters(t *testing.T) {
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		data, _ := json.Marshal(map[string]any{"headers": map[string]any{"Spotify-Connection-Id": "conn-xyz"}})
		if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	defer wsServer.Close()

	prev := dealerURL
	dealerURL = "ws" + strings.TrimPrefix(wsServer.URL, "http")
	t.Cleanup(func() { dealerURL = prev })

	client := newConnectClientForTests(roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return textResponse(http.StatusOK, "ok"), nil
	}))
	client.session.connectDeviceID = "device"
	client.session.connectionID = ""
	client.session.registeredAt = time.Time{}

	auth := connectAuth{AccessToken: "access", ClientToken: "client-token", ClientVersion: "1.0.0"}
	if err := client.ensureConnectDevice(context.Background(), auth); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if client.session.connectionID == "" {
		t.Fatalf("expected connection id")
	}
}

func TestGetConnectionIDMissingHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		data, _ := json.Marshal(map[string]any{"headers": map[string]any{"Other": "nope"}})
		if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	defer srv.Close()

	prev := dealerURL
	dealerURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Cleanup(func() { dealerURL = prev })

	if _, err := getConnectionID(context.Background(), "token", 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetConnectionIDBadHeadersType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		data, _ := json.Marshal(map[string]any{"headers": "bad"})
		if err := conn.Write(r.Context(), websocket.MessageText, data); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	defer srv.Close()

	prev := dealerURL
	dealerURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Cleanup(func() { dealerURL = prev })

	if _, err := getConnectionID(context.Background(), "token", 0); err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetConnectionIDDialError(t *testing.T) {
	prev := dealerURL
	dealerURL = "ws://127.0.0.1:1"
	t.Cleanup(func() { dealerURL = prev })

	const accessToken = "sensitive-test-token"
	_, err := getConnectionID(context.Background(), accessToken, 0)
	if err == nil {
		t.Fatalf("expected error")
	}
	if strings.Contains(err.Error(), accessToken) {
		t.Fatalf("dealer dial error exposed its access token")
	}
	if !strings.Contains(err.Error(), "[REDACTED]") {
		t.Fatalf("dealer dial error omitted its redaction marker: %v", err)
	}
	var networkError net.Error
	if !errors.As(err, &networkError) {
		t.Fatalf("dealer dial error no longer unwraps to a network error: %T", err)
	}
}

func TestGetConnectionIDBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			t.Fatalf("accept: %v", err)
		}
		defer func() { _ = conn.Close(websocket.StatusNormalClosure, "") }()
		if err := conn.Write(r.Context(), websocket.MessageText, []byte("nope")); err != nil {
			t.Fatalf("write: %v", err)
		}
	}))
	defer srv.Close()

	prev := dealerURL
	dealerURL = "ws" + strings.TrimPrefix(srv.URL, "http")
	t.Cleanup(func() { dealerURL = prev })

	if _, err := getConnectionID(context.Background(), "token", 0); err == nil {
		t.Fatalf("expected error")
	}
}
