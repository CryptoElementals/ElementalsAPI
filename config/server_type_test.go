package config

import "testing"

func TestNormalizeServerType(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", ServerTypeTrial},
		{"trial", ServerTypeTrial},
		{"TRIAL", ServerTypeTrial},
		{"normal", ServerTypeNormal},
		{"unknown", ServerTypeTrial},
	}
	for _, tc := range tests {
		if got := NormalizeServerType(tc.in); got != tc.want {
			t.Fatalf("NormalizeServerType(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
