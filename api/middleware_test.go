package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// redactHeaders must strip every credential-bearing header while leaving benign
// headers intact. Guard for finding H1 (previously only "Authorization" was
// removed, leaking X-API-Key in debug logs).
func TestRedactHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Authorization", "Bearer supersecret")
	h.Set("Authorization-ID", "keyid")
	h.Set("X-API-Key", "new-key-secret")
	h.Set("X-API-Key-ID", "new-key-id")
	h.Set("User-Agent", "curl/8.0")
	h.Set("Content-Type", "application/json")

	got := redactHeaders(h)

	for _, secret := range []string{"Authorization", "Authorization-ID", "X-API-Key", "X-API-Key-ID"} {
		if v := got.Get(secret); v != "" {
			t.Errorf("header %q was not redacted: got %q", secret, v)
		}
	}
	if got.Get("User-Agent") != "curl/8.0" {
		t.Errorf("benign header User-Agent was dropped")
	}
	if got.Get("Content-Type") != "application/json" {
		t.Errorf("benign header Content-Type was dropped")
	}

	// The original header must not be mutated.
	if h.Get("Authorization") == "" {
		t.Errorf("redactHeaders mutated the caller's header map")
	}
}

// Info-level request logging must not include the raw query string. Guard for
// finding H4.
func TestInfoLogOmitsQueryString(t *testing.T) {
	var buf bytes.Buffer
	logger = slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := &Ctx{
		Writer:  httptest.NewRecorder(),
		Request: httptest.NewRequest(http.MethodGet, "/api/v1/bastille/cmd?target=jail&command=cat+/etc/master.passwd", nil),
	}

	logRequest("info", "request", c, nil, nil)

	out := buf.String()
	if strings.Contains(out, "master.passwd") || strings.Contains(out, "command=") {
		t.Fatalf("info log leaked query string: %s", out)
	}
	if !strings.Contains(out, "/api/v1/bastille/cmd") {
		t.Fatalf("info log missing request path: %s", out)
	}
}
