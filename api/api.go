package api

import (
	"embed"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

//go:embed bastille.json rocinante.json
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

	if port != "" {
		Port = port
	} else if cfg != nil && cfg.Port != "" {
		Port = cfg.Port
	} else {
		Port = "8888"
	}

	if Host == "0.0.0.0" || Host == "localhost" || Host == "" {
		bindAddr = "0.0.0.0"
		Host = "localhost"
	} else {
		bindAddr = Host
	}

	addr := fmt.Sprintf("%s:%s", bindAddr, Port)

	loadBastilleSpec()
	loadRocinanteSpec()

	router := gin.New()
	loadRoutes(router)

	logRequest("info", fmt.Sprintf("Starting BastilleBSD API server on %s", addr), nil ,nil, nil)
	if err := router.Run(addr); err != nil {
		logRequest("error", "Server failed to start", nil, nil, err.Error())
		os.Exit(1)
	}
}
