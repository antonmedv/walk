package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func extension(path string) string {
	return strings.TrimLeft(strings.ToLower(filepath.Ext(path)), ".")
}

func fileInfo(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	return fi
}

func lookup(names []string, val string) string {
	for _, name := range names {
		val, ok := os.LookupEnv(name)
		if ok && val != "" {
			return val
		}
	}
	return val
}

func remove(path string) {
	go func() {
		cmd, ok := os.LookupEnv("WALK_REMOVE_CMD")
		if !ok {
			_ = os.RemoveAll(path)
		} else {
			_ = exec.Command(cmd, path).Run()
		}
	}()
}

func leaveOnlyAscii(content []byte) string {
	var result []byte

	for _, b := range content {
		if b == '\t' {
			result = append(result, ' ', ' ', ' ', ' ')
		} else if b == '\r' {
			continue
		} else if (b >= 32 && b <= 127) || b == '\n' { // '\n' is kept if newline needs to be retained
			result = append(result, b)
		}
	}

	return string(result)
}
