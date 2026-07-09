package api

import (
	"github.com/gin-gonic/gin"

	_ "bastille-api/docs"
	swaggerFiles "github.com/swaggo/files"
	swaggerGin "github.com/swaggo/gin-swagger"
)

func loadRoutes(router *gin.Engine) {

	router.Use(
		loggingMiddleware(),
		CORSMiddleware(),
	)

	router.GET("/swagger/*any", swaggerGin.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")

	bastille := v1.Group("/bastille")
	bastilleLive := v1.Group("/bastille/live")
	bastilleConsole := v1.Group("/bastille/console/ttyd")

	// The console proxy exposes a live, writable root terminal (ttyd -W) into a
	// jail. It MUST be authenticated; an unauthenticated route here is a remote
	// root shell for anyone who can reach the listener.
	bastilleConsole.Any("/*any", apiKeyMiddleware("bastille", "console"), consoleProxy("http://localhost:7681"))

	for path, handler := range bastilleRoutes() {
		bastille.GET("/"+path, apiKeyMiddleware("bastille", path), GetCommandSpec(path))
		bastille.POST("/"+path, apiKeyMiddleware("bastille", path), handler)

		bastilleLive.GET("/"+path, apiKeyMiddleware("bastille", path), GetCommandSpec(path))
		bastilleLive.POST("/"+path, apiKeyMiddleware("bastille", path), handler)
	}

	admin := v1.Group("/admin")
	{
		admin.POST("/add", apiKeyMiddleware("admin", "add"), AddKeyHandler)
		admin.POST("/edit", apiKeyMiddleware("admin", "edit"), EditKeyHandler)
		admin.POST("/delete", apiKeyMiddleware("admin", "delete"), DeleteKeyHandler)
	}
}

func bastilleRoutes() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"bootstrap": BastilleBootstrapHandler,
		"clone":     BastilleCloneHandler,
		"cmd":       BastilleCmdHandler,
		"config":    BastilleConfigHandler,
		"console":   BastilleConsoleHandler,
		"convert":   BastilleConvertHandler,
		"cp":        BastilleCpHandler,
		"create":    BastilleCreateHandler,
		"destroy":   BastilleDestroyHandler,
		"edit":      BastilleEditHandler,
		"etcupdate": BastilleEtcupdateHandler,
		"export":    BastilleExportHandler,
		"htop":      BastilleHtopHandler,
		"import":    BastilleImportHandler,
		"jcp":       BastilleJcpHandler,
		"limits":    BastilleLimitsHandler,
		"list":      BastilleListHandler,
		"migrate":   BastilleMigrateHandler,
		"monitor":   BastilleMonitorHandler,
		"mount":     BastilleMountHandler,
		"network":   BastilleNetworkHandler,
		"pkg":       BastillePkgHandler,
		"rcp":       BastilleRcpHandler,
		"rdr":       BastilleRdrHandler,
		"rename":    BastilleRenameHandler,
		"restart":   BastilleRestartHandler,
		"service":   BastilleServiceHandler,
		"setup":     BastilleSetupHandler,
		"start":     BastilleStartHandler,
		"stop":      BastilleStopHandler,
		"sysrc":     BastilleSysrcHandler,
		"tags":      BastilleTagsHandler,
		"template":  BastilleTemplateHandler,
		"top":       BastilleTopHandler,
		"umount":    BastilleUmountHandler,
		"update":    BastilleUpdateHandler,
		"upgrade":   BastilleUpgradeHandler,
		"verify":    BastilleVerifyHandler,
		"zfs":       BastilleZfsHandler,
	}
}