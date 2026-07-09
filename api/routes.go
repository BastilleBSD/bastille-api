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
	// Only `limits` keeps a bespoke handler: its original list/show path
	// double-appends the action token (a suspected bug preserved verbatim, not
	// encoded into the grammar). Every other command is generated from a spec —
	// flat ones from declarativeCommands, branchy ones from actionCommands.
	routes := map[string]HandlerFunc{
		"limits": BastilleLimitsHandler,
	}

	register := func(command string, handler HandlerFunc) {
		if _, exists := routes[command]; exists {
			// A command must not be defined more than once.
			panic("bastille command declared twice: " + command)
		}
		routes[command] = handler
	}

	for command, spec := range declarativeCommands {
		register(command, declarativeHandler(command, spec))
	}
	for command, spec := range actionCommands {
		register(command, actionHandler(command, spec))
	}

	return routes
}
