package api

// bastille_actions.go extends the schema-driven approach (see
// bastille_declarative.go) to the "branchy" commands whose argument vector
// depends on an action value (config, zfs, network, rdr, tags, template,
// upgrade, monitor, etcupdate).
//
// Each command is described by an actionCommand: some prefix parameters emitted
// first, then a per-action ordered sequence of items. An item is a literal
// keyword, a parameter, or the action value itself. Unmatched actions fall to
// defaultBranch, or produce 400 when there is none.
//
// Two commands remain hand-written in bastille.go: `mount` (an all-or-nothing
// optional parameter group) and `limits` (whose original list/show handling
// appends the action token twice — a suspected bug preserved verbatim rather
// than encoded into this grammar; see AUDIT follow-ups).

import (
	"net/http"
	"net/url"
	"strings"
)

// argItem is one element of an action branch's argument sequence.
type argItem struct {
	literal       string     // emit this fixed token
	param         *paramSpec // emit this query parameter
	isActionValue bool       // emit the selected action value itself
}

func lit(s string) argItem   { return argItem{literal: s} }
func arg(p paramSpec) argItem { pp := p; return argItem{param: &pp} }
func actionVal() argItem      { return argItem{isActionValue: true} }

// actionCommand is a declarative description of a branchy command.
type actionCommand struct {
	actionParam   string                // query key selecting the branch (default "action")
	prefix        []paramSpec           // params emitted before the branch
	branches      map[string][]argItem  // action value -> argument sequence
	defaultBranch []argItem             // used for unmatched actions; nil => 400
}

// appendParam appends one parameter's value(s), returning an error message if a
// required parameter is missing.
func appendParam(args []string, p paramSpec, q url.Values) ([]string, string) {
	v := q.Get(p.name)
	if v == "" {
		if p.required {
			return args, "Missing " + p.name + " parameter"
		}
		return args, ""
	}
	if p.split {
		return append(args, strings.Fields(v)...), ""
	}
	return append(args, v), ""
}

// build constructs the CLI argument vector for a branchy command. On a bad
// request (missing required param or unmatched action) it returns a nil slice
// and a human-readable message.
func (ac actionCommand) build(command string, q url.Values) (args []string, badReq string) {
	args = []string{command}

	if options := q.Get("options"); options != "" {
		args = append(args, strings.Fields(options)...)
	}

	for _, p := range ac.prefix {
		var msg string
		if args, msg = appendParam(args, p, q); msg != "" {
			return nil, msg
		}
	}

	actionKey := ac.actionParam
	if actionKey == "" {
		actionKey = "action"
	}
	action := q.Get(actionKey)

	seq, ok := ac.branches[action]
	if !ok {
		if ac.defaultBranch == nil {
			return nil, "Invalid action parameter"
		}
		seq = ac.defaultBranch
	}

	for _, item := range seq {
		switch {
		case item.isActionValue:
			if action == "" {
				return nil, "Missing action parameter"
			}
			args = append(args, action)
		case item.param != nil:
			var msg string
			if args, msg = appendParam(args, *item.param, q); msg != "" {
				return nil, msg
			}
		default:
			args = append(args, item.literal)
		}
	}

	return args, ""
}

// actionCommands is the data replacing the hand-written branchy handlers.
var actionCommands = map[string]actionCommand{
	// options + target + <action> property [value]
	"config": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"set":    {lit("set"), arg(req("property")), arg(opt("value"))},
			"add":    {lit("add"), arg(req("property")), arg(opt("value"))},
			"get":    {lit("get"), arg(req("property")), arg(opt("value"))},
			"remove": {lit("remove"), arg(req("property")), arg(opt("value"))},
		},
	},

	// bootstrap uses a release form with no target; every other action takes a target
	"etcupdate": {
		branches: map[string][]argItem{
			"bootstrap": {lit("bootstrap"), arg(req("release"))},
			"update":    {arg(req("target")), lit("update"), arg(req("release"))},
		},
		defaultBranch: []argItem{arg(req("target")), actionVal()},
	},

	// enable/disable/status act on no target; add/delete/list act on a target
	"monitor": {
		branches: map[string][]argItem{
			"enable":  {lit("enable")},
			"disable": {lit("disable")},
			"status":  {lit("status")},
			"add":     {arg(req("target")), lit("add"), arg(req("service"))},
			"delete":  {arg(req("target")), lit("delete"), arg(req("service"))},
			"list":    {arg(req("target")), lit("list"), arg(opt("service"))},
		},
		defaultBranch: []argItem{arg(req("target"))},
	},

	"network": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"add":    {lit("add"), arg(req("iface")), arg(opt("ip"))},
			"remove": {lit("remove"), arg(req("iface"))},
		},
		defaultBranch: []argItem{}, // unknown action: just the target, no error
	},

	"rdr": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"clear": {lit("clear")},
			"reset": {lit("reset")},
			"list":  {lit("list")},
			"log":   {arg(req("protocol")), arg(req("host_port")), arg(req("jail_port")), lit("log"), arg(req("log_options"))},
		},
		defaultBranch: []argItem{arg(req("protocol")), arg(req("host_port")), arg(req("jail_port"))},
	},

	"tags": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"add":    {lit("add"), arg(req("tags"))},
			"delete": {lit("delete"), arg(req("tags"))},
			"list":   {lit("list"), arg(opt("tags"))},
		},
	},

	"template": {
		branches: map[string][]argItem{
			"convert": {lit("convert"), arg(req("template"))},
		},
		defaultBranch: []argItem{arg(req("target")), arg(req("template"))},
	},

	"upgrade": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"install": {lit("install")},
		},
		defaultBranch: []argItem{arg(req("new_release"))},
	},

	"zfs": {
		prefix: []paramSpec{req("target")},
		branches: map[string][]argItem{
			"snapshot": {lit("snapshot"), arg(opt("tag"))},
			"destroy":  {lit("destroy"), arg(opt("tag"))},
			"rollback": {lit("rollback"), arg(opt("tag"))},
			"df":       {lit("df")},
			"usage":    {lit("usage")},
			"get":      {lit("get"), arg(req("key_value"))},
			"set":      {lit("set"), arg(req("key_value"))},
			"jail":     {lit("jail"), arg(req("dataset")), arg(req("jail_path"))},
			"unjail":   {lit("unjail"), arg(req("jail_path"))},
		},
	},
}

// actionHandler builds a HandlerFunc for one branchy command from its spec.
func actionHandler(command string, ac actionCommand) HandlerFunc {
	return func(c *Ctx) {

		logRequest("debug", "actionHandler:"+command, c, nil, nil)

		cmdArgs, badReq := ac.build(command, c.Request.URL.Query())
		if badReq != "" {
			c.JSON(http.StatusBadRequest, H{"error": badReq})
			logRequest("error", badReq, c, nil, nil)
			return
		}

		ParseAndRunBastilleCommand(c, cmdArgs)
	}
}
