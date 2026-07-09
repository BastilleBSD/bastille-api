package api

import (
	"net/http"
	"net/url"
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

	logRequest("debug", "BastilleLimitsHandler", c, nil, nil)

	cmdArgs, badReq := buildLimitsArgs(c.Request.URL.Query())
	if badReq != "" {
		c.JSON(http.StatusBadRequest, H{"error": badReq})
		logRequest("error", badReq, c, nil, nil)
		return
	}

	ParseAndRunBastilleCommand(c, cmdArgs)
}

// buildLimitsArgs constructs argv for the limits command. limits keeps a bespoke
// builder because its list/show action takes an optional literal "active" that
// the flat/action grammars don't model.
//
// CLI grammar (bastille limits -h):
//
//	TARGET add OPTION VALUE
//	TARGET remove OPTION
//	TARGET clear|reset|stats
//	TARGET list|show [active]
func buildLimitsArgs(q url.Values) (args []string, badReq string) {
	args = []string{"limits"}

	if options := q.Get("options"); options != "" {
		args = append(args, strings.Fields(options)...)
	}

	target := q.Get("target")
	if target == "" {
		return nil, "Missing target parameter"
	}
	args = append(args, target)

	action := q.Get("action")
	if action == "" {
		return nil, "Missing action parameter"
	}
	args = append(args, action)

	switch action {
	case "add":
		option := q.Get("option")
		value := q.Get("value")
		if option == "" {
			return nil, "Missing option parameter"
		}
		if value == "" {
			return nil, "Missing value parameter"
		}
		args = append(args, option, value)
	case "remove":
		option := q.Get("option")
		if option == "" {
			return nil, "Missing option parameter"
		}
		args = append(args, option)
	case "clear", "reset", "stats":
		// action already appended; nothing more
	case "list", "show":
		// The action token is already appended above. Previously this branch
		// re-appended it (producing "list list [active]"), which does not match
		// the CLI grammar. Only the optional literal "active" belongs here.
		if q.Get("args") == "active" {
			args = append(args, "active")
		}
	}

	return args, ""
}
