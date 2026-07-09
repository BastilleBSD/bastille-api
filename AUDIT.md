# Bastille API — Baseline Security & Code Audit

**Date:** 2026-07-09
**Scope:** Full repository (`main.go`, `api/*.go`, config, `etc/rc.d`, `Makefile`, `bastille.json` spec)
**Codebase size:** ~2,900 lines of Go (Gin framework)
**Status of code:** Functional, experimental. No code was modified during this audit — assessment only.

---

## Executive Summary

`bastille-api` is a Go/Gin service that wraps the `bastille` CLI, exposing ~40 subcommands as
HTTP endpoints plus a small admin/key-management surface and a ttyd console proxy.

The architecture is sound in its bones. It shells out via `exec.Command` with an **argv array
(no shell interpreter)**, which structurally avoids the most catastrophic class of command-injection
bugs. Authentication is a salted-hash bearer-token scheme with per-key permission scoping.

The most serious issues are **not** in command execution (which is reasonably safe); they are in the
**deployment/security posture** — an unauthenticated console proxy, no TLS, wildcard CORS, and a
shipped default credential — and in **maintainability**, where ~1,900 lines of near-identical handler
boilerplate duplicate an already-existing declarative spec file.

This document is organized into four categories matching the audit request:

- **[0] API best-practices & standards**
- **[1] Code maintainability**
- **[2] Code security**
- **[3] Efficiency & scalability**

Findings are labeled with severity: 🔴 Critical, 🟠 High, 🟡 Medium, Low.

A companion document, **`RELEASE_PLAN.md`**, translates these findings into a prioritized,
actionable path to a releasable version, including reverse-proxy deployment guidance.

---

## [2] Security

Security is presented first because it contains the release-blocking items.

### 🔴 Critical

#### C1 — The console proxy is unauthenticated

`api/routes.go:27-28`:

```go
// bastilleConsole.Any("/*any", apiKeyMiddleware("bastille", "console"), consoleProxy(...))
bastilleConsole.Any("/*any", consoleProxy("http://localhost:7681"))
```

The authenticated version is commented out. Anyone who can reach the API can hit
`/api/v1/bastille/console/ttyd/*` and obtain a **live, writable root terminal** into a jail (ttyd
runs with `-W`). This is the single most serious finding.

**Fix:** Restore `apiKeyMiddleware("bastille", "console")` on the proxy route. Ideally also bind a
console session to the key that initiated it, so one authenticated user cannot attach to another
user's terminal.

#### C2 — No TLS

`api/api.go:58` uses `router.Run(addr)` — plain HTTP. Bearer API keys travel in cleartext on every
request, and the README examples all use `http://`. For a *remote* container-management API this is
disqualifying on any untrusted network.

**Fix:** Support `RunTLS` / a TLS config block, **or** require and document a TLS-terminating reverse
proxy with the API bound to localhost. (This is the crux of the reverse-proxy discussion in
`RELEASE_PLAN.md`.)

#### C3 — Shipped default credential with full privileges

`etc/bastille-api/config.json.sample` — which the README instructs users to `cp` directly into
place — contains:

```json
"bastille": {
  "salt": "my-random-salt",
  "hash": "328de2f095e41086c07746ab22592a6c656045d3a8ff7f0ef1d721760677dea5",
  "permissions": { "bastille": ["*"], "admin": ["*"] }
}
```

Key `bastille-api-key` / id `bastille` grants full bastille **and** admin control. Any deployment
that does not immediately rotate it is fully compromised (remote root + key management).

**Fix:** Refuse to start when the known default hash is present, or generate a random key on first
run and print it once to the operator.

### 🟠 High

#### H1 — Secret leakage in debug logs

`api/middleware.go:54-55` clones request headers and deletes only `Authorization`:

```go
headers := c.Request.Header.Clone()
headers.Del("Authorization")
```

The admin endpoints carry the secret in `X-API-Key` (and `X-API-Key-ID`); those are **not** stripped.
In `--debug` mode the server logs newly-created API key secrets in plaintext.

**Fix:** Strip `X-API-Key`, `X-API-Key-ID`, and `Authorization-ID` as well — or switch to an
allowlist of safe headers instead of a denylist.

#### H2 — Config file written world-readable

`api/config.go:72` writes the config with mode `0644`. The file holds every key's salt and hash. Any
local user can read it and mount an offline attack.

**Fix:** Write with `0600`; set restrictive permissions on the parent directory too.

#### H3 — Fast hash over potentially low-entropy keys

