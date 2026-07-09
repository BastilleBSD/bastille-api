package api

// openapi.go generates the OpenAPI (Swagger 2.0) document directly from the
// command schema (declarativeCommands, actionCommands, and the hand-written
// limits command) instead of from handler annotations. This keeps the API
// documentation a byproduct of the same source of truth that drives request
// handling, so the two can never drift.
//
// The generated document is registered with swaggo/swag under the default
// instance name, which is what the swagger UI (swaggo/http-swagger) serves at
// /swagger/doc.json.

import (
	"encoding/json"
	"sort"
	"strings"
	"sync"

	"github.com/swaggo/swag"
)

// queryParam builds a Swagger query parameter object.
func queryParam(name string, required bool, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"type":        "string",
		"required":    required,
		"description": description,
	}
}

// headerParam builds a Swagger header parameter object.
func headerParam(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "header",
		"type":        "string",
		"required":    true,
		"description": description,
	}
}

// authHeaderParams are the credentials every bastille endpoint requires.
func authHeaderParams() []map[string]any {
	return []map[string]any{
		headerParam("Authorization", "Authorization header (eg: Bearer <api-token>)"),
		headerParam("Authorization-ID", "API key ID/name"),
	}
}

// docParams documents a flat command: options, its positionals, and any group.
func (spec commandSpec) docParams() []map[string]any {
	params := []map[string]any{queryParam("options", false, "space-separated CLI options/flags")}
	for _, p := range spec.params {
		params = append(params, queryParam(p.name, p.required, ""))
	}
	for _, p := range spec.group {
		params = append(params, queryParam(p.name, false, "part of an all-or-nothing group"))
	}
	return params
}

// docParams documents a branchy command: options, prefix params, the action
// selector (with its allowed values), and the union of all branch params
// (optional, since which apply depends on the action).
func (ac actionCommand) docParams() []map[string]any {
	params := []map[string]any{queryParam("options", false, "space-separated CLI options/flags")}
	for _, p := range ac.prefix {
		params = append(params, queryParam(p.name, p.required, ""))
	}

	var keys []string
	for k := range ac.branches {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	extraSeen := map[string]bool{}
	var extra []string
	collect := func(seq []argItem) {
		for _, item := range seq {
			if item.param != nil && !extraSeen[item.param.name] {
				extraSeen[item.param.name] = true
				extra = append(extra, item.param.name)
			}
		}
	}
	for _, k := range keys {
		collect(ac.branches[k])
	}
	collect(ac.defaultBranch)
	sort.Strings(extra)

	actionKey := ac.actionParam
	if actionKey == "" {
		actionKey = "action"
	}
	desc := "action selector"
	if len(keys) > 0 {
		desc = "one of: " + strings.Join(keys, ", ")
	}
	params = append(params, queryParam(actionKey, false, desc))
	for _, name := range extra {
		params = append(params, queryParam(name, false, "used by specific actions"))
	}
	return params
}

// commandDocParams returns the documented query parameters for every command.
func commandDocParams() map[string][]map[string]any {
	out := map[string][]map[string]any{}
	for cmd, spec := range declarativeCommands {
		out[cmd] = spec.docParams()
	}
	for cmd, ac := range actionCommands {
		out[cmd] = ac.docParams()
	}
	// limits keeps a bespoke handler; document it explicitly.
	out["limits"] = []map[string]any{
		queryParam("options", false, "space-separated CLI options/flags"),
		queryParam("target", true, ""),
		queryParam("action", true, "add, remove, clear, reset, stats, list, show"),
		queryParam("option", false, "used by add/remove"),
		queryParam("value", false, "used by add"),
		queryParam("args", false, `"active" with list/show`),
	}
	return out
}

func adminOperation(summary string, withScopePerms bool) map[string]any {
	params := []map[string]any{
		headerParam("Authorization", "admin Authorization header (eg: Bearer <api-token>)"),
		headerParam("Authorization-ID", "admin API key ID/name"),
		headerParam("X-API-Key", "the API key to act on"),
		headerParam("X-API-Key-ID", "the API key ID to act on"),
	}
	if withScopePerms {
		params = append(params,
			queryParam("scope", true, "bastille or admin"),
			queryParam("permissions", true, "space-separated permissions, or *"),
		)
	}
	return map[string]any{
		"post": map[string]any{
			"tags":       []string{"admin"},
			"summary":    summary,
			"produces":   []string{"application/json"},
			"parameters": params,
			"responses":  map[string]any{"200": map[string]any{"description": "OK"}},
		},
	}
}

// buildOpenAPISpec assembles the Swagger 2.0 document from the command schema.
func buildOpenAPISpec() []byte {
	descByCmd := map[string]string{}
	if bastilleSpec != nil {
		for _, c := range bastilleSpec.Commands {
			descByCmd[c.Command] = c.Description
		}
	}

	paths := map[string]any{}
	for cmd, dparams := range commandDocParams() {
		post := map[string]any{
			"tags":        []string{"bastille"},
			"summary":     "Run bastille " + cmd,
			"description": descByCmd[cmd],
			"consumes":    []string{"application/x-www-form-urlencoded"},
			"produces":    []string{"text/plain"},
			"parameters":  append(authHeaderParams(), dparams...),
			"responses":   map[string]any{"200": map[string]any{"description": "command output"}},
		}
		get := map[string]any{
			"tags":       []string{"spec"},
			"summary":    "Get supported options and parameters for " + cmd,
			"produces":   []string{"application/json"},
			"parameters": authHeaderParams(),
			"responses":  map[string]any{"200": map[string]any{"description": "command spec"}},
		}
		paths["/api/v1/bastille/"+cmd] = map[string]any{"get": get, "post": post}
	}

	paths["/api/v1/admin/add"] = adminOperation("Add an API key", true)
	paths["/api/v1/admin/edit"] = adminOperation("Edit an API key's permissions", true)
	paths["/api/v1/admin/delete"] = adminOperation("Delete an API key", false)

	spec := map[string]any{
		"swagger": "2.0",
		"info": map[string]any{
			"title":       "BastilleBSD-API",
			"version":     "0.0.1",
			"description": "API interface for Bastille. Command endpoints mirror the bastille CLI; a GET returns the supported parameters/options and a POST runs the command. A /api/v1/bastille/live/{command} variant exists for interactive (ttyd) sessions.",
			"license":     map[string]any{"name": "BSD-3-Clause", "url": "https://opensource.org/license/bsd-3-clause"},
		},
		"basePath": "/",
		"paths":    paths,
	}

	// json.Marshal emits map keys in sorted order, so the document is stable.
	out, _ := json.MarshalIndent(spec, "", "  ")
	return out
}

// generatedSpec adapts a JSON string to the swag.Swagger interface.
type generatedSpec struct{ doc string }

func (g *generatedSpec) ReadDoc() string { return g.doc }

var (
	openAPIHolder   = &generatedSpec{}
	openAPIRegister sync.Once
)

// registerOpenAPISpec (re)builds the OpenAPI document and ensures it is
// registered with swag under the default instance name. Safe to call more than
// once: registration happens once, later calls just refresh the content.
func registerOpenAPISpec() {
	openAPIHolder.doc = string(buildOpenAPISpec())
	openAPIRegister.Do(func() {
		swag.Register(swag.Name, openAPIHolder)
	})
}
