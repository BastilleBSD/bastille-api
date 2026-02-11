package api

import (
	"net/http"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

func RocinanteCommand(args ...string) (string, error) {

	logRequest("debug", "RocinanteCommand", nil, args, nil)

	cmd := exec.Command("/usr/local/bin/rocinante", args...)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		logRequest("error", "command failed", nil, args, output)
		return "", err
	}

	return output, nil
}

func RocinanteCommandLive(args ...string) (string, error) {

	logRequest("debug", "RocinanteCommandLive", nil, args, nil)

	ttydArgs := []string{
		"-i", "127.0.0.1",
		"-t", "disableLeaveAlert=true",
		"-b", "/api/v1/rocinante/console/ttyd",
		"-O",
		"-o",
		"--ipv6",
		"-m", "1",
		"-p", "7681",
		"-W",
	}

	cmdArgs := append(ttydArgs, "/usr/local/bin/rocinante")
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("/usr/local/bin/ttyd", cmdArgs...)
	if err := cmd.Start(); err != nil {
		logRequest("error", "ttyd command failed", nil, args, err)
		return "", err
	}

	return "/api/v1/rocinante/console/ttyd", nil
}

func ParseAndRunRocinanteCommand(c *gin.Context, cmdArgs []string) {

	logRequest("debug", "ParseAndRunRocinanteCommand", c, cmdArgs, nil)

	if err := ValidateRocinanteCommandParameters(c, cmdArgs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		logRequest("error", "parameter validation failed", c, cmdArgs, err)
		return
	}

	isLive := strings.Contains(c.FullPath(), "/api/v1/rocinante/live/")
	var result string
	var err error

	if isLive {
		result, err = RocinanteCommandLive(cmdArgs...)
	} else {
		result, err = RocinanteCommand(cmdArgs...)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		logRequest("error", "command failed", c, cmdArgs, err)
		return
	}

	if isLive {
		c.Header("X-TTYD-Url", result)
		c.JSON(http.StatusOK, gin.H{"path": result})
		logRequest("info", "success (live)", c, cmdArgs, result)
	} else {
		c.String(http.StatusOK, result)
		logRequest("info", "success", c, cmdArgs, result)
	}
}

// Rocinante Bootstrap POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param url query string false "url"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/bootstrap [post]
func RocinanteBootstrapHandler(c *gin.Context) {

	logRequest("debug", "RocinanteBootstrapHandler", c, nil, nil)

	cmdArgs := []string{"bootstrap"}
	options := c.Query("options")
	url := c.Query("url")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing url parameter"})
		logRequest("error", "missing url parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, url)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante cmd POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/cmd [post]
func RocinanteCmdHandler(c *gin.Context) {

	logRequest("debug", "RocinanteCmdHandler", c, nil, nil)

	cmdArgs := []string{"cmd"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante limits POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/limits [post]
func RocinanteLimitsHandler(c *gin.Context) {

	logRequest("debug", "RocinanteLimitsHandler", c, nil, nil)

	cmdArgs := []string{"limits"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante list POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/list [post]
func RocinanteListHandler(c *gin.Context) {

	logRequest("debug", "RocinanteListHandler", c, nil, nil)

	cmdArgs := []string{"list"}
	options := c.Query("options")
	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante pkg POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/pkg [post]
func RocinantePkgHandler(c *gin.Context) {

	logRequest("debug", "RocinantePkgHandler", c, nil, nil)

	cmdArgs := []string{"pkg"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante service POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param ARGS query string false "ARGS"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/service [post]
func RocinanteServiceHandler(c *gin.Context) {

	logRequest("debug", "RocinanteServiceHandler", c, nil, nil)

	cmdArgs := []string{"service"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante sysctl POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/sysctl [post]
func RocinanteSysctlHandler(c *gin.Context) {

	logRequest("debug", "RocinanteSysctlHandler", c, nil, nil)

	cmdArgs := []string{"sysctl"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante sysrc POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/sysrc [post]
func RocinanteSysrcHandler(c *gin.Context) {

	logRequest("debug", "RocinanteSysrcHandler", c, nil, nil)

	cmdArgs := []string{"sysrc"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante template POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param action query string false "action"
// @Param template query string false "template"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/template [post]
func RocinanteTemplateHandler(c *gin.Context) {

	logRequest("debug", "RocinanteTemplateHandler", c, nil, nil)

	cmdArgs := []string{"template"}
	options := c.Query("options")
	action := c.Query("action")
	template := c.Query("template")
	args := c.Query("args")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if action != "" {
		cmdArgs = append(cmdArgs, action)
	}
	if template == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing template parameter"})
		logRequest("error", "missing template parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, template)
	if args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante update POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/update [post]
func RocinanteUpdateHandler(c *gin.Context) {

	logRequest("debug", "RocinanteUpdateHandler", c, nil, nil)

	cmdArgs := []string{"update"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante upgrade POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/upgrade [post]
func RocinanteUpgradeHandler(c *gin.Context) {

	logRequest("debug", "RocinanteUpgradeHandler", c, nil, nil)

	cmdArgs := []string{"upgrade"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante verify POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param template query string false "template"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/verify [post]
func RocinanteVerifyHandler(c *gin.Context) {

	logRequest("debug", "RocinanteVerifyHandler", c, nil, nil)

	cmdArgs := []string{"verify"}
	options := c.Query("options")
	template := c.Query("template")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if template == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing template parameter"})
		logRequest("error", "missing template parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, template)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante zfs POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/zfs [post]
func RocinanteZfsHandler(c *gin.Context) {

	logRequest("debug", "RocinanteZfsHandler", c, nil, nil)

	cmdArgs := []string{"zfs"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}

// Rocinante zpool POST
// @Description
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param args query string false "args"
// @Tags rocinante
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/rocinante/zpool [post]
func RocinanteZpoolHandler(c *gin.Context) {

	logRequest("debug", "RocinanteZpoolHandler", c, nil, nil)

	cmdArgs := []string{"zpool"}
	args := c.Query("args")
	if args == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing args parameter"})
		logRequest("error", "missing args parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunRocinanteCommand(c, cmdArgs)
}