`api/utils.go:23` computes a single `sha256(salt + key)`. That is acceptable for a 256-bit random
token, but the design invites user-chosen keys (the default is literally `bastille-api-key`), which
are trivially brute-forced offline given H2.

**Fix:** Either enforce server-generated high-entropy keys, or move to a slow KDF
(argon2id / scrypt / bcrypt).

#### H4 — Secrets and sensitive arguments in the query string

Every mutating call is a POST that reads parameters from the URL query (`c.Query(...)`), and the
logging middleware records `c.Request.URL.RawQuery` at **info** level (`api/middleware.go:44`). URLs
leak into access logs, proxy logs, and browser/shell history. `bastille cmd` payloads, IP addresses,
and file paths are all logged.

**Fix:** Read parameters from the POST body (form or JSON); never log the raw query string.

### 🟡 Medium

- **M1 — Wildcard CORS.** `Access-Control-Allow-Origin: *` (`api/middleware.go:92`) on a
  root-equivalent management API. Auth is header-based (not cookie-based), so classic CSRF is limited,
  but any web origin can drive the API from a victim's browser if it can obtain a key. Make the
  allowed origin configurable and default it closed.
- **M2 — No rate limiting / lockout.** API-key guessing is unthrottled. Add per-IP throttling and
  failure backoff.
- **M3 — Argument injection into `bastille`.** Positional parameters (`target`, `name`, `ip`,
  `command`, …) are appended verbatim and are **not** checked for a leading `-`, unlike option
  *values* (`api/utils.go:85`). A caller could smuggle bastille flags via e.g. `target=-x`. No shell
  is involved, so this cannot spawn new processes, but it can alter bastille's behavior. Insert a `--`
  end-of-options separator before positional args, and/or validate them.
- **M4 — No server timeouts.** `router.Run` uses a default `http.Server` with no
  `ReadTimeout` / `WriteTimeout` / `IdleTimeout` → slowloris exposure. Construct an explicit
  `http.Server`.
- **M5 — Inconsistent admin authorization model.** `edit`/`delete` require knowing the *target key's
  own secret* (`api/admin.go:227,301`), while `add` does not. This both weakens the mental model and
  makes real key administration impossible (an admin cannot revoke a key they do not hold). Decide
  that admin scope manages keys by ID, full stop.

### Low

- **L1 —** `compareHash` (`api/utils.go:29`) ignores `hex.DecodeString` errors; two empty slices
  compare *equal* under `subtle.ConstantTimeCompare`. Not currently reachable (the generated hash is
  always valid 64-hex), but fragile — return false explicitly on decode error.

---

## [0] API Best-Practices & Standards

- **RPC-over-HTTP, not REST.** Verb-in-path (`/create`, `/destroy`) mirroring the CLI. This is a
  legitimate choice given the "CLI parity" goal — but it should be documented as RPC rather than
  implying REST, and kept consistent.
- **Inconsistent error envelopes.** Admin handlers return `{"error": "..."}` objects; the bastille
  handlers return **bare JSON strings** (e.g. `c.JSON(400, "Missing name parameter")`,
  `api/bastille.go:440`). Clients must parse two shapes. Pick one error schema and use it everywhere.
- **GET-returns-spec overloading.** `GET /bastille/{cmd}` returns the command's option spec while
  `POST` executes it. Workable, but the project already ships a full OpenAPI/Swagger document, making
  the per-endpoint GET somewhat redundant.
- **The `+`-as-space convention is fragile.** `ValidateBastilleCommandParameters` does
  `strings.ReplaceAll(optionsParam, "+", " ")` (`api/utils.go:52`), but Go's query parser already
  decodes `+`→space in `c.Query`. This double-handling mangles any value that legitimately contains
  `+`. Confirm and simplify to standard form-encoding.
- **Missing niceties:** no `/healthz` / readiness endpoint, no `/version`, no request-ID propagation,
  no explicit 404/405 JSON handlers.

---

## [1] Maintainability

**Headline issue: `api/bastille.go` is ~1,900 lines of ~40 nearly-identical handlers.** Each one
manually reads queries, checks for empties, appends args, and emits the same error boilerplate. This
is the single biggest maintainability drag — and it is especially frustrating because the project
**already has a declarative spec** (`api/bastille.json`) describing every command's options and
parameters. That spec is currently used only for light validation, so there are **two sources of
truth** kept in sync by hand.

