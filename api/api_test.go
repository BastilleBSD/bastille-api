package api

import "testing"

func TestResolveBindAddr(t *testing.T) {
	cases := []struct {
		name         string
		host         string
		wantBind     string
		wantDisplay  string
		wantLoopback bool
	}{
		{"empty defaults to loopback", "", "127.0.0.1", "localhost", true},
		{"localhost binds loopback", "localhost", "127.0.0.1", "localhost", true},
		{"explicit loopback ip", "127.0.0.1", "127.0.0.1", "localhost", true},
		{"all interfaces opt-in", "0.0.0.0", "0.0.0.0", "0.0.0.0", false},
		{"specific address", "192.168.1.10", "192.168.1.10", "192.168.1.10", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bind, display := resolveBindAddr(tc.host)
			if bind != tc.wantBind {
				t.Errorf("resolveBindAddr(%q) bind = %q, want %q", tc.host, bind, tc.wantBind)
			}
			if display != tc.wantDisplay {
				t.Errorf("resolveBindAddr(%q) display = %q, want %q", tc.host, display, tc.wantDisplay)
			}
			if got := isLoopbackBind(bind); got != tc.wantLoopback {
				t.Errorf("isLoopbackBind(%q) = %v, want %v", bind, got, tc.wantLoopback)
			}
		})
	}
}

// Regression guard for the original bug: configuring host as "localhost" must
// NOT bind all interfaces.
func TestLocalhostDoesNotBindAllInterfaces(t *testing.T) {
	bind, _ := resolveBindAddr("localhost")
	if bind == "0.0.0.0" {
		t.Fatalf("localhost host bound to 0.0.0.0; must stay on loopback")
	}
	if !isLoopbackBind(bind) {
		t.Fatalf("localhost host bound to non-loopback address %q", bind)
	}
}
