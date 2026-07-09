package api

import (
	"net/http"

	_ "bastille-api/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// buildHandler constructs the fully-wired HTTP handler: a net/http ServeMux with
// all routes, wrapped in the logging and CORS middlewares.
func buildHandler() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// The console proxy exposes a live, writable root terminal (ttyd -W) into a
	// jail. It MUST be authenticated; an unauthenticated route here is a remote
	// root shell for anyone who can reach the listener. The trailing slash makes
	// this a subtree match across all methods.
	const consolePath = "/api/v1/bastille/console/ttyd/"
	mux.Handle(consolePath, chain(consolePath,
		apiKeyMiddleware("bastille", "console"),
		consoleProxy("http://localhost:7681"),
	))

	for path, handler := range bastilleRoutes() {
		base := "/api/v1/bastille/" + path
		live := "/api/v1/bastille/live/" + path

		mux.Handle("GET "+base, chain(base, apiKeyMiddleware("bastille", path), GetCommandSpec(path)))
		mux.Handle("POST "+base, chain(base, apiKeyMiddleware("bastille", path), handler))

		mux.Handle("GET "+live, chain(live, apiKeyMiddleware("bastille", path), GetCommandSpec(path)))
		mux.Handle("POST "+live, chain(live, apiKeyMiddleware("bastille", path), handler))
	}

	mux.Handle("POST /api/v1/admin/add", chain("/api/v1/admin/add", apiKeyMiddleware("admin", "add"), AddKeyHandler))
	mux.Handle("POST /api/v1/admin/edit", chain("/api/v1/admin/edit", apiKeyMiddleware("admin", "edit"), EditKeyHandler))
	mux.Handle("POST /api/v1/admin/delete", chain("/api/v1/admin/delete", apiKeyMiddleware("admin", "delete"), DeleteKeyHandler))

	return loggingMiddleware(corsMiddleware(mux))
}

func bastilleRoutes() map[string]HandlerFunc {
	// Branchy commands with conditional argument grammars remain hand-written
	// (see bastille.go). Everything else is generated from declarativeCommands.
	routes := map[string]HandlerFunc{
		"bootstrap": BastilleBootstrapHandler,
		"config":    BastilleConfigHandler,
		"console":   BastilleConsoleHandler,
		"etcupdate": BastilleEtcupdateHandler,
		"limits":    BastilleLimitsHandler,
		"monitor":   BastilleMonitorHandler,
		"mount":     BastilleMountHandler,
		"network":   BastilleNetworkHandler,
		"rdr":       BastilleRdrHandler,
		"setup":     BastilleSetupHandler,
		"tags":      BastilleTagsHandler,
		"template":  BastilleTemplateHandler,
		"upgrade":   BastilleUpgradeHandler,
		"zfs":       BastilleZfsHandler,
	}

	for command, spec := range declarativeCommands {
		if _, exists := routes[command]; exists {
			// A command must not be defined both ways.
			panic("bastille command declared twice: " + command)
		}
		routes[command] = declarativeHandler(command, spec)
	}

	return routes
}
