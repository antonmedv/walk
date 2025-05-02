package main

import (
	"fmt"
	"io/fs"
	"math"
	"time"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

func compile(code string) *vm.Program {
	p, err := expr.Compile(code, expr.Env(Env{}))
	if err != nil {
		panic(err)
	}
	return p
}

type Env struct {
	DirPath     string
	Files       []fs.DirEntry `expr:"files"`
	CurrentFile fs.DirEntry   `expr:"current_file"`
}

func (e Env) Sprintf(format string, a ...any) string {
	return fmt.Sprintf(format, a...)
}

func (e Env) PadLeft(s string, n int) string {
	return fmt.Sprintf("%*s", n, s)
}

func (e Env) PadRight(s string, n int) string {
	return fmt.Sprintf("%*s", n, s)
}

func (e Env) Size() string {
	info, err := e.CurrentFile.Info()
	if err != nil {
		return "N/A"
	}
	size := float64(info.Size())
	if size == 0 {
		return "0B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	base := math.Log(size) / math.Log(1024)
	unitIndex := int(math.Floor(base))
	if unitIndex >= len(units) {
		unitIndex = len(units) - 1
	}
	value := size / math.Pow(1024, float64(unitIndex))
	if unitIndex == 0 {
		return fmt.Sprintf("%.0f%s", value, units[unitIndex])
	}
	return fmt.Sprintf("%.1f%s", value, units[unitIndex])
}

func (e Env) Mode() string {
	info, err := e.CurrentFile.Info()
	if err != nil {
		return "?????????"
	}

	mode := info.Mode()

	result := make([]byte, 10)

	switch {
	case mode.IsDir():
		result[0] = 'd'
	case mode&fs.ModeSymlink != 0:
		result[0] = 'l'
	case mode&fs.ModeSocket != 0:
		result[0] = 's'
	case mode&fs.ModeNamedPipe != 0:
		result[0] = 'p'
	case mode&fs.ModeCharDevice != 0:
		result[0] = 'c'
	case mode&fs.ModeDevice != 0:
		result[0] = 'b'
	default:
		result[0] = '-'
	}

	// Owner permissions
	result[1] = permBit(mode&0400, 'r')
	result[2] = permBit(mode&0200, 'w')
	result[3] = permBit(mode&0100, 'x')

	// Group permissions
	result[4] = permBit(mode&040, 'r')
	result[5] = permBit(mode&020, 'w')
	result[6] = permBit(mode&010, 'x')

	// Others permissions
	result[7] = permBit(mode&04, 'r')
	result[8] = permBit(mode&02, 'w')
	result[9] = permBit(mode&01, 'x')

	// Handle special permission bits
	if mode&fs.ModeSetuid != 0 {
		result[3] = 's'
	}
	if mode&fs.ModeSetgid != 0 {
		result[6] = 's'
	}
	if mode&fs.ModeSticky != 0 {
		result[9] = 't'
	}

	return string(result)
}

func (e Env) ModTime() string {
	info, err := e.CurrentFile.Info()
	if err != nil {
		return "???"
	}
	modTime := info.ModTime()
	now := time.Now()
	if modTime.Year() == now.Year() {
		return modTime.Format("Jan 2 15:04")
	}
	return modTime.Format("Jan 2 2006")
}
