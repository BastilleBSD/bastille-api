package api

import (
	"net/http"
	"os/exec"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

var bastilleLock sync.Mutex

func BastilleCommand(args ...string) (string, error) {

	logRequest("debug", "BastilleCommand", nil, args, nil)

	bastilleLock.Lock()
	defer bastilleLock.Unlock()

	cmd := exec.Command("/usr/local/bin/bastille", args...)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		logRequest("error", "command failed", nil, args, output)
		return "", err
	}

	return output, nil
}

func BastilleCommandLive(args ...string) (string, error) {

	logRequest("debug", "BastilleCommandLive", nil, args, nil)

	bastilleLock.Lock()
	defer bastilleLock.Unlock()

	ttydArgs := []string{
		"-i", "127.0.0.1",
		"-t", "disableLeaveAlert=true",
		"-b", "/api/v1/bastille/console/ttyd",
		"-o",
		"-O",
		"--ipv6",
		"-m", "1",
		"-p", "7681",
		"-W",
	}

	cmdArgs := append(ttydArgs, "/usr/local/bin/bastille")
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command("/usr/local/bin/ttyd", cmdArgs...)
	if err := cmd.Start(); err != nil {
		logRequest("error", "ttyd command failed", nil, args, err)
		return "", err
	}

	return "/api/v1/bastille/console/ttyd", nil
}

