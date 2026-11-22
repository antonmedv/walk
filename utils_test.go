package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListIgnore(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"NTUSER.DAT", "a.txt", "b.log"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("WALK_IGNORE", "NTUSER*;[") // an invalid pattern is skipped
	m := &model{path: dir}
	m.list()
	var got []string
	for _, f := range m.files {
		got = append(got, f.Name())
	}
	if want := "a.txt,b.log"; strings.Join(got, ",") != want {
		t.Errorf("list() = %v, want %v", got, want)
	}
}

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

func TestLeaveOnlyPrintable(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello\nworld", "hello\nworld"},
		{"a\tb", "a    b"},
		{"line\r\n", "line\n"},
		{"Привет, 世界 🙂", "Привет, 世界 🙂"},
		{"é", "é"}, // combining mark
		{"\x1b[31mred\x1b[0m", "[31mred[0m"},
		{"bell\x07 del\x7f nul\x00", "bell del nul"},
		{"c1\u009b", "c1"},
	}
	for _, tt := range tests {
		if got := leaveOnlyPrintable([]byte(tt.in)); got != tt.want {
			t.Errorf("leaveOnlyPrintable(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
