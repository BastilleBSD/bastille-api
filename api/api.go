package api

import (
	"embed"
	"fmt"
	"net"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed bastille.json
var specSheets embed.FS

// @title BastilleBSD-API
// @version 0.0.1
// @description API interface for Bastille and Rocinante
// @termsOfService http://swagger.io/terms/
// @license.name BSD-3-Clause
// @license.url https://opensource.org/license/bsd-3-clause
// @BasePath /
func Start(config string, port string) {

	var bindAddr string

	if config != "" {
		configFile = config
	}

	_, err := loadConfig()
	if err != nil {
		logRequest("error", "Failed to load config", nil, nil, err.Error())
		os.Exit(1)
	}

	if hasDefaultCredential(cfg) {
		logRequest("error", "Refusing to start: the shipped default API key is present in "+
			"the config. Remove it and create your own key before starting.", nil, nil, nil)
		os.Exit(1)
	}

	if len(cfg.APIKeys) == 0 {
		if err := bootstrapFirstKey(); err != nil {
			logRequest("error", "Failed to generate bootstrap API key", nil, nil, err.Error())
			os.Exit(1)
		}
	}

	if port != "" {
		Port = port
	} else if cfg != nil && cfg.Port != "" {
		Port = cfg.Port
	} else {
		Port = "8888"
	}

	bindAddr, Host = resolveBindAddr(Host)

	if !isLoopbackBind(bindAddr) {
		logRequest("info", fmt.Sprintf("WARNING: binding to non-loopback address %q. "+
			"TLS and authentication must be handled directly (e.g. a reverse proxy will NOT protect this listener).", bindAddr), nil, nil, nil)
	}

	addr := fmt.Sprintf("%s:%s", bindAddr, Port)

	loadBastilleSpec()

	router := gin.New()
	loadRoutes(router)

	logRequest("info", fmt.Sprintf("Starting BastilleBSD API server on %s", addr), nil ,nil, nil)
	if err := router.Run(addr); err != nil {
		logRequest("error", "Server failed to start", nil, nil, err.Error())
		os.Exit(1)
	}
}

// resolveBindAddr maps the configured host to the address the server should
// actually bind, plus a display host for logging.
//
// Defaulting is loopback-first: an empty, "localhost" or "127.0.0.1" host binds
// only the loopback interface so the API stays reachable exclusively from the
// local machine (the recommended reverse-proxy topology). Exposing the API on
// all interfaces requires the operator to explicitly set host to "0.0.0.0", and
// any other value binds that specific address.
func resolveBindAddr(host string) (bindAddr string, displayHost string) {
	switch host {
	case "", "localhost", "127.0.0.1":
		return "127.0.0.1", "localhost"
	case "0.0.0.0":
		return "0.0.0.0", "0.0.0.0"
	default:
		return host, host
	}
}

// isLoopbackBind reports whether the given bind address is restricted to the
// local machine.
func isLoopbackBind(bindAddr string) bool {
	if bindAddr == "localhost" {
		return true
	}
	ip := net.ParseIP(bindAddr)
	return ip != nil && ip.IsLoopback()
}
