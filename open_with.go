package main

import (
	"strings"
)

var openWith = make(map[string]string)

func parseOpenWith(s string) {
	for _, pair := range strings.Split(s, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		split := strings.Split(pair, ":")
		if len(split) != 2 {
			continue
		}
		openWith[split[0]] = split[1]
	}
}
