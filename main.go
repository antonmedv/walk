package main

import (
	"bytes"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/crypto/ssh/terminal"
	"io/fs"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	. "strings"
)

var (
	modified  = lipgloss.NewStyle().Foreground(lipgloss.Color("#4688F2"))
	added     = lipgloss.NewStyle().Foreground(lipgloss.Color("#47DE47"))
	untracked = lipgloss.NewStyle().Foreground(lipgloss.Color("#E84343"))
	selector  = lipgloss.NewStyle().
			Background(lipgloss.Color("#825DF2")).
			Foreground(lipgloss.Color("#FFFFFF"))
)

func main() {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if len(os.Args) == 2 {
		path = os.Args[1]
	}

	// If stdout of ll piped, use ls behavior: one line, no colors.
	fi, err := os.Stdout.Stat()
	if err != nil {
		panic(err)
	}
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, file := range files {
			fmt.Println(file.Name())
		}
		return
	}

	m := &model{path: path}
	m.list()
	m.status()

	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
	}
}

type model struct {
	path          string
	c, r          int
	columns, rows int
	width, height int
	styles        map[string]lipgloss.Style
	files         []fs.FileInfo
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "esc", "q":
			fmt.Println()
			fmt.Fprintln(os.Stderr, m.path)
			return m, tea.Quit

		case "e":
			cmd := exec.Command("less", filepath.Join(m.path, m.files[m.c*m.rows+m.r].Name()))
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Run()

		case "enter":
			m.path = filepath.Join(m.path, m.files[m.c*m.rows+m.r].Name())
			m.c = 0
			m.r = 0
			m.list()
			m.status()
			return m, nil

		case "backspace":
			m.path = filepath.Join(m.path, "..")
			m.list()
			m.status()
			return m, nil

		case "up":
			m.r--
			if m.r < 0 {
				m.r = m.rows - 1
				m.c--
			}
			if m.c < 0 {
				m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
				m.c = m.columns - 1
			}

		case "down":
			m.r++
			if m.r >= m.rows {
				m.r = 0
				m.c++
			}
			if m.c >= m.columns {
				m.c = 0
			}
			if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
				m.r = 0
				m.c = 0
			}

		case "left":
			m.c--
			if m.c < 0 {
				m.c = m.columns - 1
			}
			if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
				m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
				m.c = m.columns - 1
			}

		case "right":
			m.c++
			if m.c >= m.columns {
				m.c = 0
			}
			if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
				m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
				m.c = m.columns - 1
			}
		}
	}

	return m, nil
}

func (m *model) View() string {
	if len(m.files) == 0 {
		return "No files"
	}

	// We need terminal size to nicely fit on screen.
	fd := int(os.Stdin.Fd())
	width, height, err := terminal.GetSize(fd)
	if err != nil {
		width, height = 80, 60
	}

	// If it's possible to fit all files in one column on half of screen, just use one column.
	// Otherwise, let's squeeze listing in half of screen.
	m.columns = len(m.files)/(height/2) + 1

start:
	// Let's try to fit everything in terminal width with this many columns.
	// If we are not able to do it, decrease column number and goto start.
	m.rows = int(math.Ceil(float64(len(m.files)) / float64(m.columns)))
	names := make([][]string, m.columns)
	n := 0
	for i := 0; i < m.columns; i++ {
		names[i] = make([]string, m.rows)
		// Columns size is going to be of max file name size.
		max := 0
		for j := 0; j < m.rows; j++ {
			name := ""
			if n < len(m.files) {
				name = m.files[n].Name()
				if m.files[n].IsDir() {
					// Dirs should have a slash at the end.
					name += "/"
				}
				n++
			}
			if max < len(name) {
				max = len(name)
			}
			names[i][j] = name
		}
		// Append spaces to make all names in one column of same size.
		for j := 0; j < m.rows; j++ {
			names[i][j] += Repeat(" ", max-len(names[i][j]))
		}
	}

	const separator = "    " // Separator between columns.
	for j := 0; j < m.rows; j++ {
		row := make([]string, m.columns)
		for i := 0; i < m.columns; i++ {
			row[i] = names[i][j]
		}
		if len(Join(row, separator)) > width && m.columns > 1 {
			// Yep. No luck, let's decrease number of columns and try one more time.
			m.columns--
			goto start
		}
	}

	// Let's add colors from git status to file names.
	output := make([]string, m.rows)
	for j := 0; j < m.rows; j++ {
		row := make([]string, m.columns)
		for i := 0; i < m.columns; i++ {
			if i == m.c && j == m.r {
				row[i] = selector.Render(names[i][j])
				continue
			}
			s, ok := m.styles[TrimRight(names[i][j], " ")]
			if ok {
				row[i] = s.Render(names[i][j])
			} else {
				row[i] = names[i][j]
			}

		}
		output[j] = Join(row, separator)
	}

	return Join(output, "\n")
}

func (m *model) list() {
	m.files = nil
	m.styles = nil

	// Maybe it is and argument, so get absolute path.
	cwd, err := filepath.Abs(m.path)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Is it a file?
	if fi := fileInfo(cwd); !fi.IsDir() {
		return
	}

	// ReadDir already returns files and dirs sorted by filename.
	m.files, err = ioutil.ReadDir(cwd)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func (m *model) status() {
	// Going to keep file names and format string for git status.
	m.styles = map[string]lipgloss.Style{}

	status := gitStatus()
	for _, file := range m.files {
		name := file.Name()
		if file.IsDir() {
			name += "/"
		}
		// gitStatus returns file names of modified files from repo root.
		fullPath := filepath.Join(m.path, name)
		for path, mode := range status {
			if subPath(path, fullPath) {
				if mode[0] == '?' || mode[1] == '?' {
					m.styles[name] = untracked
				} else if mode[0] == 'A' || mode[1] == 'A' {
					m.styles[name] = added
				} else if mode[0] == 'M' || mode[1] == 'M' {
					m.styles[name] = modified
				}
			}
		}
	}
}

func subPath(path string, fullPath string) bool {
	p := Split(path, "/")
	for i, s := range Split(fullPath, "/") {
		if i >= len(p) {
			return false
		}
		if p[i] != s {
			return false
		}
	}
	return true
}

func gitRepo() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	return Trim(out.String(), "\n"), err
}

func gitStatus() map[string]string {
	repo, err := gitRepo()
	if err != nil {
		return nil
	}
	cmd := exec.Command("git", "status", "--porcelain=v1")
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		return nil
	}
	m := map[string]string{}
	for _, line := range Split(Trim(out.String(), "\n"), "\n") {
		if len(line) == 0 {
			continue
		}
		m[filepath.Join(repo, line[3:])] = line[:2]
	}
	return m
}

func fileInfo(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return fi
}
