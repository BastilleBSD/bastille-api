package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// testKey installs a single API key into the global config and returns the
// header values needed to authenticate as it.
func testKey(t *testing.T, id, secret string, bastille, admin []string) (authz, authzID string) {
	t.Helper()
	salt := "test-salt"
	cfg = &ConfigStruct{
		APIKeys: map[string]APIKeyStruct{
			id: {
				Salt: salt,
				Hash: generateHash(secret, salt),
				Permissions: PermissionsStruct{
					Bastille: bastille,
					Admin:    admin,
				},
			},
		},
	}
	return "Bearer " + secret, id
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	loadRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func doGet(t *testing.T, srv *httptest.Server, path, authz, authzID string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srv.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	if authzID != "" {
		req.Header.Set("Authorization-ID", authzID)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

// The console proxy must reject unauthenticated requests. This is the guard for
// finding C1 (previously the route had no auth middleware at all).
func TestConsoleRequiresAuth(t *testing.T) {
	InitLogger(false)
	const path = "/api/v1/bastille/console/ttyd/"

	t.Run("no credentials -> 401", func(t *testing.T) {
		testKey(t, "op", "s3cret", []string{"console"}, nil)
		srv := newTestServer(t)
		if code := doGet(t, srv, path, "", ""); code != http.StatusUnauthorized {
			t.Fatalf("unauthenticated console request: got %d, want 401", code)
		}
	})

	t.Run("valid credentials pass auth", func(t *testing.T) {
		authz, authzID := testKey(t, "op", "s3cret", []string{"console"}, nil)
		srv := newTestServer(t)
		// ttyd is not running in the test environment, so the proxy layer returns
		// 503. The point is that auth passed: the response is NOT 401/403.
		code := doGet(t, srv, path, authz, authzID)
		if code == http.StatusUnauthorized || code == http.StatusForbidden {
			t.Fatalf("authenticated console request was rejected by auth: got %d", code)
		}
	})

	t.Run("key without console permission -> 403", func(t *testing.T) {
		authz, authzID := testKey(t, "op", "s3cret", []string{"list"}, nil)
		srv := newTestServer(t)
		if code := doGet(t, srv, path, authz, authzID); code != http.StatusForbidden {
			t.Fatalf("console without permission: got %d, want 403", code)
		}
	})
}
