//go:build windows

package main

func (e Env) Owner() (string, error) {
	return "¯\\_(ツ)_/¯", nil
}
