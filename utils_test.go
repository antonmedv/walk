package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSubPath(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	if err := os.Mkdir(a, 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		target string
		want   bool
	}{
		{filepath.Join(a, "a"), true},
		{filepath.Join(a, "b", "c"), true},
		{a, true},
		{filepath.Join(dir, "a2"), false},
		{filepath.Join(dir, "b", "a"), false},
	}
	for _, tt := range tests {
		if got := isSubPath(a, tt.target); got != tt.want {
			t.Errorf("isSubPath(%q, %q) = %v, want %v", a, tt.target, got, tt.want)
		}
	}
}
