package main

import (
	"io/fs"
	"testing"
)

type mockFile struct {
	fs.DirEntry
	mode fs.FileMode
	size int64
}

type mockFileInfo struct {
	fs.FileInfo
	mode fs.FileMode
	size int64
}

func (m mockFile) Info() (fs.FileInfo, error) {
	return mockFileInfo{
		size: m.size,
		mode: m.mode,
	}, nil
}

func (m mockFileInfo) Mode() fs.FileMode { return m.mode }

func (m mockFileInfo) Size() int64 { return m.size }

func TestFileSize(t *testing.T) {
	testCases := []struct {
		size     int64
		expected string
	}{
		{0, "0B"},
		{500, "500B"},
		{1024, "1.0KB"},
		{1500, "1.5KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
		{1024 * 1024 * 1024 * 1024, "1.0TB"},
	}

	env := Env{}
	for _, tc := range testCases {
		mockFile := mockFile{size: tc.size}
		result := env.FileSize(mockFile)
		if result != tc.expected {
			t.Errorf("Failed: %v != %v", result, tc.expected)
		}
	}
}

func TestFileMode(t *testing.T) {
	testCases := []struct {
		mode     fs.FileMode
		expected string
	}{
		{0755 | fs.ModeDir, "drwxr-xr-x"},                        // Directory
		{0644, "-rw-r--r--"},                                     // Regular file
		{0777 | fs.ModeSetuid, "-rwsrwxrwx"},                     // SetUID
		{0777 | fs.ModeSetgid, "-rwxrwsrwx"},                     // SetGID
		{0777 | fs.ModeSticky, "-rwxrwxrwt"},                     // Sticky bit
		{fs.ModeSymlink | 0777, "lrwxrwxrwx"},                    // Symbolic link
		{fs.ModeSocket | 0777, "srwxrwxrwx"},                     // Socket
		{fs.ModeNamedPipe | 0777, "prwxrwxrwx"},                  // Named pipe
		{fs.ModeDevice | fs.ModeCharDevice | 0777, "crwxrwxrwx"}, // Character device
		{fs.ModeDevice | 0777, "brwxrwxrwx"},                     // Block device
	}

	env := Env{}
	for _, tc := range testCases {
		mockFile := mockFile{mode: tc.mode}
		result := env.FileMode(mockFile)
		if result != tc.expected {
			t.Errorf("Failed: %v != %v", result, tc.expected)
		}
	}
}
