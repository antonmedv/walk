package main

import (
	"embed"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type iconMap map[string]string

func parseIcons() iconMap {
	im := make(iconMap)

	im.parseFile()

	return im
}

//go:embed etc/icons
var f embed.FS

func (im iconMap) parseFile() {
	icons, _ := f.Open("etc/icons")
	pairs, err := readPairs(icons)
	if err != nil {
		log.Printf("reading icons file: %s", err)
		return
	}

	for _, pair := range pairs {
		key, val := pair[0], pair[1]

		key = replaceTilde(key)

		if filepath.IsAbs(key) {
			key = filepath.Clean(key)
		}

		im[key] = val
	}
}

func (im iconMap) getIcon(f os.FileInfo) string {
	if f.IsDir() {
		if val, ok := im[f.Name()+"/"]; ok {
			return val
		}
	}

	var key string

	switch {
	case f.IsDir() && f.Mode()&os.ModeSticky != 0 && f.Mode()&0002 != 0:
		key = "tw"
	case f.IsDir() && f.Mode()&0002 != 0:
		key = "ow"
	case f.IsDir() && f.Mode()&os.ModeSticky != 0:
		key = "st"
	case f.IsDir():
		key = "di"
	case f.Mode()&os.ModeNamedPipe != 0:
		key = "pi"
	case f.Mode()&os.ModeSocket != 0:
		key = "so"
	case f.Mode()&os.ModeDevice != 0:
		key = "bd"
	case f.Mode()&os.ModeCharDevice != 0:
		key = "cd"
	case f.Mode()&os.ModeSetuid != 0:
		key = "su"
	case f.Mode()&os.ModeSetgid != 0:
		key = "sg"
	case f.Mode()&0111 != 0:
		key = "ex"
	}

	if val, ok := im[key]; ok {
		return val
	}

	if val, ok := im[f.Name()+"*"]; ok {
		return val
	}

	if val, ok := im["*"+f.Name()]; ok {
		return val
	}

	if val, ok := im[filepath.Base(f.Name())+".*"]; ok {
		return val
	}

	ext := filepath.Ext(f.Name())
	if val, ok := im["*"+strings.ToLower(ext)]; ok {
		return val
	}

	if val, ok := im["fi"]; ok {
		return val
	}

	return " "
}

func replaceTilde(s string) string {
	u, err := user.Current()
	if err != nil {
		log.Printf("user: %s", err)
	}
	if strings.HasPrefix(s, "~") {
		s = strings.Replace(s, "~", u.HomeDir, 1)
	}
	return s
}