**High-leverage refactor:** extend `bastille.json` (or a sibling schema) to describe each parameter's
rules (required? positional order? allowed values? subcommand branching), then drive **one generic
handler** from it. This could collapse ~1,900 lines to a couple hundred and make "add a new bastille
command" a data change rather than a code change. The branchy handlers (`limits`, `zfs`, `etcupdate`,
`rdr`, `monitor`) need a small conditional grammar in the schema, but they are the minority.

Other items:

- **No tests at all.** Given the security surface, table-driven tests around
  `ValidateBastilleCommandParameters`, the auth middleware, and argument construction would pay off
  immediately.
- **`loadBastilleSpec()`'s error is ignored** at `api/api.go:52`. If the embedded spec ever fails to
  parse, `bastilleSpec` is nil and **every** validation call panics (`api/utils.go:20`). Relatedly,
  `ValidateBastilleCommandParameters` never nil-checks `subcmd` (`api/utils.go:28`) — a route/spec
  drift becomes a nil-dereference DoS.
- **Dead code:** `BastilleCommandOutputStruct` (`api/structs.go:39`) is unused.
- **Log-context inconsistency:** many handlers pass `nil` instead of `c` to `logRequest` (e.g.
  `BastilleCreateHandler`), so those events lose method/path/client context. There is also a typo,
  `"missin Authorization header"` (`api/middleware.go:118`).
- **Build hygiene:** the `Makefile` runs `go build -o ... main.go` (single-file build); prefer
  `go build -o ... .` so the whole module builds properly.
- **Unnecessarily-exported mutable globals** (`Host`, `Port`, `APIURL` in `api/config.go`).

---

## [3] Efficiency & Scalability

- **🟠 Global mutex serializes the entire API.** `bastilleLock` (`api/bastille.go:12`) is held for the
  *full duration* of every command via `CombinedOutput()`. One `bastille upgrade` / `bootstrap` /
  `pkg` (minutes long) blocks **every other request**, including read-only `list`. If bastille itself
  is not concurrency-safe, some locking is required — but a single global lock covering reads and
  writes alike will not scale past a couple of users. Consider per-jail locking and separating
  read-only commands.
- **🟠 No timeout on command execution.** `exec.Command` plus a held lock means a **hung bastille
  process deadlocks the whole server permanently**. Switch to `exec.CommandContext` with a timeout
  tied to the request.
- **Output fully buffered.** `CombinedOutput` holds all output in memory; long/large outputs (pkg,
  bootstrap) have no streaming or progress for non-"live" calls.
- **ttyd process lifecycle leaks.** `BastilleCommandLive` (`api/bastille.go:33`) starts a new ttyd on
  a fixed port `7681` (`-m 1`) on every call and never reaps it. Concurrent live sessions collide on
  the port, and orphaned ttyd processes accumulate.

---

## Finding Index (by severity)

| ID | Severity | Category | Summary |
|----|----------|----------|---------|
| C1 | 🔴 Critical | Security | Console proxy is unauthenticated (root terminal) |
| C2 | 🔴 Critical | Security | No TLS; API keys sent in cleartext |
| C3 | 🔴 Critical | Security | Shipped default credential with full admin+bastille rights |
| H1 | 🟠 High | Security | `X-API-Key` secrets leaked in debug logs |
| H2 | 🟠 High | Security | Config written world-readable (`0644`) |
| H3 | 🟠 High | Security | Fast SHA-256 hash over possibly low-entropy keys |
| H4 | 🟠 High | Security | Secrets/args in query string, logged at info |
| M1 | 🟡 Medium | Security | Wildcard CORS |
| M2 | 🟡 Medium | Security | No rate limiting / lockout |
| M3 | 🟡 Medium | Security | Argument injection via unvalidated positional params |
| M4 | 🟡 Medium | Security | No HTTP server timeouts (slowloris) |
| M5 | 🟡 Medium | Security | Inconsistent admin authZ model |
| L1 | Low | Security | `compareHash` ignores hex decode errors |
| B1 | — | Best-practice | Inconsistent error envelopes |
| B2 | — | Best-practice | Fragile `+`-as-space handling |
| B3 | — | Best-practice | No health/version endpoints |
| Mn1 | — | Maintainability | ~1,900 lines of duplicated handler boilerplate |
| Mn2 | — | Maintainability | No tests |
| Mn3 | — | Maintainability | Ignored spec-load error → nil-deref panics |
| E1 | 🟠 High | Efficiency | Global mutex serializes all requests |
| E2 | 🟠 High | Efficiency | No exec timeout → possible permanent deadlock |
| E3 | — | Efficiency | Output fully buffered; no streaming |
| E4 | — | Efficiency | ttyd processes leak; fixed port collision |