func ParseAndRunBastilleCommand(c *gin.Context, cmdArgs []string) {

	logRequest("debug", "ParseAndRunBastilleCommand", c, cmdArgs, nil)

	if err := ValidateBastilleCommandParameters(c, cmdArgs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		logRequest("error", "parameter validation failed", c, cmdArgs, err)
		return
	}

	isLive := strings.Contains(c.FullPath(), "/api/v1/bastille/live/")
	var result string
	var err error

	if isLive {
		result, err = BastilleCommandLive(cmdArgs...)
	} else {
		result, err = BastilleCommand(cmdArgs...)
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

// Bastille bootstrap POST
// @Description Bootstrap a release or template(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param arch query string false "arch"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/bootstrap [post]
func BastilleBootstrapHandler(c *gin.Context) {

	logRequest("debug", "BastilleBootstrapHandler", c, nil, nil)

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

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille clone POST
// @Description Clone an existing jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param new_name query string false "new_name"
// @Param ip query string false "ip"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/clone [post]
func BastilleCloneHandler(c *gin.Context) {

	logRequest("debug", "BastilleCloneHandler", c, nil, nil)

	cmdArgs := []string{"clone"}
	options := c.Query("options")
	target := c.Query("target")
	new_name := c.Query("new_name")
	ip := c.Query("ip")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if new_name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing new_name parameter"})
		logRequest("error", "missing new_name parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, new_name)

	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing ip parameter"})
		logRequest("error", "missing ip parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, ip)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille cmd POST
// @Description Execute command(s) inside jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param command query string false "command"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/cmd [post]
func BastilleCmdHandler(c *gin.Context) {

	logRequest("debug", "BastilleCmdHandler", c, nil, nil)

	cmdArgs := []string{"cmd"}
	options := c.Query("options")
	target := c.Query("target")
	command := c.Query("command")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if command == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing command parameter"})
		logRequest("error", "missing command parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(command)...)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille config POST
// @Description Get, set, add or remove properties from jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param property query string false "property"
// @Param value query string false "value"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/config [post]
func BastilleConfigHandler(c *gin.Context) {

	logRequest("debug", "BastilleConfigHandler", c, nil, nil)

	cmdArgs := []string{"config"}
	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	property := c.Query("property")
	value := c.Query("value")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if action != "set" && action != "add" && action != "get" && action != "remove" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown action parameter"})
		logRequest("error", "unknown action parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, action)

	if property == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing property parameter"})
		logRequest("error", "missing property parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, property)

	if value != "" {
		cmdArgs = append(cmdArgs, value)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille console POST
// @Description Console into a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param user query string false "user"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/console [post]
func BastilleConsoleHandler(c *gin.Context) {

	logRequest("debug", "BastilleConsoleHandler", c, nil, nil)

	cmdArgs := []string{"console"}
	options := c.Query("options")
	target := c.Query("target")
	user := c.Query("user")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if user != "" {
		cmdArgs = append(cmdArgs, user)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille convert POST
// @Description Convert a jail from thin to thick; convert a jail to a custom release.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param release query string false "release"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/convert [post]
func BastilleConvertHandler(c *gin.Context) {

	logRequest("debug", "BastilleConvertHandler", c, nil, nil)

	cmdArgs := []string{"convert"}
	options := c.Query("options")
	target := c.Query("target")
	release := c.Query("release")

	if options != "" {
		options = options + " -ay"
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	} else {
		cmdArgs = append(cmdArgs, "-ay")
	}

	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if release != "" {
		cmdArgs = append(cmdArgs, release)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille cp POST
// @Description Copy file(s)/directorie(s) from host to jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param host_path query string false "host_path"
// @Param jail_path query string false "jail_path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/cp [post]
func BastilleCpHandler(c *gin.Context) {

	logRequest("debug", "BastilleCpHandler", c, nil, nil)

	cmdArgs := []string{"cp"}
	options := c.Query("options")
	target := c.Query("target")
	host_path := c.Query("host_path")
	jail_path := c.Query("jail_path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing target parameter"})
		logRequest("error", "missing target parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if host_path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing host_path parameter"})
		logRequest("error", "missing host_path parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, host_path)

	if jail_path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing jail_path parameter"})
		logRequest("error", "missing jail_path parameter", c, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, jail_path)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille create POST
// @Description Create a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param name query string false "name"
// @Param release query string false "release"
// @Param ip query string false "ip"
// @Param iface query string false "iface"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/create [post]
func BastilleCreateHandler(c *gin.Context) {

	logRequest("debug", "BastilleCreateHandler", nil, nil, nil)

	cmdArgs := []string{"create"}

	options := c.Query("options")
	name := c.Query("name")
	release := c.Query("release")
	ip := c.Query("ip")
	iface := c.Query("iface")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, "Missing name parameter.")
		logRequest("error", "missing name parameter", nil, cmdArgs, nil)
		return
	}
	if release == "" {
		c.JSON(http.StatusBadRequest, "Missing release parameter")
		logRequest("error", "missing release parameter", nil, cmdArgs, nil)
		return
	}
	if ip == "" {
		c.JSON(http.StatusBadRequest, "Missing ip parameter")
		logRequest("error", "missing ip parameter", nil, cmdArgs, nil)
		return
	}

	cmdArgs = append(cmdArgs, name, release, ip)

	if iface != "" {
		cmdArgs = append(cmdArgs, iface)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille destroy POST
// @Description Destroy jail(s) or release(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/destroy [post]
func BastilleDestroyHandler(c *gin.Context) {

	logRequest("debug", "BastilleDestroyHandler", nil, nil, nil)

	cmdArgs := []string{"destroy"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille edit POST
// @Description Edit jail configuration files (advanced).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param file query string false "file"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/edit [post]
func BastilleEditHandler(c *gin.Context) {

	logRequest("debug", "BastilleEditHandler", nil, nil, nil)

	cmdArgs := []string{"edit"}

	options := c.Query("options")
	target := c.Query("target")
	file := c.Query("file")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if file != "" {
		cmdArgs = append(cmdArgs, file)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille etcupdate POST
// @Description Update /etc for jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param release query string false "release"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/etcupdate [post]
func BastilleEtcupdateHandler(c *gin.Context) {

	logRequest("debug", "BastilleEtcupdateHandler", nil, nil, nil)

	cmdArgs := []string{"etcupdate"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	release := c.Query("release")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}

	if action == "bootstrap" {
		if release == "" {
			c.JSON(http.StatusBadRequest, "Missing release parameter")
			logRequest("error", "missing release parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, action, release)
	} else {
		if target == "" {
			c.JSON(http.StatusBadRequest, "Missing target parameter")
			logRequest("error", "missing target parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, target)
		if action == "update" {
			if release == "" {
				c.JSON(http.StatusBadRequest, "Missing release parameter")
				logRequest("error", "missing release parameter", nil, cmdArgs, nil)
				return
			}
			cmdArgs = append(cmdArgs, release)
		} else {
			if action == "" {
				c.JSON(http.StatusBadRequest, "Missing action parameter")
				logRequest("error", "missing action parameter", nil, cmdArgs, nil)
				return
			}
			cmdArgs = append(cmdArgs, action)
		}
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille export POST
// @Description Export a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param path query string false "path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/export [post]
func BastilleExportHandler(c *gin.Context) {

	logRequest("debug", "BastilleExportHandler", nil, nil, nil)

	cmdArgs := []string{"export"}

	options := c.Query("options")
	target := c.Query("target")
	path := c.Query("path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if path != "" {
		cmdArgs = append(cmdArgs, path)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille htop POST
// @Description Interactive process viewer (requires htop).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/htop [post]
func BastilleHtopHandler(c *gin.Context) {

	logRequest("debug", "BastilleHtopHandler", nil, nil, nil)

	cmdArgs := []string{"htop"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille import POST
// @Description Import a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param file query string false "file"
// @Param release query string false "release"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/import [post]
func BastilleImportHandler(c *gin.Context) {

	logRequest("debug", "BastilleImportHandler", nil, nil, nil)

	cmdArgs := []string{"import"}

	options := c.Query("options")
	file := c.Query("file")
	release := c.Query("release")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if file == "" {
		c.JSON(http.StatusBadRequest, "Missing file parameter")
		logRequest("error", "missing file parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, file)
	if release != "" {
		cmdArgs = append(cmdArgs, release)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille jcp POST
// @Description Copy file(s)/directorie(s) from jail to jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param source_jail query string false "source_jail"
// @Param source_path query string false "source_path"
// @Param destination_jail query string false "destination_jail"
// @Param destination_path query string false "destination_path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/jcp [post]
func BastilleJcpHandler(c *gin.Context) {

	logRequest("debug", "BastilleJcpHandler", nil, nil, nil)

	cmdArgs := []string{"jcp"}

	options := c.Query("options")
	source_jail := c.Query("source_jail")
	source_path := c.Query("source_path")
	destination_jail := c.Query("destination_jail")
	destination_path := c.Query("destination_path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if source_jail == "" {
		c.JSON(http.StatusBadRequest, "Missing source_jail parameter")
		logRequest("error", "missing source_jail parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, source_jail)
	if source_path == "" {
		c.JSON(http.StatusBadRequest, "Missing source_path parameter")
		logRequest("error", "missing source_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, source_path)
	if destination_jail == "" {
		c.JSON(http.StatusBadRequest, "Missing destination_jail parameter")
		logRequest("error", "missing destination_jail parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, destination_jail)
	if destination_path == "" {
		c.JSON(http.StatusBadRequest, "Missing destination_path parameter")
		logRequest("error", "missing destination_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, destination_path)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille limits POST
// @Description Apply resources limits to jail(s). See rctl(8) and cpuset(1).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param args query string false "args"
// @Param option query string false "option"
// @Param value query string false "value"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/limits [post]
func BastilleLimitsHandler(c *gin.Context) {

	logRequest("debug", "BastilleLimitsHandler", nil, nil, nil)

	cmdArgs := []string{"limits"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	args := c.Query("args")
	option := c.Query("option")
	value := c.Query("value")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if action == "" {
		c.JSON(http.StatusBadRequest, "Missing action parameter")
		logRequest("error", "missing action parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, action)

	switch action {
	case "add":
		if option == "" {
			c.JSON(http.StatusBadRequest, "Missing option parameter")
			logRequest("error", "missing option parameter", nil, cmdArgs, nil)
			return
		}
		if value == "" {
			c.JSON(http.StatusBadRequest, "Missing value parameter")
			logRequest("error", "missing value parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, option, value)
	case "remove":
		if option == "" {
			c.JSON(http.StatusBadRequest, "Missing option parameter")
			logRequest("error", "missing option parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, option)
	case "clear", "reset", "stats":
		// just append the action
	case "list", "show":
		if args == "active" {
			cmdArgs = append(cmdArgs, action, args)
		} else {
			cmdArgs = append(cmdArgs, action)
		}
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille list POST
// @Description List jails, releases, templates and more...
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param item query string false "item"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/list [post]
func BastilleListHandler(c *gin.Context) {

	logRequest("debug", "BastilleListHandler", nil, nil, nil)

	cmdArgs := []string{"list"}

	options := c.Query("options")
	item := c.Query("item")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if item != "" {
		cmdArgs = append(cmdArgs, item)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille migrate POST
// @Description Migrate jail(s) to a remote system.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param destination query string false "destination"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/migrate [post]
func BastilleMigrateHandler(c *gin.Context) {

	logRequest("debug", "BastilleMigrateHandler", nil, nil, nil)

	cmdArgs := []string{"migrate"}

	options := c.Query("options")
	target := c.Query("target")
	destination := c.Query("destination")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if destination == "" {
		c.JSON(http.StatusBadRequest, "Missing destination parameter")
		logRequest("error", "missing destination parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, destination)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille monitor POST
// @Description Monitor and attempt to restart jail service(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param service query string false "service"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/monitor [post]
func BastilleMonitorHandler(c *gin.Context) {

	logRequest("debug", "BastilleMonitorHandler", nil, nil, nil)

	cmdArgs := []string{"monitor"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	service := c.Query("service")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}

	if action == "enable" || action == "disable" || action == "status" {
		cmdArgs = append(cmdArgs, action)
	} else if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	} else {
		cmdArgs = append(cmdArgs, target)
	}

	if action == "add" || action == "delete" {
		if service == "" {
			c.JSON(http.StatusBadRequest, "Missing service parameter")
			logRequest("error", "missing service parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, action, service)
	} else if action == "list" {
		cmdArgs = append(cmdArgs, action)
		if service != "" {
			cmdArgs = append(cmdArgs, service)
		}
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille mount POST
// @Description Mount file(s)/directorie(s) inside jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param host_path query string false "host_path"
// @Param jail_path query string false "jail_path"
// @Param fs_type query string false "fs_type"
// @Param fs_options query string false "fs_options"
// @Param dump query string false "dump"
// @Param pass_number query string false "pass_number"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/mount [post]
func BastilleMountHandler(c *gin.Context) {

	logRequest("debug", "BastilleMountHandler", nil, nil, nil)

	cmdArgs := []string{"mount"}

	options := c.Query("options")
	target := c.Query("target")
	host_path := c.Query("host_path")
	jail_path := c.Query("jail_path")
	fs_type := c.Query("fs_type")
	fs_options := c.Query("fs_options")
	dump := c.Query("dump")
	pass_number := c.Query("pass_number")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if host_path == "" {
		c.JSON(http.StatusBadRequest, "Missing host_path parameter")
		logRequest("error", "missing host_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, host_path)
	if jail_path == "" {
		c.JSON(http.StatusBadRequest, "Missing jail_path parameter")
		logRequest("error", "missing jail_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, jail_path)

	if fs_type != "" || fs_options != "" || dump != "" || pass_number != "" {
		if fs_type == "" || fs_options == "" || dump == "" || pass_number == "" {
			c.JSON(http.StatusBadRequest, "Missing mount parameter(s)")
			logRequest("error", "missing mount parameter(s)", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, fs_type, fs_options, dump, pass_number)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille network POST
// @Description Add or remove interface(s) from jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param iface query string false "iface"
// @Param ip query string false "ip"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/network [post]
func BastilleNetworkHandler(c *gin.Context) {

	logRequest("debug", "BastilleNetworkHandler", nil, nil, nil)

	cmdArgs := []string{"network"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	iface := c.Query("iface")
	ip := c.Query("ip")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if action == "add" {
		if iface == "" {
			c.JSON(http.StatusBadRequest, "Missing iface parameter")
			logRequest("error", "missing iface parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, action, iface)
		if ip != "" {
			cmdArgs = append(cmdArgs, ip)
		}
	} else if action == "remove" {
		if iface == "" {
			c.JSON(http.StatusBadRequest, "Missing iface parameter")
			logRequest("error", "missing iface parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, action, iface)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille pkg POST
// @Description Manage packages inside jail(s). See pkg(8).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param args query string false "args"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/pkg [post]
func BastillePkgHandler(c *gin.Context) {

	logRequest("debug", "BastillePkgHandler", nil, nil, nil)

	cmdArgs := []string{"pkg"}

	options := c.Query("options")
	target := c.Query("target")
	args := c.Query("args")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if args == "" {
		c.JSON(http.StatusBadRequest, "Missing args parameter")
		logRequest("error", "missing args parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille rcp POST
// @Description Copy file(s)/directorie(s) from jail to host.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param jail_path query string false "jail_path"
// @Param host_path query string false "host_path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/rcp [post]
func BastilleRcpHandler(c *gin.Context) {

	logRequest("debug", "BastilleRcpHandler", nil, nil, nil)

	cmdArgs := []string{"rcp"}

	options := c.Query("options")
	target := c.Query("target")
	jailPath := c.Query("jail_path")
	hostPath := c.Query("host_path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if jailPath == "" {
		c.JSON(http.StatusBadRequest, "Missing jail_path parameter")
		logRequest("error", "missing jail_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, jailPath)

	if hostPath == "" {
		c.JSON(http.StatusBadRequest, "Missing host_path parameter")
		logRequest("error", "missing host_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, hostPath)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille rdr POST
// @Description Redirect host port to jail port.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param protocol query string false "protocol"
// @Param host_port query string false "host_port"
// @Param jail_port query string false "jail_port"
// @Param log_options query string false "log_options"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/rdr [post]
func BastilleRdrHandler(c *gin.Context) {

	logRequest("debug", "BastilleRdrHandler", nil, nil, nil)

	cmdArgs := []string{"rdr"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	protocol := c.Query("protocol")
	hostPort := c.Query("host_port")
	jailPort := c.Query("jail_port")
	logOptions := c.Query("log_options")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if action == "clear" || action == "reset" || action == "list" {
		cmdArgs = append(cmdArgs, action)
	} else {
		if protocol == "" {
			c.JSON(http.StatusBadRequest, "Missing protocol parameter")
			logRequest("error", "missing protocol parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, protocol)

		if hostPort == "" {
			c.JSON(http.StatusBadRequest, "Missing host_port parameter")
			logRequest("error", "missing host_port parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, hostPort)

		if jailPort == "" {
			c.JSON(http.StatusBadRequest, "Missing jail_port parameter")
			logRequest("error", "missing jail_port parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, jailPort)

		if action == "log" {
			if logOptions == "" {
				c.JSON(http.StatusBadRequest, "Missing log_options parameter")
				logRequest("error", "missing log_options parameter", nil, cmdArgs, nil)
				return
			}
			cmdArgs = append(cmdArgs, action, logOptions)
		}
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille rename POST
// @Description Rename a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param new_name query string false "new_name"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/rename [post]
func BastilleRenameHandler(c *gin.Context) {

	logRequest("debug", "BastilleRenameHandler", nil, nil, nil)

	cmdArgs := []string{"rename"}

	options := c.Query("options")
	target := c.Query("target")
	newName := c.Query("new_name")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	if newName == "" {
		c.JSON(http.StatusBadRequest, "Missing new_name parameter")
		logRequest("error", "missing new_name parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target, newName)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille restart POST
// @Description Restart a jail.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/restart [post]
func BastilleRestartHandler(c *gin.Context) {

	logRequest("debug", "BastilleRestartHandler", nil, nil, nil)

	cmdArgs := []string{"restart"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille service POST
// @Description Manage services within jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param service query string false "service"
// @Param args query string false "args"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/service [post]
func BastilleServiceHandler(c *gin.Context) {

	logRequest("debug", "BastilleServiceHandler", nil, nil, nil)

	cmdArgs := []string{"service"}

	options := c.Query("options")
	target := c.Query("target")
	service := c.Query("service")
	args := c.Query("args")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if service == "" {
		c.JSON(http.StatusBadRequest, "Missing service parameter")
		logRequest("error", "missing service parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, service)
	if args == "" {
		c.JSON(http.StatusBadRequest, "Missing args parameter")
		logRequest("error", "missing args parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille setup POST
// @Description Auto-configure network, firewall, storage and more...
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param item query string false "item"
// @Param args query string false "args"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/setup [post]
func BastilleSetupHandler(c *gin.Context) {

	logRequest("debug", "BastilleSetupHandler", nil, nil, nil)

	cmdArgs := []string{"setup"}

	options := c.Query("options")
	item := c.Query("item")
	args := c.Query("args")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if item != "" {
		cmdArgs = append(cmdArgs, item)
	}
	if args != "" {
		cmdArgs = append(cmdArgs, args)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille start POST
// @Description Start stopped jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/start [post]
func BastilleStartHandler(c *gin.Context) {

	logRequest("debug", "BastilleStartHandler", nil, nil, nil)

	cmdArgs := []string{"start"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille stop POST
// @Description Stop running jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/stop [post]
func BastilleStopHandler(c *gin.Context) {

	logRequest("debug", "BastilleStopHandler", nil, nil, nil)

	cmdArgs := []string{"stop"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille sysrc POST
// @Description Edit rc files inside jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param args query string false "args"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/sysrc [post]
func BastilleSysrcHandler(c *gin.Context) {

	logRequest("debug", "BastilleSysrcHandler", nil, nil, nil)

	cmdArgs := []string{"sysrc"}

	options := c.Query("options")
	target := c.Query("target")
	args := c.Query("args")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if args == "" {
		c.JSON(http.StatusBadRequest, "Missing args parameter")
		logRequest("error", "missing args parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, strings.Fields(args)...)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille tags POST
// @Description Add or remove tags to jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param tags query string false "tags"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/tags [post]
func BastilleTagsHandler(c *gin.Context) {

	logRequest("debug", "BastilleTagsHandler", nil, nil, nil)

	cmdArgs := []string{"tags"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	tags := c.Query("tags")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if action == "add" || action == "delete" {
		if tags == "" {
			c.JSON(http.StatusBadRequest, "Missing tags parameter")
			logRequest("error", "missing tags parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, action, tags)
	} else if action == "list" {
		cmdArgs = append(cmdArgs, action)
		if tags != "" {
			cmdArgs = append(cmdArgs, tags)
		}
	} else {
		c.JSON(http.StatusBadRequest, "Invalid action parameter")
		logRequest("error", "invalid action parameter", nil, cmdArgs, nil)
		return
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille template POST
// @Description Apply templates to jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param template query string false "template"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/template [post]
func BastilleTemplateHandler(c *gin.Context) {

	logRequest("debug", "BastilleTemplateHandler", nil, nil, nil)

	cmdArgs := []string{"template"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	template := c.Query("template")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}

	if action == "convert" {
		cmdArgs = append(cmdArgs, action)
		if template == "" {
			c.JSON(http.StatusBadRequest, "Missing template parameter")
			logRequest("error", "missing template parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, template)
	} else {
		if target == "" {
			c.JSON(http.StatusBadRequest, "Missing target parameter")
			logRequest("error", "missing target parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, target)
		if template == "" {
			c.JSON(http.StatusBadRequest, "Missing template parameter")
			logRequest("error", "missing template parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, template)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille top POST
// @Description Process viewer. See top(1).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/top [post]
func BastilleTopHandler(c *gin.Context) {

	logRequest("debug", "BastilleTopHandler", nil, nil, nil)

	cmdArgs := []string{"top"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille umount POST
// @Description Unmount file(s)/directorie(s) from jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param jail_path query string false "jail_path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/umount [post]
func BastilleUmountHandler(c *gin.Context) {

	logRequest("debug", "BastilleUmountHandler", nil, nil, nil)

	cmdArgs := []string{"umount"}

	options := c.Query("options")
	target := c.Query("target")
	jailPath := c.Query("jail_path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)
	if jailPath == "" {
		c.JSON(http.StatusBadRequest, "Missing jail_path parameter")
		logRequest("error", "missing jail_path parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, jailPath)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille update POST
// @Description Update a jail or release.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/update [post]
func BastilleUpdateHandler(c *gin.Context) {

	logRequest("debug", "BastilleUpdateHandler", nil, nil, nil)

	cmdArgs := []string{"update"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille upgrade POST
// @Description Upgrade a jail to new release.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param new_release query string false "new_release"
// @Param action query string false "action"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/upgrade [post]
func BastilleUpgradeHandler(c *gin.Context) {

	logRequest("debug", "BastilleUpgradeHandler", nil, nil, nil)

	cmdArgs := []string{"upgrade"}

	options := c.Query("options")
	target := c.Query("target")
	newRelease := c.Query("new_release")
	action := c.Query("action")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	if action == "install" {
		cmdArgs = append(cmdArgs, action)
	} else {
		if newRelease == "" {
			c.JSON(http.StatusBadRequest, "Missing new_release parameter")
			logRequest("error", "missing new_release parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, newRelease)
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille verify POST
// @Description Compare release against a "known good" index.
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/verify [post]
func BastilleVerifyHandler(c *gin.Context) {

	logRequest("debug", "BastilleVerifyHandler", nil, nil, nil)

	cmdArgs := []string{"verify"}

	options := c.Query("options")
	target := c.Query("target")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// Bastille zfs POST
// @Description Manage ZFS options/attributes for jail(s).
// @Param Authorization header string true "Authentication token (e.g., Bearer <token>)"
// @Param options query string false "options"
// @Param target query string false "target"
// @Param action query string false "action"
// @Param tag query string false "tag"
// @Param key_value query string false "key_value"
// @Param dataset query string false "dataset"
// @Param jail_path query string false "jail_path"
// @Tags bastille
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string
// @Router /api/v1/bastille/zfs [post]
func BastilleZfsHandler(c *gin.Context) {

	logRequest("debug", "BastilleZfsHandler", nil, nil, nil)

	cmdArgs := []string{"zfs"}

	options := c.Query("options")
	target := c.Query("target")
	action := c.Query("action")
	tag := c.Query("tag")
	keyValue := c.Query("key_value")
	dataset := c.Query("dataset")
	jailPath := c.Query("jail_path")

	if options != "" {
		cmdArgs = append(cmdArgs, strings.Fields(options)...)
	}
	if target == "" {
		c.JSON(http.StatusBadRequest, "Missing target parameter")
		logRequest("error", "missing target parameter", nil, cmdArgs, nil)
		return
	}
	cmdArgs = append(cmdArgs, target)

	switch action {
	case "snapshot", "destroy", "rollback":
		cmdArgs = append(cmdArgs, action)
		if tag != "" {
			cmdArgs = append(cmdArgs, tag)
		}
	case "df", "usage":
		cmdArgs = append(cmdArgs, action)
	case "get", "set":
		cmdArgs = append(cmdArgs, action)
		if keyValue == "" {
			c.JSON(http.StatusBadRequest, "Missing key_value parameter")
			logRequest("error", "missing key_value parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, keyValue)
	case "jail":
		cmdArgs = append(cmdArgs, action)
		if dataset == "" {
			c.JSON(http.StatusBadRequest, "Missing dataset parameter")
			logRequest("error", "missing dataset parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, dataset)
		if jailPath == "" {
			c.JSON(http.StatusBadRequest, "Missing jail_path parameter")
			logRequest("error", "missing jail_path parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, jailPath)
	case "unjail":
		cmdArgs = append(cmdArgs, action)
		if jailPath == "" {
			c.JSON(http.StatusBadRequest, "Missing jail_path parameter")
			logRequest("error", "missing jail_path parameter", nil, cmdArgs, nil)
			return
		}
		cmdArgs = append(cmdArgs, jailPath)
	default:
		c.JSON(http.StatusBadRequest, "Invalid action parameter")
		logRequest("error", "invalid action parameter", nil, cmdArgs, nil)
		return
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}
