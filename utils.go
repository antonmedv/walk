package main

import (
	"io"
	"io/fs"
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

func permBit(bit fs.FileMode, c byte) byte {
	if bit != 0 {
		return c
	}
	return '-'
}

// copyFile copies a single file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	
	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()
	
	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}
	
	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	
	if err = os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}
	
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		
		if entry.IsDir() {
			err = copyDir(srcPath, dstPath)
		} else {
			err = copyFile(srcPath, dstPath)
		}
		
		if err != nil {
			return err
		}
	}
	
	return nil
}
