package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestListIgnorePatterns(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".keep", "keep.txt", "ntuser.dat"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		patterns   []string
		hideHidden bool
		want       []string
	}{
		{name: "unset", want: []string{".keep", "keep.txt", "node_modules", "ntuser.dat"}},
		{name: "configured", patterns: []string{"ntuser*", "[", "node_modules"}, want: []string{".keep", "keep.txt"}},
		{name: "hide hidden", patterns: []string{"ntuser*", "node_modules"}, hideHidden: true, want: []string{"keep.txt"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := model{path: dir, ignorePatterns: tt.patterns, hideHidden: tt.hideHidden}
			m.list()
			if m.err != nil {
				t.Fatal(m.err)
			}
			var got []string
			for _, file := range m.files {
				got = append(got, file.Name())
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("listed names = %v, want %v", got, tt.want)
			}
		})
	}
}
