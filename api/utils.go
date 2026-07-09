package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

)

func ValidateBastilleCommandParameters(c *Ctx, cmdArgs []string) error {

	logRequest("debug", "ValidateBastilleCommandParameters", c, cmdArgs, nil)

	query := c.Request.URL.Query()
	cmdName := cmdArgs[0]

	var subcmd *BastilleCommandStruct
	for _, cmd := range bastilleSpec.Commands {
		if cmd.Command == cmdName {
			subcmd = &cmd
			break
		}
	}

	paramsMap := make(map[string]struct{})
	for _, p := range subcmd.Parameters {
		paramsMap[strings.ToLower(p)] = struct{}{}
	}

	for param := range query {
		if _, ok := paramsMap[strings.ToLower(param)]; !ok {
			err := fmt.Errorf("invalid parameter %q for command %q", param, cmdName)
			logRequest("error", "invalid parameter", c, cmdArgs, err.Error())
			return err
		}
	}

	optionsValueMap := make(map[string]interface{})
	for _, opt := range subcmd.Options {
		if opt.SFlag != "" {
			optionsValueMap[opt.SFlag] = opt.Value
		}
		if opt.LFlag != "" {
			optionsValueMap[opt.LFlag] = opt.Value
		}
	}

	optionsParam := query.Get("options")
	if optionsParam != "" {
		optionsParam = strings.ReplaceAll(optionsParam, "+", " ")
		opts := strings.Fields(optionsParam)

		for i := 0; i < len(opts); i++ {
			arg := opts[i]
			valueType, ok := optionsValueMap[arg]
			if !ok {
				err := fmt.Errorf("invalid option %q for command %q", arg, cmdName)
				logRequest("error", "invalid option", c, cmdArgs, err.Error())
				return err
			}

			if valueType == "" || valueType == nil {
				continue
			}

			if i+1 >= len(opts) {
				err := fmt.Errorf("option %q requires a value", arg)
				logRequest("error", "invalid option arg", c, cmdArgs, err.Error())
				return err
			}

			i++
			val := opts[i]

			if valueType == "int" {
				if _, err := strconv.Atoi(val); err != nil {
					err := fmt.Errorf("option %q requires a numeric value", arg)
					logRequest("error", "invalid option arg", c, cmdArgs, err.Error())
					return err
				}
			}

			if strings.HasPrefix(val, "-") {
				err := fmt.Errorf("option %q requires a value", arg)
				logRequest("error", "invalid option arg", c, cmdArgs, err.Error())
				return err
			}
		}
	}

	logRequest("debug", "command validated", c, cmdArgs, nil)

	return nil
}

// Return command options and parameters GET
// @Description Return supported options and parameters for any command
// @Tags spec
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param Authorization-ID header string true "API Key ID/Name (eg: bastille-key)"
// @Param Authorization header string true "Authorization Header (eg: Bearer <api-token>)"
// @Param command path string true "Command name"
// @Success 200 {object} interface{} "Command specs for the requested command"
// @Router /api/v1/bastille/{command} [get]
func GetCommandSpec(cmdName string) HandlerFunc {

	return func(c *Ctx) {

		logRequest("debug", "GetCommandSpec", c, cmdName, nil)

		var cmd interface{}
		for _, sc := range bastilleSpec.Commands {
			if sc.Command == cmdName {
				cmd = sc
				break
			}
		}

		c.JSON(http.StatusOK, cmd)
	}
}
