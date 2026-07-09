package api

// bastille_declarative.go is a prototype of the Mn1 "schema-driven handlers"
// refactor. The large majority of the Bastille subcommands follow one shape:
//
//	bastille <command> [options...] <positional args in a fixed order>
//
// where some positional args are required, some are optional-trailing, and a few
// are whitespace-split (a single query value that expands to multiple CLI args,
// e.g. `cmd`/`pkg`/`sysrc`). That entire family can be described as data and run
// by one generic handler, replacing ~25 near-identical hand-written functions.
//
// The genuinely branchy subcommands (config, zfs, limits, rdr, monitor, network,
// tags, template, etcupdate, upgrade, setup, mount, bootstrap, console) still
// have conditional argument grammars and remain hand-written in bastille.go for
// now. Extending the schema with a small "action" grammar to absorb those is the
// remaining Mn1 work.

import (
	"net/http"
	"net/url"
	"strings"
)

// paramSpec describes one positional parameter of a command.
type paramSpec struct {
	name     string // query parameter name (must match api/bastille.json)
	required bool   // reject the request with 400 if missing
	split    bool   // expand the value on whitespace into multiple CLI args
}

// commandSpec is a fully declarative description of a simple Bastille command.
type commandSpec struct {
	extraArgs []string    // server-controlled flags injected right after options
	params    []paramSpec // positional parameters, in CLI order
}

// req/opt are small constructors that keep the command table readable.
func req(name string) paramSpec      { return paramSpec{name: name, required: true} }
func opt(name string) paramSpec      { return paramSpec{name: name} }
func reqSplit(name string) paramSpec { return paramSpec{name: name, required: true, split: true} }

// declarativeCommands is the data that replaces the hand-written handlers. Adding
// a command of this shape is now a one-line table entry instead of a ~30-line
// function.
var declarativeCommands = map[string]commandSpec{
	// options + a single required target
	"start":   {params: []paramSpec{req("target")}},
	"stop":    {params: []paramSpec{req("target")}},
	"restart": {params: []paramSpec{req("target")}},
	"destroy": {params: []paramSpec{req("target")}},
	"top":     {params: []paramSpec{req("target")}},
	"htop":    {params: []paramSpec{req("target")}},
	"update":  {params: []paramSpec{req("target")}},
	"verify":  {params: []paramSpec{req("target")}},

	// options + a single optional argument
	"list": {params: []paramSpec{opt("item")}},

	// options + multiple positional arguments
	"create":  {params: []paramSpec{req("name"), req("release"), req("ip"), opt("iface")}},
	"clone":   {params: []paramSpec{req("target"), req("new_name"), req("ip")}},
	"rename":  {params: []paramSpec{req("target"), req("new_name")}},
	"migrate": {params: []paramSpec{req("target"), req("destination")}},
	"cp":      {params: []paramSpec{req("target"), req("host_path"), req("jail_path")}},
	"rcp":     {params: []paramSpec{req("target"), req("jail_path"), req("host_path")}},
	"jcp":     {params: []paramSpec{req("source_jail"), req("source_path"), req("destination_jail"), req("destination_path")}},
	"umount":  {params: []paramSpec{req("target"), req("jail_path")}},

	// options + required + optional trailing
	"export": {params: []paramSpec{req("target"), opt("path")}},
	"import": {params: []paramSpec{req("file"), opt("release")}},
	"edit":   {params: []paramSpec{req("target"), opt("file")}},

	// options + a required, whitespace-split argument list
	"cmd":     {params: []paramSpec{req("target"), reqSplit("command")}},
	"pkg":     {params: []paramSpec{req("target"), reqSplit("args")}},
	"sysrc":   {params: []paramSpec{req("target"), reqSplit("args")}},
	"service": {params: []paramSpec{req("target"), req("service"), reqSplit("args")}},

	// options with a server-injected flag (bastille convert always runs -ay)
	"convert": {extraArgs: []string{"-ay"}, params: []paramSpec{req("target"), opt("release")}},
}

// build constructs the CLI argument vector from the request query. The order
// mirrors the original hand-written handlers exactly: command, then options,
// then any injected flags, then positional args. If a required parameter is
// absent, it returns its name in missingParam and a nil slice.
func (spec commandSpec) build(command string, q url.Values) (args []string, missingParam string) {
	args = []string{command}

	if options := q.Get("options"); options != "" {
		args = append(args, strings.Fields(options)...)
	}
	args = append(args, spec.extraArgs...)

	for _, p := range spec.params {
		value := q.Get(p.name)
		if value == "" {
			if p.required {
				return nil, p.name
			}
			continue
		}
		if p.split {
			args = append(args, strings.Fields(value)...)
		} else {
			args = append(args, value)
		}
	}

	return args, ""
}

// declarativeHandler builds a HandlerFunc for one command from its spec.
func declarativeHandler(command string, spec commandSpec) HandlerFunc {
	return func(c *Ctx) {

		logRequest("debug", "declarativeHandler:"+command, c, nil, nil)

		cmdArgs, missing := spec.build(command, c.Request.URL.Query())
		if missing != "" {
			c.JSON(http.StatusBadRequest, H{"error": "Missing " + missing + " parameter"})
			logRequest("error", "missing "+missing+" parameter", c, nil, nil)
			return
		}

		ParseAndRunBastilleCommand(c, cmdArgs)
	}
}
