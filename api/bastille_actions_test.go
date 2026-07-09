package api

import (
	"net/url"
	"reflect"
	"testing"
)

// The action grammar must reproduce the exact argv the original hand-written
// branchy handlers produced. Each case's want is derived directly from the
// pre-refactor bastille.go logic.
func TestActionBuildArgs(t *testing.T) {
	cases := []struct {
		name    string
		command string
		query   string
		want    []string
	}{
		// config
		{"config set with value", "config", "target=j&action=set&property=foo&value=bar", []string{"config", "j", "set", "foo", "bar"}},
		{"config get without value", "config", "target=j&action=get&property=foo", []string{"config", "j", "get", "foo"}},

		// zfs
		{"zfs snapshot with tag", "zfs", "target=j&action=snapshot&tag=t1", []string{"zfs", "j", "snapshot", "t1"}},
		{"zfs snapshot without tag", "zfs", "target=j&action=snapshot", []string{"zfs", "j", "snapshot"}},
		{"zfs get key_value", "zfs", "target=j&action=get&key_value=k%3Dv", []string{"zfs", "j", "get", "k=v"}},
		{"zfs jail dataset+path", "zfs", "target=j&action=jail&dataset=tank/d&jail_path=/mnt", []string{"zfs", "j", "jail", "tank/d", "/mnt"}},
		{"zfs unjail uses dataset (Bug2)", "zfs", "target=j&action=unjail&dataset=tank/d", []string{"zfs", "j", "unjail", "tank/d"}},
		{"zfs options ordering", "zfs", "target=j&action=df&options=-x", []string{"zfs", "-x", "j", "df"}},

		// network
		{"network add with ip", "network", "target=j&action=add&iface=e0&ip=1.2.3.4", []string{"network", "j", "add", "e0", "1.2.3.4"}},
		{"network add without ip", "network", "target=j&action=add&iface=e0", []string{"network", "j", "add", "e0"}},
		{"network no action -> just target", "network", "target=j", []string{"network", "j"}},

		// rdr
		{"rdr default redirect", "rdr", "target=j&protocol=tcp&host_port=80&jail_port=8080", []string{"rdr", "j", "tcp", "80", "8080"}},
		{"rdr log", "rdr", "target=j&action=log&protocol=tcp&host_port=80&jail_port=8080&log_options=vv", []string{"rdr", "j", "tcp", "80", "8080", "log", "vv"}},
		{"rdr clear", "rdr", "target=j&action=clear", []string{"rdr", "j", "clear"}},

		// tags
		{"tags add", "tags", "target=j&action=add&tags=web", []string{"tags", "j", "add", "web"}},
		{"tags list without tags", "tags", "target=j&action=list", []string{"tags", "j", "list"}},

		// template
		{"template convert", "template", "action=convert&template=base", []string{"template", "convert", "base"}},
		{"template default target+template", "template", "target=j&template=base", []string{"template", "j", "base"}},

		// upgrade
		{"upgrade install", "upgrade", "target=j&action=install", []string{"upgrade", "j", "install"}},
		{"upgrade default new_release", "upgrade", "target=j&new_release=14.0-RELEASE", []string{"upgrade", "j", "14.0-RELEASE"}},

		// monitor
		{"monitor enable (no target)", "monitor", "action=enable", []string{"monitor", "enable"}},
		{"monitor add service", "monitor", "target=j&action=add&service=nginx", []string{"monitor", "j", "add", "nginx"}},
		{"monitor list optional service present", "monitor", "target=j&action=list&service=nginx", []string{"monitor", "j", "list", "nginx"}},
		{"monitor no action -> just target", "monitor", "target=j", []string{"monitor", "j"}},

		// etcupdate
		{"etcupdate bootstrap release", "etcupdate", "action=bootstrap&release=14.0", []string{"etcupdate", "bootstrap", "14.0"}},
		{"etcupdate update", "etcupdate", "target=j&action=update&release=14.0", []string{"etcupdate", "j", "update", "14.0"}},
		{"etcupdate default emits action value", "etcupdate", "target=j&action=diff", []string{"etcupdate", "j", "diff"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ac, ok := actionCommands[tc.command]
			if !ok {
				t.Fatalf("command %q not in actionCommands", tc.command)
			}
			q, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			got, bad := ac.build(tc.command, q)
			if bad != "" {
				t.Fatalf("unexpected bad request: %s", bad)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("build(%s?%s)\n got: %v\nwant: %v", tc.command, tc.query, got, tc.want)
			}
		})
	}
}

// Bad requests: missing required params and unmatched actions.
func TestActionBuildBadRequests(t *testing.T) {
	cases := []struct {
		command string
		query   string
		wantMsg string
	}{
		{"config", "target=j", "Invalid action parameter"},           // action not one of set/add/get/remove
		{"config", "target=j&action=set", "Missing property parameter"},
		{"config", "action=set&property=p", "Missing target parameter"}, // prefix required
		{"zfs", "target=j&action=bogus", "Invalid action parameter"},
		{"zfs", "target=j&action=get", "Missing key_value parameter"},
		{"zfs", "target=j&action=unjail", "Missing dataset parameter"},
		{"tags", "target=j&action=bogus", "Invalid action parameter"},
		{"rdr", "target=j&action=log&protocol=tcp&host_port=80&jail_port=8080", "Missing log_options parameter"},
		{"etcupdate", "target=j", "Missing action parameter"},
		{"upgrade", "target=j", "Missing new_release parameter"},
		{"network", "action=add&iface=e0", "Missing target parameter"},
	}

	for _, tc := range cases {
		t.Run(tc.command+"/"+tc.wantMsg, func(t *testing.T) {
			ac := actionCommands[tc.command]
			q, _ := url.ParseQuery(tc.query)
			got, bad := ac.build(tc.command, q)
			if bad != tc.wantMsg {
				t.Fatalf("bad = %q, want %q", bad, tc.wantMsg)
			}
			if got != nil {
				t.Fatalf("expected nil argv on bad request, got %v", got)
			}
		})
	}
}
