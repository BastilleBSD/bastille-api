package api

import (
	"net/url"
	"reflect"
	"testing"
)

// buildLimitsArgs must match the documented CLI grammar:
//
//	TARGET add OPTION VALUE | remove OPTION | clear|reset|stats | list|show [active]
//
// The list/show cases are the regression guard: the action token must appear
// exactly once (previously "list" was appended twice, yielding "list list").
func TestBuildLimitsArgs(t *testing.T) {
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"add", "target=j&action=add&option=memoryuse&value=1G", []string{"limits", "j", "add", "memoryuse", "1G"}},
		{"remove", "target=j&action=remove&option=memoryuse", []string{"limits", "j", "remove", "memoryuse"}},
		{"clear", "target=j&action=clear", []string{"limits", "j", "clear"}},
		{"reset", "target=j&action=reset", []string{"limits", "j", "reset"}},
		{"stats", "target=j&action=stats", []string{"limits", "j", "stats"}},
		{"options ordering", "target=j&action=stats&options=-a", []string{"limits", "-a", "j", "stats"}},

		// Regression: single action token, not "list list".
		{"list plain", "target=j&action=list", []string{"limits", "j", "list"}},
		{"list active", "target=j&action=list&args=active", []string{"limits", "j", "list", "active"}},
		{"show plain", "target=j&action=show", []string{"limits", "j", "show"}},
		{"show active", "target=j&action=show&args=active", []string{"limits", "j", "show", "active"}},
		// A non-"active" args value is ignored, matching the CLI (only [active] is valid).
		{"list ignores non-active args", "target=j&action=list&args=bogus", []string{"limits", "j", "list"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, err := url.ParseQuery(tc.query)
			if err != nil {
				t.Fatal(err)
			}
			got, bad := buildLimitsArgs(q)
			if bad != "" {
				t.Fatalf("unexpected bad request: %s", bad)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("buildLimitsArgs(%s)\n got: %v\nwant: %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestBuildLimitsArgsBadRequests(t *testing.T) {
	cases := []struct {
		query   string
		wantMsg string
	}{
		{"", "Missing target parameter"},
		{"target=j", "Missing action parameter"},
		{"target=j&action=add&value=1G", "Missing option parameter"},
		{"target=j&action=add&option=memoryuse", "Missing value parameter"},
		{"target=j&action=remove", "Missing option parameter"},
	}

	for _, tc := range cases {
		t.Run(tc.wantMsg, func(t *testing.T) {
			q, _ := url.ParseQuery(tc.query)
			got, bad := buildLimitsArgs(q)
			if bad != tc.wantMsg {
				t.Fatalf("bad = %q, want %q", bad, tc.wantMsg)
			}
			if got != nil {
				t.Fatalf("expected nil argv, got %v", got)
			}
		})
	}
}
