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

	// use the following to require auth
	// bastilleConsole.Any("/*any", apiKeyMiddleware("bastille", "console"), consoleProxy("http://localhost:7681"))
	bastilleConsole.Any("/*any", consoleProxy("http://localhost:7681"))

	for path, handler := range bastilleRoutes() {
		bastille.GET("/"+path, apiKeyMiddleware("bastille", path), GetCommandSpec(path, "bastille"))
		bastille.POST("/"+path, apiKeyMiddleware("bastille", path), handler)

		bastilleLive.GET("/"+path, apiKeyMiddleware("bastille", path), GetCommandSpec(path, "bastille"))
		bastilleLive.POST("/"+path, apiKeyMiddleware("bastille", path), handler)
	}

	rocinante := v1.Group("/rocinante")
	rocinanteLive := v1.Group("/rocinante/live")
	rocinanteConsole := v1.Group("/rocinante/console/ttyd")

	// use the following to require auth
	// rocinanteConsole.Any("/*any", apiKeyMiddleware("bastille", "console"), consoleProxy("http://localhost:7681"))
	rocinanteConsole.Any("/*any", consoleProxy("http://localhost:7681"))

	for path, handler := range rocinanteRoutes() {
		rocinante.GET("/"+path, apiKeyMiddleware("rocinante", path), GetCommandSpec(path, "rocinante"))
		rocinante.POST("/"+path, apiKeyMiddleware("rocinante", path), handler)

		rocinanteLive.GET("/"+path, apiKeyMiddleware("rocinante", path), GetCommandSpec(path, "rocinante"))
		rocinanteLive.POST("/"+path, apiKeyMiddleware("rocinante", path), handler)
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

func rocinanteRoutes() map[string]gin.HandlerFunc {
	return map[string]gin.HandlerFunc{
		"bootstrap": RocinanteBootstrapHandler,
		"cmd":       RocinanteCmdHandler,
		"limits":    RocinanteLimitsHandler,
		"list":      RocinanteListHandler,
		"pkg":       RocinantePkgHandler,
		"service":   RocinanteServiceHandler,
		"sysctl":    RocinanteSysctlHandler,
		"sysrc":     RocinanteSysrcHandler,
		"template":  RocinanteTemplateHandler,
		"update":    RocinanteUpdateHandler,
		"upgrade":   RocinanteUpgradeHandler,
		"verify":    RocinanteVerifyHandler,
		"zfs":       RocinanteZfsHandler,
		"zpool":     RocinanteZpoolHandler,
	}
}
