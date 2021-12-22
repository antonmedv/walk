package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	. "strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jwalton/go-supportscolor"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

var (
	modified  = lipgloss.NewStyle().Foreground(lipgloss.Color("#588FE6"))
	added     = lipgloss.NewStyle().Foreground(lipgloss.Color("#6ECC8E"))
	untracked = lipgloss.NewStyle().Foreground(lipgloss.Color("#D95C50"))
	cursor    = lipgloss.NewStyle().Background(lipgloss.Color("#825DF2")).Foreground(lipgloss.Color("#FFFFFF"))
	bar       = lipgloss.NewStyle().Background(lipgloss.Color("#5C5C5C")).Foreground(lipgloss.Color("#FFFFFF"))
)

func main() {
	term := supportscolor.Stderr()
	if term.Has16m {
		lipgloss.SetColorProfile(termenv.TrueColor)
	} else if term.Has256 {
		lipgloss.SetColorProfile(termenv.ANSI256)
	} else {
		lipgloss.SetColorProfile(termenv.ANSI)
	}

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if len(os.Args) == 2 {
		// Show usage on --help.
		if os.Args[1] == "--help" {
			_, _ = fmt.Fprintln(os.Stderr, "\n  "+cursor.Render(" llama ")+`

  Usage: llama [path]

  Key bindings:
    Arrows     Move cursor
    Enter      Enter directory
    Backspace  Exit directory
    [A-Z]      Fuzzy search
    Esc        Exit with cd
    Ctrl+C     Exit with noop
`)
			os.Exit(1)
		}
		// Maybe it is and argument, so get absolute path.
		path, err = filepath.Abs(os.Args[1])
		if err != nil {
			panic(err)
		}
	}

	m := &model{
		path:      path,
		width:     80,
		height:    60,
		positions: make(map[string]position),
	}
	m.list()
	m.status()

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	if err := p.Start(); err != nil {
		panic(err)
	}
	os.Exit(m.exitCode)
}

type model struct {
	path           string                    // Current dir path we are looking at.
	files          []fs.DirEntry             // Files we are looking at.
	c, r           int                       // Selector position in columns and rows.
	columns, rows  int                       // Displayed amount of rows and columns.
	width, height  int                       // Terminal size.
	offset         int                       // Scroll position.
	styles         map[string]lipgloss.Style // Colors of different files based on git status.
	editMode       bool                      // User opened file for editing.
	vimMode        bool                      // enable vim key bindings.
	positions      map[string]position       // Map of cursor positions per path.
	search         string                    // Search file by this name.
	searchMode     bool                      // fuzzy search with "/".
	updatedAt      time.Time                 // Time of last key press.
	matchedIndexes []int                     // List of char found indexes.
	prevName       string                    // Base name of previous directory before "up".
	findPrevName   bool                      // On View(), set c&r to point to prevName.
	exitCode       int                       // Exit code.
}

