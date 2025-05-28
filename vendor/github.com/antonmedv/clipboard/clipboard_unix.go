//go:build freebsd || linux || netbsd || openbsd || solaris || dragonfly
// +build freebsd linux netbsd openbsd solaris dragonfly

package clipboard

import (
	"errors"
	"os"
	"os/exec"
)

const (
	xsel               = "xsel"
	xclip              = "xclip"
	powershellExe      = "powershell.exe"
	clipExe            = "clip.exe"
	wlcopy             = "wl-copy"
	wlpaste            = "wl-paste"
	termuxClipboardGet = "termux-clipboard-get"
	termuxClipboardSet = "termux-clipboard-set"
)

var (
	xselPasteArgs = []string{xsel, "--output", "--clipboard"}
	xselCopyArgs  = []string{xsel, "--input", "--clipboard"}

	xclipPasteArgs = []string{xclip, "-out", "-selection", "clipboard"}
	xclipCopyArgs  = []string{xclip, "-in", "-selection", "clipboard"}

	powershellExePasteArgs = []string{powershellExe, "Get-Clipboard"}
	clipExeCopyArgs        = []string{clipExe}

	wlpasteArgs = []string{wlpaste, "--no-newline"}
	wlcopyArgs  = []string{wlcopy}

	termuxPasteArgs = []string{termuxClipboardGet}
	termuxCopyArgs  = []string{termuxClipboardSet}

	missingCommands = errors.New("No clipboard utilities available. Please install xsel, xclip, wl-clipboard or Termux:API add-on for termux-clipboard-get/set.")
)

type commandInfo struct {
	trimDOS bool

	pasteCmdArgs []string
	copyCmdArgs  []string

	unsupported bool
}

func findClipboardUtility() commandInfo {
	c := commandInfo{}

	if os.Getenv("WAYLAND_DISPLAY") != "" {
		c.pasteCmdArgs = wlpasteArgs
		c.copyCmdArgs = wlcopyArgs

		if _, err := exec.LookPath(wlcopy); err == nil {
			if _, err := exec.LookPath(wlpaste); err == nil {
				return c
			}
		}
	}

	if os.Getenv("WSL_DISTRO_NAME") != "" {
		c.pasteCmdArgs = powershellExePasteArgs
		c.copyCmdArgs = clipExeCopyArgs
		c.trimDOS = true

		if _, err := exec.LookPath(clipExe); err == nil {
			if _, err := exec.LookPath(powershellExe); err == nil {
				return c
			}
		}
	}

	c.pasteCmdArgs = xclipPasteArgs
	c.copyCmdArgs = xclipCopyArgs

	if _, err := exec.LookPath(xclip); err == nil {
		return c
	}

	c.pasteCmdArgs = xselPasteArgs
	c.copyCmdArgs = xselCopyArgs

	if _, err := exec.LookPath(xsel); err == nil {
		return c
	}

	c.pasteCmdArgs = termuxPasteArgs
	c.copyCmdArgs = termuxCopyArgs

	if _, err := exec.LookPath(termuxClipboardSet); err == nil {
		if _, err := exec.LookPath(termuxClipboardGet); err == nil {
			return c
		}
	}

	return commandInfo{unsupported: true}
}

func getPasteCommand(c commandInfo) *exec.Cmd {
	return exec.Command(c.pasteCmdArgs[0], c.pasteCmdArgs[1:]...)
}

func getCopyCommand(c commandInfo) *exec.Cmd {
	return exec.Command(c.copyCmdArgs[0], c.copyCmdArgs[1:]...)
}

func readAll() (string, error) {
	c := findClipboardUtility()
	if c.unsupported {
		return "", missingCommands
	}

	pasteCmd := getPasteCommand(c)
	out, err := pasteCmd.Output()
	if err != nil {
		return "", err
	}
	result := string(out)
	if c.trimDOS && len(result) > 1 {
		result = result[:len(result)-2]
	}
	return result, nil
}

func writeAll(text string) error {
	c := findClipboardUtility()
	if c.unsupported {
		return missingCommands
	}

	copyCmd := getCopyCommand(c)
	in, err := copyCmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := copyCmd.Start(); err != nil {
		return err
	}
	if _, err := in.Write([]byte(text)); err != nil {
		return err
	}
	if err := in.Close(); err != nil {
		return err
	}
	return copyCmd.Wait()
}
