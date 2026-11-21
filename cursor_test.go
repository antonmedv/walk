package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCursorClampedAfterHidingFiles(t *testing.T) {
	initStyles()
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		for _, name := range []string{fmt.Sprintf("file%d", i), fmt.Sprintf(".hidden%d", i)} {
			if err := os.WriteFile(filepath.Join(dir, name), nil, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	m := &model{
		path:       dir,
		termWidth:  80,
		termHeight: 60,
		positions:  make(map[string]position),
	}
	m.list()
	m.View()
	m.moveEnd()
	m.View()
	if name, _ := m.currentFileName(); name != "file4" {
		t.Fatalf("expected cursor on file4, got %q", name)
	}

	m.hideHidden = true
	m.list()
	m.View()

	name, ok := m.currentFileName()
	if !ok {
		t.Fatalf("cursor out of range: r=%d c=%d rows=%d columns=%d files=%d", m.r, m.c, m.rows, m.columns, len(m.files))
	}
	if name != "file4" {
		t.Fatalf("expected cursor on last file file4, got %q", name)
	}
}
