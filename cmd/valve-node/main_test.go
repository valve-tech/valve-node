package main

import (
	"strings"
	"testing"
)

// TestBindWarningLine locks in the pre-v0.1.0 exposure-warning fix: binding
// beyond loopback must produce a loud, non-empty warning naming the actual
// exposure (full control over plain HTTP); loopback binds (127.0.0.1 and
// localhost, with or without a port) must produce no warning at all.
func TestBindWarningLine(t *testing.T) {
	cases := []struct {
		name      string
		bind      string
		wantEmpty bool
	}{
		{name: "default loopback with port", bind: "127.0.0.1:8799", wantEmpty: true},
		{name: "localhost with port", bind: "localhost:8799", wantEmpty: true},
		{name: "bare loopback no port", bind: "127.0.0.1", wantEmpty: true},
		{name: "all interfaces", bind: "0.0.0.0:8799", wantEmpty: false},
		{name: "specific LAN address", bind: "192.168.1.5:8799", wantEmpty: false},
		{name: "all interfaces shorthand", bind: ":8799", wantEmpty: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := bindWarningLine(tc.bind)
			if tc.wantEmpty && got != "" {
				t.Fatalf("bindWarningLine(%q) = %q, want empty (loopback bind)", tc.bind, got)
			}
			if !tc.wantEmpty {
				if got == "" {
					t.Fatalf("bindWarningLine(%q) = empty, want a warning", tc.bind)
				}
				if !containsAll(got, "WARNING", "plain HTTP", "control") {
					t.Fatalf("bindWarningLine(%q) = %q, missing expected warning content", tc.bind, got)
				}
			}
		})
	}
}

// TestBindFlagUsageWarnsOfExposure locks in that --bind's own --help text
// carries the exposure warning too, not just the runtime print.
func TestBindFlagUsageWarnsOfExposure(t *testing.T) {
	if !containsAll(bindFlagUsage, "WARNING", "127.0.0.1", "plain HTTP") {
		t.Fatalf("bindFlagUsage = %q, missing expected warning content", bindFlagUsage)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
