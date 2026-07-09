package api

import (
	"net/http"
	"os/exec"
	"strings"
	"sync"
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

func ParseAndRunBastilleCommand(c *Ctx, cmdArgs []string) {

	logRequest("debug", "ParseAndRunBastilleCommand", c, cmdArgs, nil)

	if err := ValidateBastilleCommandParameters(c, cmdArgs); err != nil {
		c.JSON(http.StatusBadRequest, H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, H{"error": err.Error()})
		logRequest("error", "command failed", c, cmdArgs, err)
		return
	}

	if isLive {
		c.Header("X-TTYD-Url", result)
		c.JSON(http.StatusOK, H{"path": result})
		logRequest("info", "success (live)", c, cmdArgs, result)
	} else {
		c.String(http.StatusOK, result)
		logRequest("info", "success", c, cmdArgs, result)
	}
}

// Bastille limits POST
// @Description Apply resources limits to jail(s). See rctl(8) and cpuset(1).
// @Param Authorization-ID header string true "API Key ID/Name (eg: bastille-key)"
// @Param Authorization header string true "Authorization Header (eg: Bearer <api-token>)"
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
func BastilleLimitsHandler(c *Ctx) {

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
