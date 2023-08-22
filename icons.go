package main

import (
	"bufio"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"unicode"
)

type iconMap map[string]string

var icons iconMap

func parseIcons() {
	icons = make(iconMap)
	icons.parse()
}

//go:embed etc/icons
var f embed.FS

func (im iconMap) parse() {
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
	if f.Mode()&0111 != 0 {
		if val, ok := im["ex"]; ok {
			return val
		}
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

// This function reads whitespace separated string pairs at each line. Single
// or double quotes can be used to escape whitespaces. Hash characters can be
// used to add a comment until the end of line. Leading and trailing space is
// trimmed. Empty lines are skipped.
func readPairs(r io.Reader) ([][]string, error) {
	var pairs [][]string
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()

		squote, dquote := false, false
		for i := 0; i < len(line); i++ {
			if line[i] == '\'' && !dquote {
				squote = !squote
			} else if line[i] == '"' && !squote {
				dquote = !dquote
			}
			if !squote && !dquote && line[i] == '#' {
				line = line[:i]
				break
			}
		}

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		squote, dquote = false, false
		pair := strings.FieldsFunc(line, func(r rune) bool {
			if r == '\'' && !dquote {
				squote = !squote
			} else if r == '"' && !squote {
				dquote = !dquote
			}
			return !squote && !dquote && unicode.IsSpace(r)
		})

		if len(pair) != 2 {
			return nil, fmt.Errorf("expected pair but found: %s", s.Text())
		}

		for i := 0; i < len(pair); i++ {
			squote, dquote = false, false
			buf := make([]rune, 0, len(pair[i]))
			for _, r := range pair[i] {
				if r == '\'' && !dquote {
					squote = !squote
					continue
				}
				if r == '"' && !squote {
					dquote = !dquote
					continue
				}
				buf = append(buf, r)
			}
			pair[i] = string(buf)
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}
