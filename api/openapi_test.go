package api

import (
	"encoding/json"
	"testing"
)

func parseSpec(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(buildOpenAPISpec(), &doc); err != nil {
		t.Fatalf("generated spec is not valid JSON: %v", err)
	}
	return doc
}

// The generated document must be a well-formed Swagger 2.0 spec covering every
// command route plus the admin routes.
func TestOpenAPISpecShape(t *testing.T) {
	doc := parseSpec(t)

	if doc["swagger"] != "2.0" {
		t.Fatalf("swagger version = %v, want 2.0", doc["swagger"])
	}
	info, ok := doc["info"].(map[string]any)
	if !ok || info["title"] != "BastilleBSD-API" {
		t.Fatalf("info.title missing/wrong: %v", doc["info"])
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatal("paths missing")
	}

	// Every registered bastille command must have a documented path with GET+POST.
	for cmd := range bastilleRoutes() {
		p, ok := paths["/api/v1/bastille/"+cmd].(map[string]any)
		if !ok {
			t.Errorf("missing path for command %q", cmd)
			continue
		}
		if _, ok := p["get"]; !ok {
			t.Errorf("command %q missing GET", cmd)
		}
		if _, ok := p["post"]; !ok {
			t.Errorf("command %q missing POST", cmd)
		}
	}

	for _, admin := range []string{"add", "edit", "delete"} {
		if _, ok := paths["/api/v1/admin/"+admin]; !ok {
			t.Errorf("missing admin path %q", admin)
		}
	}
}

// Spot-check that parameters are derived from the schema: a flat command exposes
// its positionals, and a branchy command exposes its action + branch params.
func TestOpenAPISpecParameters(t *testing.T) {
	doc := parseSpec(t)
	paths := doc["paths"].(map[string]any)

	paramNames := func(command, method string) map[string]bool {
		op := paths["/api/v1/bastille/"+command].(map[string]any)[method].(map[string]any)
		names := map[string]bool{}
		for _, raw := range op["parameters"].([]any) {
			p := raw.(map[string]any)
			names[p["name"].(string)] = true
		}
		return names
	}

	// create (flat) POST must document its positionals and auth headers.
	create := paramNames("create", "post")
	for _, want := range []string{"Authorization", "Authorization-ID", "options", "name", "release", "ip", "iface"} {
		if !create[want] {
			t.Errorf("create POST missing parameter %q", want)
		}
	}

	// zfs (branchy) POST must document the action selector and union of branch params.
	zfs := paramNames("zfs", "post")
	for _, want := range []string{"target", "action", "tag", "key_value", "dataset", "jail_path"} {
		if !zfs[want] {
			t.Errorf("zfs POST missing parameter %q", want)
		}
	}

	// GET (spec) endpoints only need the auth headers.
	get := paramNames("list", "get")
	if !get["Authorization"] || !get["Authorization-ID"] {
		t.Errorf("list GET missing auth headers: %v", get)
	}
}

// registerOpenAPISpec must be safe to call repeatedly (buildHandler runs per
// request-server in tests); a second call must not panic on re-registration.
func TestRegisterOpenAPISpecIdempotent(t *testing.T) {
	registerOpenAPISpec()
	registerOpenAPISpec()
}