type position struct {
	c, r   int
	offset int
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editMode {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height - 1 // Account for the location bar.
		// Reset position history as c&r changes.
		m.positions = make(map[string]position)
		// Keep cursor at same place.
		m.prevName = m.cursorFileName()
		m.findPrevName = true
		// Also, m.c&r no longer point to correct indexes.
		m.c = 0
		m.r = 0
		return m, nil

	case tea.KeyMsg:
		m.vimMode, _ = strconv.ParseBool(lookup([]string{"LLAMA_VIM_KEYBINDINGS"}, "false"))

		if msg.Type == tea.KeyRunes && m.searchMode {
			// Input a regular character, do the search.
			if time.Now().Sub(m.updatedAt).Seconds() >= 1 {
				m.search = string(msg.Runes)
			} else {
				m.search += string(msg.Runes)
			}
			m.updatedAt = time.Now()
			names := make([]string, len(m.files))
			for i, fi := range m.files {
				names[i] = fi.Name()
			}
			matches := fuzzy.Find(m.search, names)
			if len(matches) > 0 {
				m.matchedIndexes = matches[0].MatchedIndexes
				index := matches[0].Index
				m.c = index / m.rows
				m.r = index % m.rows
			}
		} else {
			navKeys := []string{"up", "down", "left", "right"}
			if m.vimMode {
				navKeys = []string{"k", "j", "h", "l"}
			}

			switch keypress := msg.String(); keypress {
			case "/":
				if m.vimMode {
					m.searchMode = true
				}

			case "ctrl+c":
				_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
				m.exitCode = 2
				return m, tea.Quit

			case "esc":
				if m.searchMode && m.vimMode {
					m.searchMode = false
				} else {
					_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
					fmt.Println(m.path)            // Write to cd.
					m.exitCode = 0
					return m, tea.Quit
				}

			case "enter":
				m.searchMode = false
				newPath := filepath.Join(m.path, m.cursorFileName())
				if fi := fileInfo(newPath); fi.IsDir() {
					// Enter subdirectory.
					m.path = newPath
					if p, ok := m.positions[m.path]; ok {
						m.c = p.c
						m.r = p.r
						m.offset = p.offset
					} else {
						m.c = 0
						m.r = 0
						m.offset = 0
					}
					m.list()
					m.status()
				} else {
					// Open file.
					cmd := exec.Command(lookup([]string{"LLAMA_EDITOR", "EDITOR"}, "less"), filepath.Join(m.path, m.cursorFileName()))
					cmd.Stdin = os.Stdin
					cmd.Stdout = os.Stderr // Render to stderr.
					m.editMode = true
					_ = cmd.Run()
					m.editMode = false
					return m, tea.HideCursor
				}

			case "backspace":
				m.prevName = filepath.Base(m.path)
				m.path = filepath.Join(m.path, "..")
				if p, ok := m.positions[m.path]; ok {
					m.c = p.c
					m.r = p.r
					m.offset = p.offset
				} else {
					m.findPrevName = true
					m.list()
					m.status()
				}
				m.list()
				m.status()

			case navKeys[0]:
				m.r--
				if m.r < 0 {
					m.r = m.rows - 1
					m.c--
				}
				if m.c < 0 {
					m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
					m.c = m.columns - 1
				}

			case navKeys[1]:
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

			case navKeys[2]:
				m.c--
				if m.c < 0 {
					m.c = m.columns - 1
				}
				if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
					m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
					m.c = m.columns - 1
				}

			case navKeys[3]:
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
	}
	m.updateOffset()
	m.saveCursorPosition()
	return m, nil
}

func (m *model) View() string {
	if len(m.files) == 0 {
		return "No files"
	}

	// If it's possible to fit all files in one column on a third of the screen,
	// just use one column. Otherwise, let's squeeze listing in half of screen.
	m.columns = len(m.files) / (m.height / 3)
	if m.columns <= 0 {
		m.columns = 1
	}

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
				if m.findPrevName && m.prevName == name {
					m.c = i
					m.r = j
				}
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
		if len(Join(row, separator)) > m.width && m.columns > 1 {
			// Yep. No luck, let's decrease number of columns and try one more time.
			m.columns--
			goto start
		}
	}

	// If we need to select previous directory on "up".
	if m.findPrevName {
		m.findPrevName = false
		m.updateOffset()
		m.saveCursorPosition()
	}

	// Let's add colors from git status to file names.
	output := make([]string, m.rows)
	for j := 0; j < m.rows; j++ {
		row := make([]string, m.columns)
		for i := 0; i < m.columns; i++ {
			if i == m.c && j == m.r {
				row[i] = cursor.Render(names[i][j])
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
	if len(output) >= m.offset+m.height {
		output = output[m.offset : m.offset+m.height]
	}
	// Location bar.
	location := m.path
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		location = Replace(m.path, userHomeDir, "~", 1)
	}
	if len(location) > m.width {
		location = location[len(location)-m.width:]
	}
	locationBar := bar.Render(location)

	return locationBar + "\n" + Join(output, "\n")
}

func (m *model) list() {
	var err error
	m.files = nil
	m.styles = nil

	// ReadDir already returns files and dirs sorted by filename.
	m.files, err = os.ReadDir(m.path)
	if err != nil {
		panic(err)
	}
}

func (m *model) status() {
	// Going to keep file names and format string for git status.
	m.styles = map[string]lipgloss.Style{}

	status := m.gitStatus()
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

func (m *model) gitRepo() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Dir = m.path
	err := cmd.Run()
	return Trim(out.String(), "\n"), err
}

func (m *model) gitStatus() map[string]string {
	repo, err := m.gitRepo()
	if err != nil {
		return nil
	}
	cmd := exec.Command("git", "status", "--porcelain=v1")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Dir = m.path
	err = cmd.Run()
	if err != nil {
		return nil
	}
	paths := map[string]string{}
	for _, line := range Split(Trim(out.String(), "\n"), "\n") {
		if len(line) == 0 {
			continue
		}
		paths[filepath.Join(repo, line[3:])] = line[:2]
	}
	return paths
}

func (m *model) updateOffset() {
	// Scrolling down.
	if m.r >= m.offset+m.height {
		m.offset = m.r - m.height + 1
	}
	// Scrolling up.
	if m.r < m.offset {
		m.offset = m.r
	}
	// Don't scroll more than there are rows.
	if m.offset > m.rows-m.height && m.rows > m.height {
		m.offset = m.rows - m.height
	}
}

// Save position to restore later.
func (m *model) saveCursorPosition() {
	m.positions[m.path] = position{
		c:      m.c,
		r:      m.r,
		offset: m.offset,
	}
}

func (m *model) cursorFileName() string {
	i := m.c*m.rows + m.r
	if i < len(m.files) {
		return m.files[i].Name()
	}
	return ""
}

func fileInfo(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	return fi
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

func lookup(names []string, val string) string {
	for _, name := range names {
		val, ok := os.LookupEnv(name)
		if ok && val != "" {
			return val
		}
	}
	return val
}

// Copy-pasted from github.com/muesli/termenv@v0.9.0/termenv_unix.go.
// TODO: Refactor after, [feature](https://ï.at/stderr) implemented.
func colorProfile() termenv.Profile {
	term := os.Getenv("TERM")
	colorTerm := os.Getenv("COLORTERM")

	switch ToLower(colorTerm) {
	case "24bit":
		fallthrough
	case "truecolor":
		if term == "screen" || !HasPrefix(term, "screen") {
			// enable TrueColor in tmux, but not for old-school screen
			return termenv.TrueColor
		}
	case "yes":
		fallthrough
	case "true":
		return termenv.ANSI256
	}

	if Contains(term, "256color") {
		return termenv.ANSI256
	}
	if Contains(term, "color") {
		return termenv.ANSI
	}

	return termenv.Ascii
}
