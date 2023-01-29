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
	// if val, ok := im[f.path]; ok {
	// 	return val
	// }

	if f.IsDir() {
		if val, ok := im[f.Name()+"/"]; ok {
			return val
		}
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
