# Bastille API — Release-Readiness Plan

**Companion to:** `AUDIT.md`
**Date:** 2026-07-09
**Goal:** Take the experimental API from "generally functional" to "safe to tag and ship."

This plan is organized as phases. **Phase 0 is the release gate** — the project should not cut a
public release until those items are done, because they are exploitable in a default install
regardless of network placement. Later phases are hardening, quality, and maintainability.

A dedicated section, **["Reverse-Proxy Deployment"](#reverse-proxy-deployment)**, addresses the
maintainer's preferred topology (API behind a TLS-terminating reverse proxy), spells out exactly
which findings a proxy does and does not mitigate, and proposes how to make that wiring easy for
users who "just install a couple of packages."

---

## Phase 0 — Release blockers (must fix in the application)

These are **not** mitigated by a reverse proxy (see the table below). They must be fixed in code
before release.

### 0.1 Re-enable console authentication (C1)
`api/routes.go:28` — restore the auth middleware on the console proxy:

```go
bastilleConsole.Any("/*any", apiKeyMiddleware("bastille", "console"), consoleProxy("http://localhost:7681"))
```
Remove the unauthenticated line. Do this even if a proxy will also gate the path — defense in depth,
and the internal port must never be a bare root shell.

### 0.2 Eliminate the default credential (C3)
- Remove the working key from `config.json.sample`; ship it with an empty `apiKeys` map.
- On startup, if the config contains the known default hash **or** no keys at all, either (a) refuse
  to start with a clear message, or (b) generate a random key + id, write it `0600`, and print it
  **once** to stdout/log for the operator to capture.
- Document that the first key must be created out-of-band or via the generated bootstrap key.

### 0.3 Make the bind address safe for the proxy model (prerequisite for everything below)
**This is currently a bug.** `api/api.go:43-48`:

```go
if Host == "0.0.0.0" || Host == "localhost" || Host == "" {
    bindAddr = "0.0.0.0"          // <-- binds ALL interfaces
    Host = "localhost"
}
```

Setting `host: "localhost"` in the config actually binds the API to **0.0.0.0 (all interfaces)**,
which defeats the entire "keep it behind a proxy on localhost" strategy. Fix so that:
- `localhost` / empty → bind `127.0.0.1` (loopback only) — and make this the **default**.
- An explicit external address or `0.0.0.0` is opt-in, and when chosen, startup logs a warning that
  TLS/authentication must be handled directly.

### 0.4 TLS decision (C2)
Pick one and document it as the supported install:
- **Recommended:** API binds `127.0.0.1`, TLS is terminated by a reverse proxy (see below). No TLS
  code in the app.
- **Alternative:** add native `RunTLS` support (cert/key paths in config) for users who do not want a
  proxy.

Either way, the README must stop showing `http://` against a network address as the normal path.

### 0.5 Stop leaking secrets in logs (H1, H4)
- `api/middleware.go` — strip `X-API-Key`, `X-API-Key-ID`, and `Authorization-ID` from the cloned
  headers (or switch to a safe-header allowlist).
- Stop logging `RawQuery` at info level; if request logging is needed, log method + path only.
- Move mutating parameters out of the query string into the POST body as part of Phase 2.

### 0.6 Lock down the config file (H2)
`api/config.go:72` — write with `0600`, and create the parent directory `0700`. The file contains
key salts and hashes.

**Phase 0 exit criteria:** a default `make install` produces a server that (a) has no working
credential until the operator creates one, (b) binds loopback by default, (c) never exposes an
unauthenticated shell, and (d) does not write secrets to logs or world-readable files.

---

## Phase 1 — Hardening (should ship in the first release or the one right after)

| Item | Finding | Action |
|------|---------|--------|
| Exec timeout | E2 | Use `exec.CommandContext` with a configurable timeout; never hold the global lock on a hung process. |
| Concurrency | E1 | Move off a single global mutex — per-jail locking, and let read-only commands (`list`, `top`, `limits show`) run without the write lock. |
| HTTP timeouts | M4 | Construct an explicit `http.Server` with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`. |
| Argument injection | M3 | Insert a `--` end-of-options separator before positional args; reject positional values beginning with `-` unless explicitly whitelisted. |
| Key hashing | H3 | Enforce server-generated high-entropy keys, or adopt argon2id/scrypt/bcrypt. |
| Rate limiting | M2 | Per-IP throttle + auth-failure backoff (can be delegated to the proxy — see table). |
| CORS | M1 | Make the allowed origin configurable; default closed. |
| Admin authZ | M5 | Redefine admin scope to manage keys by ID without needing each key's own secret. |
| Robustness | Mn3 | Check `loadBastilleSpec()`'s error at `api/api.go:52`; nil-check `subcmd` in `ValidateBastilleCommandParameters`. |
| Hash safety | L1 | `compareHash` should return false on hex-decode error. |

---

## Phase 2 — Maintainability & quality (post-release, high value)

- **Schema-driven handlers (Mn1).** Extend `api/bastille.json` to describe each parameter's rules
  (required, positional order, allowed values, subcommand branching) and drive a **single generic
  handler** from it. Target: collapse `api/bastille.go` from ~1,900 lines to a few hundred and make
  adding a command a data change. Handle the branchy commands (`limits`, `zfs`, `etcupdate`, `rdr`,
  `monitor`) with a small conditional grammar.
- **Tests (Mn2).** Table-driven tests for `ValidateBastilleCommandParameters`, the auth middleware,
  argument construction, and admin key lifecycle. This is a precondition for safely doing the
  refactor above.
- **API consistency (B1, B2, B3).** One error envelope everywhere; remove the redundant `+`→space
  handling; add `/healthz` and `/version`.
- **Cleanups.** Remove dead `BastilleCommandOutputStruct`; fix the `"missin"` typo; pass `c` to
  `logRequest` consistently; `go build .` in the Makefile; unexport globals that need not be exported.
- **ttyd lifecycle (E4).** Track and reap ttyd processes; allocate ports dynamically instead of the
  fixed `7681`.
- **Streaming (E3).** Consider streaming command output instead of buffering with `CombinedOutput`.

---

## Reverse-Proxy Deployment

The maintainer's preferred (and recommended) topology: **API bound to `127.0.0.1`, a reverse proxy
terminates TLS and faces the network.** This section defines exactly what that buys you and how to
make it turnkey.

### What a reverse proxy mitigates — and what it does not

| Finding | Mitigated by a proxy? | Notes |
|---------|-----------------------|-------|
| **C2** No TLS | ✅ Yes | Proxy terminates TLS; API on loopback speaks plain HTTP safely. This is the primary reason for the topology. |
| **H4** Secrets in transit | ✅ Yes (wire only) | TLS protects keys/args on the network. Local log leakage still requires the Phase 0.5 fix. |
| **M4** Slowloris / slow clients | ✅ Yes | nginx/Caddy buffer and time out slow clients before they reach the API. |
| **M2** Rate limiting | ✅ Yes | Can be enforced at the proxy (`limit_req` / Caddy `rate_limit`), which is often easier than in-app. |
| **M1** CORS | 🟡 Partial | Proxy can override/restrict CORS headers, but the app still emits `*`; best fixed in both. |
| **C1** Unauthenticated console | 🟡 Partial (defense-in-depth only) | A proxy can require auth (mTLS/basic) on `/api/v1/bastille/console/`, but the internal port is still a bare root shell. **Must still fix in-app (0.1).** |
| **E1/E2** Global lock / hung exec | 🟡 Cosmetic only | Proxy can return 504 to the client, but the API remains internally deadlocked. **Not** a real mitigation. |
| **C3** Default credential | ❌ No | Application-level. |
| **H1** Secrets in debug logs | ❌ No | Application-level. |
| **H2** Config file perms | ❌ No | Application-level. |
| **H3** Weak hashing | ❌ No | Application-level. |
| **M3** Argument injection | ❌ No | Authenticated-user-level; proxy cannot see it. |
| **M5** Admin authZ model | ❌ No | Application-level. |

**Takeaway:** a proxy is necessary for TLS and helpful for rate-limiting/slowloris, but it does
**not** substitute for Phase 0. The proxy and the in-app fixes are complementary — ship both.

### Hard requirement for the proxy model

The API **must** bind loopback (Phase 0.3). If it binds `0.0.0.0`, clients can bypass the proxy and
hit plain HTTP directly, nullifying TLS. Recommended default config:

```json
{ "host": "127.0.0.1", "port": "8888", "apiKeys": {} }
```

### Recommended proxy: Caddy (easiest to automate)

Caddy is in FreeBSD ports (`pkg install caddy`) and does automatic HTTPS. For an
internet-reachable host with a real DNS name it obtains and renews Let's Encrypt certs with **zero
extra config** — the closest thing to "install a couple of packages and go."

`/usr/local/etc/caddy/Caddyfile`:

```caddy
api.example.org {
    # Automatic HTTPS via Let's Encrypt.
    reverse_proxy 127.0.0.1:8888

    # Optional: rate limit (M2) and slow-client protection (M4) live here.

    # Optional defense-in-depth on the console (C1) — still keep in-app auth.
    @console path /api/v1/bastille/console/*
    # basic_auth @console { ... }   # or mTLS, if desired
}
```

For **LAN / internal** installs (no public DNS, so no Let's Encrypt), use Caddy's internal CA or a
self-signed cert:

```caddy
https://bastille.lan {
    tls internal          # Caddy's local CA; clients must trust its root
    reverse_proxy 127.0.0.1:8888
}
```

### Alternative: nginx

`/usr/local/etc/nginx/bastille-api.conf`:

```nginx
# Rate limit zone (M2)
limit_req_zone $binary_remote_addr zone=bastille_api:10m rate=10r/s;

server {
    listen 443 ssl;
    server_name api.example.org;

    ssl_certificate     /usr/local/etc/ssl/bastille-api.crt;
    ssl_certificate_key /usr/local/etc/ssl/bastille-api.key;

    # Slowloris protection (M4)
    client_body_timeout 10s;
    client_header_timeout 10s;

    location /api/v1/bastille/console/ {
        # WebSocket upgrade for ttyd
        proxy_pass http://127.0.0.1:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }

    location / {
        limit_req zone=bastille_api burst=20 nodelay;
        proxy_pass http://127.0.0.1:8888;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

> **Note on the console:** the ttyd console is a WebSocket. Whichever proxy is used, the console
> location needs WebSocket upgrade handling (shown above for nginx; Caddy's `reverse_proxy` handles
> upgrades automatically).

### Making the proxy wiring easy for package-based installs

The maintainer's concern — users who "just install a couple of packages" cannot be expected to
hand-write proxy configs. Options, roughly in order of least user effort:

1. **Ship ready-to-use sample proxy configs in the repo** (`etc/caddy/Caddyfile.sample`,
   `etc/nginx/bastille-api.conf.sample`) and install them alongside the existing config sample. Users
   copy one file and edit the hostname.
2. **Add a FreeBSD `pkg-message`** (post-install note) that tells the operator exactly which two
   packages to install (`bastille-api` + `caddy`), where the sample Caddyfile is, and the one line to
   edit. This is the standard FreeBSD idiom for "you have one more step."
3. **Default to loopback + Caddy in docs.** Document Caddy as the blessed path precisely because its
   automatic-HTTPS story requires the least configuration. Recommend nginx only for users who already
   run it.
4. **(Later) A `bastille-api setup` helper** subcommand that writes a starter Caddyfile from the
   configured hostname/port and prints next steps — turning "hand-write a proxy config" into "answer
   one prompt." This is the most automated option but the most work; defer past first release.

What should **not** be attempted: silently starting or reconfiguring a proxy on the user's behalf
from the API's rc.d script. Proxy config is host policy; generate samples and document, but let the
operator own the wiring.

---

## Milestone checklist

**Release gate (Phase 0):**
- [ ] Console route requires auth (0.1)
- [ ] No working default credential ships; bootstrap flow defined (0.2)
- [ ] Binds `127.0.0.1` by default; bind bug fixed (0.3)
- [ ] TLS story chosen and documented (0.4)
- [ ] Secrets stripped from logs; query not logged (0.5)
- [ ] Config file written `0600` (0.6)
- [ ] Sample Caddy/nginx configs shipped + `pkg-message`

**First-release hardening (Phase 1):**
- [ ] Exec timeout (E2) and concurrency rework (E1)
- [ ] HTTP server timeouts (M4)
- [ ] `--` arg separator / positional validation (M3)
- [ ] Key hashing / entropy (H3), CORS config (M1), admin authZ (M5)
- [ ] Spec-load + nil-subcmd guards (Mn3), compareHash fix (L1)

**Quality (Phase 2):**
- [ ] Schema-driven handler refactor (Mn1) behind a test suite (Mn2)
- [ ] Error-envelope + `+`-handling + health/version cleanups (B1/B2/B3)
- [ ] ttyd lifecycle (E4), dead code / build / log cleanups
