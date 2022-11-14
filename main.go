package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	. "strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type keymap struct {
	ForceQuit key.Binding
	Quit      key.Binding
	Open      key.Binding

	// Arrow-based movement.
	Back  key.Binding
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding

	// Vim-based movement.
	VimUp    key.Binding
	VimDown  key.Binding
	VimLeft  key.Binding
	VimRight key.Binding

	// Search mode, applicable only when Vim mode is active.
	Search     key.Binding
	ExitSearch key.Binding
}

var defaultKeymap = keymap{
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	Quit:      key.NewBinding(key.WithKeys("esc")),
	Open:      key.NewBinding(key.WithKeys("enter")),
	Back:      key.NewBinding(key.WithKeys("backspace")),
	Up:        key.NewBinding(key.WithKeys("up")),
	Down:      key.NewBinding(key.WithKeys("down")),
	Left:      key.NewBinding(key.WithKeys("left")),
	Right:     key.NewBinding(key.WithKeys("right")),

	// Vim mode only.
	VimUp:    key.NewBinding(key.WithKeys("k")),
	VimDown:  key.NewBinding(key.WithKeys("j")),
	VimLeft:  key.NewBinding(key.WithKeys("h")),
	VimRight: key.NewBinding(key.WithKeys("l")),

	// Vim mode only.
	Search:     key.NewBinding(key.WithKeys("/")),
	ExitSearch: key.NewBinding(key.WithKeys("esc")),
}

func main() {
	output := termenv.NewOutput(os.Stderr)
	lipgloss.SetColorProfile(output.ColorProfile())

	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	vimMode := true
	if lookup([]string{"LLAMA_VIM_KEYBINDINGS"}, "") == "false" {
		vimMode = false
	}

	if len(os.Args) == 2 {
		// Show usage on --help or -h.
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			usage(vimMode)
		}

		// Maybe it is and argument, so get absolute path.
		path, err = filepath.Abs(os.Args[1])
		if err != nil {
			panic(err)
		}
	}

	keys := defaultKeymap
	if !vimMode {
		keys.VimUp.SetEnabled(false)
		keys.VimDown.SetEnabled(false)
		keys.VimLeft.SetEnabled(false)
		keys.VimRight.SetEnabled(false)
		keys.Search.SetEnabled(false)
	}

	m := &model{
		vimMode:   vimMode,
		keys:      keys,
		path:      path,
		width:     80,
		height:    60,
		positions: make(map[string]position),
	}
	m.list()
	m.status()

	if vimMode {
		// Initialize search mode keybindings.
		m.disableSearchMode()
	}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	if _, err := p.Run(); err != nil {
		panic(err)
	}
	os.Exit(m.exitCode)
}

// Show usage and exit.
func usage(vimMode bool) {
	_, _ = fmt.Fprintln(os.Stderr, "\n  "+cursor.Render(" llama ")+"\n\n  Usage: llama [path]\n")
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	if vimMode {
		fmt.Fprintln(w, "    Arrows, hjkl\tMove cursor")
	} else {
		fmt.Fprintln(w, "    Arrows\tMove cursor")
	}
	fmt.Fprintln(w, "    Enter\tEnter directory")
	fmt.Fprintln(w, "    Backspace\tExit directory")
	if vimMode {
		fmt.Fprintln(w, "    /\tEnter fuzzy match mode")
		fmt.Fprintln(w, "    Esc\tExit fuzzy match mode (when active)")
	}
	fmt.Fprintln(w, "    Esc\tExit with cd")
	fmt.Fprintln(w, "    Ctrl+C\tExit with noop")
	w.Flush()
	fmt.Print("\n")

	os.Exit(1)
}

type model struct {
	vimMode        bool                      // Whether or not we're using Vim keybindings.
	keys           keymap                    // Keybindings.
	path           string                    // Current dir path we are looking at.
	files          []fs.DirEntry             // Files we are looking at.
	c, r           int                       // Selector position in columns and rows.
	columns, rows  int                       // Displayed amount of rows and columns.
	width, height  int                       // Terminal size.
	offset         int                       // Scroll position.
	styles         map[string]lipgloss.Style // Colors of different files based on git status.
	positions      map[string]position       // Map of cursor positions per path.
	search         string                    // Type to select files with this value.
	searchMode     bool                      // Whether type-to-select is active. Always active in non-vim-mode.
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
		if !m.vimMode || m.searchMode {
			if msg.Type == tea.KeyRunes {
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
			}
		}

		switch {
		case key.Matches(msg, m.keys.ForceQuit):
			_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
			m.exitCode = 2
			return m, tea.Quit

		case key.Matches(msg, m.keys.Quit):
			_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
			fmt.Println(m.path)            // Write to cd.
			m.exitCode = 0
			return m, tea.Quit

		case key.Matches(msg, m.keys.Open):
			m.disableSearchMode()
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
				// Open file. This will block until complete.
				return m, m.openEditor()
			}

		case key.Matches(msg, m.keys.Back):
			m.disableSearchMode()
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

		case key.Matches(msg, m.keys.Up):
			m.moveUp()

		case key.Matches(msg, m.keys.VimUp):
			if !m.searchMode {
				m.moveUp()
			}

		case key.Matches(msg, m.keys.Down):
			m.moveDown()

		case key.Matches(msg, m.keys.VimDown):
			if !m.searchMode {
				m.moveDown()
			}

		case key.Matches(msg, m.keys.Left):
			m.moveLeft()

		case key.Matches(msg, m.keys.VimLeft):
			if !m.searchMode {
				m.moveLeft()
			}

		case key.Matches(msg, m.keys.Right):
			m.moveLeft()

		case key.Matches(msg, m.keys.VimRight):
			if !m.searchMode {
				m.moveRight()
			}

		case key.Matches(msg, m.keys.Search):
			m.enableSearchMode()

		case key.Matches(msg, m.keys.ExitSearch):
			m.disableSearchMode()
		}

	}
	m.updateOffset()
	m.saveCursorPosition()
	return m, nil
}

func (m *model) moveUp() {
	m.r--
	if m.r < 0 {
		m.r = m.rows - 1
		m.c--
	}
	if m.c < 0 {
		m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
		m.c = m.columns - 1
	}
}

func (m *model) moveDown() {
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
}

func (m *model) moveLeft() {
	m.c--
	if m.c < 0 {
		m.c = m.columns - 1
	}
	if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
		m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
		m.c = m.columns - 1
	}
}

func (m *model) moveRight() {
	m.c++
	if m.c >= m.columns {
		m.c = 0
	}
	if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
		m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
		m.c = m.columns - 1
	}
}

func (m *model) enableSearchMode() {
	m.searchMode = true
	m.keys.Search.SetEnabled(false)
	m.keys.ExitSearch.SetEnabled(true)
	m.keys.Quit.SetEnabled(false)
}

func (m *model) disableSearchMode() {
	m.searchMode = false
	m.keys.Search.SetEnabled(true)
	m.keys.ExitSearch.SetEnabled(false)
	m.keys.Quit.SetEnabled(true)
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

	filterIndicator := ""
	if m.searchMode {
		filterIndicator = " [Search]"
	}

	return locationBar + filterIndicator + "\n" + Join(output, "\n")
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

func (m model) openEditor() tea.Cmd {
	execCmd := exec.Command(lookup([]string{"LLAMA_EDITOR", "EDITOR"}, "less"), filepath.Join(m.path, m.cursorFileName()))
	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		// Note: we could return a message here indicating that editing is
		// finished and altering our application about any errors. For now,
		// however, that's not necessary.
		return nil
	})
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
