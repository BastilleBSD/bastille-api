package api

import (
	"net/url"
	"reflect"
	"testing"
)

// The declarative arg builder must reproduce the exact argument vectors the
// original hand-written handlers produced. Each case encodes the expected argv
// for a representative command shape.
func TestDeclarativeBuildArgs(t *testing.T) {
	cases := []struct {
		name    string
		command string
		query   string
		want    []string
	}{
		{
			name:    "simple required target",
			command: "start",
			query:   "target=jail1",
			want:    []string{"start", "jail1"},
		},
		{
			name:    "options are split and precede positionals",
			command: "destroy",
			query:   "target=jail1&options=-f+-a+-y",
			want:    []string{"destroy", "-f", "-a", "-y", "jail1"},
		},
		{
			name:    "optional trailing arg present",
			command: "list",
			query:   "item=jails",
			want:    []string{"list", "jails"},
		},
		{
			name:    "optional trailing arg absent",
			command: "list",
			query:   "",
			want:    []string{"list"},
		},
		{
			name:    "multiple positionals with optional iface",
			command: "create",
			query:   "name=test&release=15.0-release&ip=10.0.0.12&iface=vtnet0",
			want:    []string{"create", "test", "15.0-release", "10.0.0.12", "vtnet0"},
		},
		{
			name:    "optional iface omitted",
			command: "create",
			query:   "name=test&release=15.0-release&ip=10.0.0.12",
			want:    []string{"create", "test", "15.0-release", "10.0.0.12"},
		},
		{
			name:    "whitespace-split command arg",
			command: "cmd",
			query:   "target=jail1&command=ls+-la+/tmp",
			want:    []string{"cmd", "jail1", "ls", "-la", "/tmp"},
		},
		{
			name:    "service with split args",
			command: "service",
			query:   "target=jail1&service=nginx&args=restart+now",
			want:    []string{"service", "jail1", "nginx", "restart", "now"},
		},
		{
			name:    "injected extraArgs (convert -ay) after options",
			command: "convert",
			query:   "target=jail1&options=-x&release=custom",
			want:    []string{"convert", "-x", "-ay", "jail1", "custom"},
		},
		{
			name:    "reordered positionals (rcp: jail_path before host_path)",
			command: "rcp",
			query:   "target=jail1&jail_path=/a&host_path=/b",
			want:    []string{"rcp", "jail1", "/a", "/b"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec, ok := declarativeCommands[tc.command]
			if !ok {
				t.Fatalf("command %q not in declarativeCommands", tc.command)
			}
			q, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			got, missing := spec.build(tc.command, q)
			if missing != "" {
				t.Fatalf("unexpected missing param %q", missing)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("build(%s?%s)\n got: %v\nwant: %v", tc.command, tc.query, got, tc.want)
			}
		})
	}
}

// Missing required parameters (and partial optional groups) must be reported
// with a full message, and produce no argv.
func TestDeclarativeBuildBadRequest(t *testing.T) {
	cases := []struct {
		name    string
		command string
		query   string
		wantMsg string
	}{
		{"start missing target", "start", "", "Missing target parameter"},
		{"create missing ip", "create", "name=test&release=15.0-release", "Missing ip parameter"},
		{"cmd missing command", "cmd", "target=jail1", "Missing command parameter"},
		{"clone missing ip", "clone", "target=jail1&new_name=jail2", "Missing ip parameter"},
		{"mount partial group", "mount", "target=j&host_path=/h&jail_path=/j&fs_type=nullfs", "Missing mount parameter(s)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := declarativeCommands[tc.command]
			q, _ := url.ParseQuery(tc.query)
			got, badReq := spec.build(tc.command, q)
			if badReq != tc.wantMsg {
				t.Fatalf("badReq = %q, want %q", badReq, tc.wantMsg)
			}
			if got != nil {
				t.Fatalf("expected nil argv on bad request, got %v", got)
			}
		})
	}
}

// mount's all-or-nothing fstab group: absent group and full group both succeed.
func TestMountGroup(t *testing.T) {
	spec := declarativeCommands["mount"]

	q1, _ := url.ParseQuery("target=j&host_path=/h&jail_path=/j")
	got1, bad1 := spec.build("mount", q1)
	if bad1 != "" {
		t.Fatalf("unexpected bad request: %s", bad1)
	}
	if want := []string{"mount", "j", "/h", "/j"}; !reflect.DeepEqual(got1, want) {
		t.Fatalf("mount without group\n got: %v\nwant: %v", got1, want)
	}

	q2, _ := url.ParseQuery("target=j&host_path=/h&jail_path=/j&fs_type=nullfs&fs_options=ro&dump=0&pass_number=0")
	got2, bad2 := spec.build("mount", q2)
	if bad2 != "" {
		t.Fatalf("unexpected bad request: %s", bad2)
	}
	if want := []string{"mount", "j", "/h", "/j", "nullfs", "ro", "0", "0"}; !reflect.DeepEqual(got2, want) {
		t.Fatalf("mount with full group\n got: %v\nwant: %v", got2, want)
	}
}

// Guard against a command being defined both hand-written and declaratively.
func TestNoDuplicateCommandDefinitions(t *testing.T) {
	// bastilleRoutes panics on a duplicate; a successful build with the full
	// expected count confirms the two sets are disjoint and complete.
	routes := bastilleRoutes()
	if len(routes) != 39 {
		t.Fatalf("expected 39 bastille routes, got %d", len(routes))
	}
	for command := range declarativeCommands {
		if _, ok := routes[command]; !ok {
			t.Errorf("declarative command %q missing from routes", command)
		}
	}
}
