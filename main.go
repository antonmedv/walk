package main

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	. "strings"
	"sync"
	"text/tabwriter"
	"time"
	"unicode/utf8"

	"github.com/antonmedv/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

var Version = "v1.10.1"

const separator = "    " // Separator between columns.

var (
	mainColor   = lipgloss.Color("#825DF2")
	barColor    = lipgloss.Color("#5C5C5C")
	searchColor = lipgloss.Color("#499F1C")
	bold        = lipgloss.NewStyle().Bold(true)
	warning     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).PaddingLeft(1).PaddingRight(1)
	cursor      = lipgloss.NewStyle().Background(mainColor).Foreground(lipgloss.Color("#FFFFFF"))
	bar         = lipgloss.NewStyle().Background(barColor).Foreground(lipgloss.Color("#FFFFFF"))
	search      = lipgloss.NewStyle().Background(searchColor).Foreground(lipgloss.Color("#FFFFFF"))
	danger      = lipgloss.NewStyle().Background(lipgloss.Color("#FF0000")).Foreground(lipgloss.Color("#FFFFFF"))

	previewPlain = lipgloss.NewStyle().PaddingLeft(2)
	previewSplit = lipgloss.NewStyle().
			MarginLeft(1).
			PaddingLeft(1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(mainColor).
			BorderLeft(true)
	preview = previewPlain

	fileSeparator    = string(filepath.Separator)
	showIcons        = false
	dirOnly          = false
	startPreviewMode = false
	fuzzyByDefault   = false
	hideHiddenFlag   = false
	strlen           = runewidth.StringWidth
)

var (
	keyForceQuit = key.NewBinding(key.WithKeys("ctrl+c"))
	keyQuit      = key.NewBinding(key.WithKeys("esc"))
	keyQuitQ     = key.NewBinding(key.WithKeys("q"))
	keyOpen      = key.NewBinding(key.WithKeys("enter"))
	keyBack      = key.NewBinding(key.WithKeys("backspace"))
	keyFnDelete  = key.NewBinding(key.WithKeys("delete"))
	keyUp        = key.NewBinding(key.WithKeys("up"))
	keyDown      = key.NewBinding(key.WithKeys("down"))
	keyLeft      = key.NewBinding(key.WithKeys("left"))
	keyRight     = key.NewBinding(key.WithKeys("right"))
	keyTop       = key.NewBinding(key.WithKeys("shift+up"))
	keyBottom    = key.NewBinding(key.WithKeys("shift+down"))
	keyLeftmost  = key.NewBinding(key.WithKeys("shift+left"))
	keyRightmost = key.NewBinding(key.WithKeys("shift+right"))
	keyPageUp    = key.NewBinding(key.WithKeys("pgup"))
	keyPageDown  = key.NewBinding(key.WithKeys("pgdown"))
	keyHome      = key.NewBinding(key.WithKeys("home"))
	keyEnd       = key.NewBinding(key.WithKeys("end"))
	keyVimUp     = key.NewBinding(key.WithKeys("k"))
	keyVimDown   = key.NewBinding(key.WithKeys("j"))
	keyVimLeft   = key.NewBinding(key.WithKeys("h"))
	keyVimRight  = key.NewBinding(key.WithKeys("l"))
	keyVimTop    = key.NewBinding(key.WithKeys("g"))
	keyVimBottom = key.NewBinding(key.WithKeys("G"))
	keySearch    = key.NewBinding(key.WithKeys("/"))
	keyPreview   = key.NewBinding(key.WithKeys(" "))
	keyDelete    = key.NewBinding(key.WithKeys("d"))
	keyUndo      = key.NewBinding(key.WithKeys("u"))
	keyYank      = key.NewBinding(key.WithKeys("y"))
	keyHidden    = key.NewBinding(key.WithKeys("."))
	keyHelp      = key.NewBinding(key.WithKeys("?"))
)

func main() {
	go emitCO2(time.Second)

	startPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	argsWithoutFlags := make([]string, 0)
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--help" || os.Args[1] == "-h" {
			usage(os.Stderr, true)
			os.Exit(1)
		}
		if os.Args[i] == "--version" || os.Args[1] == "-v" {
			version()
		}
		if os.Args[i] == "--icons" {
			showIcons = true
			parseIcons()
			continue
		}
		if os.Args[i] == "--dir-only" {
			dirOnly = true
			continue
		}
		if os.Args[i] == "--preview" {
			startPreviewMode = true
			continue
		}
		if os.Args[i] == "--fuzzy" {
			fuzzyByDefault = true
			continue
		}
		if os.Args[i] == "--hide-hidden" {
			hideHiddenFlag = true
			continue
		}
		if os.Args[i] == "--with-border" {
			preview = previewSplit
			continue
		}
		argsWithoutFlags = append(argsWithoutFlags, os.Args[i])
	}

	if len(argsWithoutFlags) > 0 {
		startPath, err = filepath.Abs(argsWithoutFlags[0])
		if err != nil {
			panic(err)
		}
	}

	output := termenv.NewOutput(os.Stderr)
	lipgloss.SetColorProfile(output.ColorProfile())

	m := &model{
		path:        startPath,
		width:       80,
		height:      60,
		positions:   make(map[string]position),
		previewMode: startPreviewMode,
		hideHidden:  hideHiddenFlag,
	}
	m.list()

	opts := []tea.ProgramOption{
		tea.WithOutput(os.Stderr),
	}
	if m.previewMode {
		opts = append(opts, tea.WithAltScreen())
	}
	p := tea.NewProgram(m, opts...)
	if _, err := p.Run(); err != nil {
		panic(err)
	}
	os.Exit(m.exitCode)
}

type model struct {
	path              string              // Current dir path we are looking at.
	files             []fs.DirEntry       // Files we are looking at.
	err               error               // Error while listing files.
	c, r              int                 // Selector position in columns and rows.
	columns, rows     int                 // Displayed amount of rows and columns.
	width, height     int                 // Terminal size.
	offset            int                 // Scroll position.
	positions         map[string]position // Map of cursor positions per path.
	search            string              // Type to select files with this value.
	searchMode        bool                // Whether type-to-select is active.
	searchId          int                 // Search id to indicate what search we are currently on.
	matchedIndexes    []int               // List of char found indexes.
	prevName          string              // Base name of previous directory before "up".
	findPrevName      bool                // On View(), set c&r to point to prevName.
	exitCode          int                 // Exit code.
	previewMode       bool                // Whether preview is active.
	previewContent    string              // Content of preview.
	deleteCurrentFile bool                // Whether to delete current file.
	toBeDeleted       []toDelete          // Map of files to be deleted.
	yankedFilePath    string              // Show yank info
	hideHidden        bool                // Hide hidden files
	showHelp          bool                // Show help
}

type position struct {
	c, r   int
	offset int
}

type toDelete struct {
	path string
	at   time.Time
}

type (
	clearSearchMsg int
	toBeDeletedMsg int
)

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.height < 3 {
			m.height = 3
		}
		// Reset position history as c&r changes.
		m.positions = make(map[string]position)
		// Keep cursor at same place.
		fileName, ok := m.fileName()
		if ok {
			m.prevName = fileName
			m.findPrevName = true
		}
		// Also, m.c&r no longer point to the correct indexes.
		m.c = 0
		m.r = 0
		return m, nil

	case tea.KeyMsg:
		// Make undo work even if we are in fuzzy mode.
		if key.Matches(msg, keyUndo) && len(m.toBeDeleted) > 0 {
			m.toBeDeleted = m.toBeDeleted[:len(m.toBeDeleted)-1]
			m.list()
			m.previewContent = ""
			return m, nil
		}

		if fuzzyByDefault {
			if key.Matches(msg, keyBack) {
				if len(m.search) > 0 {
					m.search = m.search[:strlen(m.search)-1]
					return m, nil
				}
			} else if msg.Type == tea.KeyRunes {
				m.updateSearch(msg)
				// Save search id to clear only current search after delay.
				// User may have already started typing next search.
				m.searchId++
				searchId := m.searchId
				return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
					return clearSearchMsg(searchId)
				})
			}
		} else if m.searchMode {
			if key.Matches(msg, keySearch) {
				m.searchMode = false
				return m, nil
			} else if key.Matches(msg, keyBack) {
				if len(m.search) > 0 {
					m.search = m.search[:strlen(m.search)-1]
				} else {
					m.searchMode = false
				}
				return m, nil
			} else if msg.Type == tea.KeyRunes {
				m.updateSearch(msg)
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, keyForceQuit):
			_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
			m.exitCode = 2
			m.dontDoPendingDeletions()
			return m, tea.Quit

		case key.Matches(msg, keyQuit, keyQuitQ):
			_, _ = fmt.Fprintln(os.Stderr) // Keep last item visible after prompt.
			fmt.Println(m.path)            // Write to cd.
			m.exitCode = 0
			m.performPendingDeletions()
			return m, tea.Quit

		case key.Matches(msg, keyOpen):
			m.search = ""
			m.searchMode = false
			filePath, ok := m.filePath()
			if !ok {
				return m, nil
			}
			if fi := fileInfo(filePath); fi.IsDir() {
				// Enter subdirectory.
				m.path = filePath
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
			} else {
				// Open file. This will block until complete.
				return m, m.openEditor()
			}

		case key.Matches(msg, keyBack):
			m.search = ""
			m.searchMode = false
			m.prevName = filepath.Base(m.path)
			m.path = filepath.Join(m.path, "..")
			if p, ok := m.positions[m.path]; ok {
				m.c = p.c
				m.r = p.r
				m.offset = p.offset
			} else {
				m.findPrevName = true
			}
			m.list()
			return m, nil

		case key.Matches(msg, keyUp):
			m.moveUp()

		case key.Matches(msg, keyTop, keyPageUp, keyVimTop):
			m.moveTop()

		case key.Matches(msg, keyBottom, keyPageDown, keyVimBottom):
			m.moveBottom()

		case key.Matches(msg, keyLeftmost):
			m.moveLeftmost()

		case key.Matches(msg, keyRightmost):
			m.moveRightmost()

		case key.Matches(msg, keyHome):
			m.moveStart()

		case key.Matches(msg, keyEnd):
			m.moveEnd()

		case key.Matches(msg, keyVimUp):
			m.moveUp()

		case key.Matches(msg, keyDown):
			m.moveDown()

		case key.Matches(msg, keyVimDown):
			m.moveDown()

		case key.Matches(msg, keyLeft):
			m.moveLeft()

		case key.Matches(msg, keyVimLeft):
			m.moveLeft()

		case key.Matches(msg, keyRight):
			m.moveRight()

		case key.Matches(msg, keyVimRight):
			m.moveRight()

		case key.Matches(msg, keySearch):
			m.searchMode = true
			m.searchId++
			m.search = ""

		case key.Matches(msg, keyPreview):
			m.previewMode = !m.previewMode
			// Reset position history as c&r changes.
			m.positions = make(map[string]position)
			// Keep cursor at same place.
			fileName, ok := m.fileName()
			if !ok {
				return m, nil
			}
			m.prevName = fileName
			m.findPrevName = true

			if m.previewMode {
				return m, tea.EnterAltScreen
			} else {
				m.previewContent = ""
				return m, tea.ExitAltScreen
			}

		case key.Matches(msg, keyDelete, keyFnDelete):
			filePathToDelete, ok := m.filePath()
			if ok {
				if m.deleteCurrentFile {
					m.deleteCurrentFile = false
					m.toBeDeleted = append(m.toBeDeleted, toDelete{
						path: filePathToDelete,
						at:   time.Now().Add(6 * time.Second),
					})
					m.list()
					m.previewContent = ""
					return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
						return toBeDeletedMsg(0)
					})
				} else {
					m.deleteCurrentFile = true
				}
			}
			return m, nil

		case key.Matches(msg, keyYank):
			filePath, ok := m.filePath()
			if ok {
				clipboard.WriteAll(filePath)
				m.yankedFilePath = filePath
			}
			return m, nil

		case key.Matches(msg, keyHelp):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, keyHidden):
			m.hideHidden = !m.hideHidden
			m.list()

		} // End of switch statement for key presses.

		m.deleteCurrentFile = false
		m.showHelp = false
		m.yankedFilePath = ""
		m.updateOffset()
		m.saveCursorPosition()

	case clearSearchMsg:
		if m.searchId == int(msg) {
			m.search = ""
			m.searchMode = false
		}

	case toBeDeletedMsg:
		toBeDeleted := make([]toDelete, 0)
		for _, td := range m.toBeDeleted {
			if td.at.After(time.Now()) {
				toBeDeleted = append(toBeDeleted, td)
			} else {
				remove(td.path)
			}
		}
		m.toBeDeleted = toBeDeleted
		if len(m.toBeDeleted) > 0 {
			return m, tea.Tick(time.Second, func(time.Time) tea.Msg {
				return toBeDeletedMsg(0)
			})
		}
	}

	return m, nil
}

func (m *model) updateSearch(msg tea.KeyMsg) {
	m.search += string(msg.Runes)
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
	m.updateOffset()
	m.saveCursorPosition()
}

func (m *model) View() string {
	if m.showHelp {
		out := &Builder{}
		out.WriteString(bar.Render("help") + "\n\n")
		usage(out, false)
		return out.String()
	}

	width := m.width
	if m.previewMode {
		width = m.width / 2
	}
	height := m.listHeight()

	var names [][]string
	names, m.rows, m.columns = wrap(m.files, width, height, func(name string, i, j int) {
		if m.findPrevName && m.prevName == name {
			m.c = i
			m.r = j
		}
	})

	// If we need to select previous directory on "up".
	if m.findPrevName {
		m.findPrevName = false
		m.updateOffset()
		m.saveCursorPosition()
	}

	// After we have updated offset and saved cursor position, we can
	// preview currently selected file.
	m.preview()

	// Get output rows width before coloring.
	outputWidth := strlen(path.Base(m.path)) // Use current dir name as default.
	if m.previewMode {
		row := make([]string, m.columns)
		for i := 0; i < m.columns; i++ {
			if len(names[i]) > 0 {
				row[i] = names[i][0]
			} else {
				outputWidth = width
			}
		}
		outputWidth = max(outputWidth, strlen(Join(row, separator)))
	} else {
		outputWidth = width
	}

	// Let's add colors to file names.
	output := make([]string, m.rows)
	for j := 0; j < m.rows; j++ {
		row := make([]string, m.columns)
		for i := 0; i < m.columns; i++ {
			if i == m.c && j == m.r {
				if m.deleteCurrentFile {
					row[i] = danger.Render(names[i][j])
				} else {
					row[i] = cursor.Render(names[i][j])
				}
			} else {
				row[i] = names[i][j]
			}
		}
		output[j] = Join(row, separator)
	}

	if len(output) >= m.offset+height {
		output = output[m.offset : m.offset+height]
	}

	// Preview pane.
	fileName, _ := m.fileName()
	previewPane := bar.Render(fileName) + "\n"
	previewPane += m.previewContent

	// Location bar (grey).
	location := m.path
	if userHomeDir, err := os.UserHomeDir(); err == nil {
		location = Replace(m.path, userHomeDir, "~", 1)
	}
	if runtime.GOOS == "windows" {
		location = ReplaceAll(Replace(location, "\\/", fileSeparator, 1), "/", fileSeparator)
	}

	// Filter bar (green).
	filter := ""
	if m.searchMode || fuzzyByDefault {
		filter = fileSeparator + m.search

		// If fuzzy is on and search is empty, don't show filter.
		if fuzzyByDefault && m.search == "" {
			filter = ""
		}
	}
	barLen := strlen(location) + strlen(filter)
	if barLen > outputWidth {
		location = location[min(barLen-outputWidth, strlen(location)):]
	}
	barStr := bar.Render(location) + search.Render(filter)

	main := barStr + "\n" + Join(output, "\n")

	if m.err != nil {
		main = barStr + "\n" + warning.Render(m.err.Error())
	} else if len(m.files) == 0 {
		main = barStr + "\n" + warning.Render("No files")
	}

	// Delete bar.
	if len(m.toBeDeleted) > 0 {
		toDelete := m.toBeDeleted[len(m.toBeDeleted)-1]
		timeLeft := int(toDelete.at.Sub(time.Now()).Seconds())
		deleteBar := fmt.Sprintf("%v deleted. (u)ndo %v", path.Base(toDelete.path), timeLeft)
		main += "\n" + danger.Render(deleteBar)
	}

	// Yank success.
	if m.yankedFilePath != "" {
		yankBar := fmt.Sprintf("copied: %v", m.yankedFilePath)
		main += "\n" + bar.Render(yankBar)
	}

	if m.previewMode {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			main,
			preview.
				MaxHeight(m.height).
				Render(previewPane),
		)
	} else {
		return main
	}
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

func (m *model) moveTop() {
	m.r = 0
}

func (m *model) moveBottom() {
	m.r = m.rows - 1
	if m.c == m.columns-1 && (m.columns-1)*m.rows+m.r >= len(m.files) {
		m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
	}
}

func (m *model) moveLeftmost() {
	m.c = 0
}

func (m *model) moveRightmost() {
	m.c = m.columns - 1
	if (m.columns-1)*m.rows+m.r >= len(m.files) {
		m.r = m.rows - 1 - (m.columns*m.rows - len(m.files))
	}
}

func (m *model) moveStart() {
	m.moveLeftmost()
	m.moveTop()
}

func (m *model) moveEnd() {
	m.moveRightmost()
	m.moveBottom()
}

func (m *model) list() {
	var err error
	m.files = nil

	// ReadDir already returns files and dirs sorted by filename.
	files, err := os.ReadDir(m.path)
	if err != nil {
		m.err = err
		return
	} else {
		m.err = nil
	}

files:
	for _, file := range files {
		if m.hideHidden && HasPrefix(file.Name(), ".") {
			continue files
		}
		if dirOnly && !file.IsDir() {
			continue files
		}
		for _, toDelete := range m.toBeDeleted {
			if path.Join(m.path, file.Name()) == toDelete.path {
				continue files
			}
		}
		m.files = append(m.files, file)
	}
}

func (m *model) listHeight() int {
	h := m.height - 1 // Subtract 1 for location bar.
	if len(m.toBeDeleted) > 0 {
		h-- // Subtract 1 for delete bar.
	}
	return h
}

func (m *model) updateOffset() {
	height := m.listHeight()
	// Scrolling down.
	if m.r >= m.offset+height {
		m.offset = m.r - height + 1
	}
	// Scrolling up.
	if m.r < m.offset {
		m.offset = m.r
	}
	// Don't scroll more than there are rows.
	if m.offset > m.rows-height && m.rows > height {
		m.offset = m.rows - height
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *model) saveCursorPosition() {
	m.positions[m.path] = position{
		c:      m.c,
		r:      m.r,
		offset: m.offset,
	}
}

func (m *model) fileName() (string, bool) {
	i := m.c*m.rows + m.r
	if i >= len(m.files) || i < 0 {
		return "", false
	}
	return m.files[i].Name(), true
}

func (m *model) filePath() (string, bool) {
	fileName, ok := m.fileName()
	if !ok {
		return fileName, false
	}
	return path.Join(m.path, fileName), true
}

func (m *model) openEditor() tea.Cmd {
	filePath, ok := m.filePath()
	if !ok {
		return nil
	}

	cmdline := Split(lookup([]string{"WALK_EDITOR", "EDITOR"}, "less"), " ")
	cmdline = append(cmdline, filePath)

	execCmd := exec.Command(cmdline[0], cmdline[1:]...)
	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		// Note: we could return a message here indicating that editing is
		// finished and altering our application about any errors. For now,
		// however, that's not necessary.
		return nil
	})
}

func (m *model) preview() {
	if !m.previewMode {
		return
	}
	filePath, ok := m.filePath()
	if !ok {
		// Normally this should not happen
		m.previewContent = warning.Render("Invalid file to preview")
		return
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		m.previewContent = warning.Render(err.Error())
		return
	}

	width := m.width / 2
	height := m.height - 1 // Subtract 1 for name bar.

	if fileInfo.IsDir() {
		files, err := os.ReadDir(filePath)
		if err != nil {
			m.previewContent = warning.Render(err.Error())
			return
		}

		if len(files) == 0 {
			m.previewContent = warning.Render("No files")
			return
		}

		names, rows, columns := wrap(files, width, height, nil)

		output := make([]string, rows)
		for j := 0; j < rows; j++ {
			row := make([]string, columns)
			for i := 0; i < columns; i++ {
				row[i] = names[i][j]
			}
			output[j] = Join(row, separator)
		}
		if len(output) >= height {
			output = output[0:height]
		}
		m.previewContent = Join(output, "\n")
		return
	}

	if isImageExt(filePath) {
		img, err := drawImage(filePath, width, height)
		if err != nil {
			m.previewContent = warning.Render("No image preview available")
			return
		}
		m.previewContent = img
		return
	}

	var content []byte
	// If file is too big (> 100kb), read only first 100kb.
	if fileInfo.Size() > 100*1024 {
		file, err := os.Open(filePath)
		if err != nil {
			m.previewContent = err.Error()
			return
		}
		defer file.Close()
		content = make([]byte, 100*1024)
		_, err = file.Read(content)
		if err != nil {
			m.previewContent = err.Error()
			return
		}
	} else {
		content, err = os.ReadFile(filePath)
		if err != nil {
			m.previewContent = err.Error()
			return
		}
	}

	switch {
	case utf8.Valid(content):
		m.previewContent = leaveOnlyAscii(content)
	default:
		m.previewContent = warning.Render("No preview available")
	}
}

func leaveOnlyAscii(content []byte) string {
	var result []byte

	for _, b := range content {
		if b == '\t' {
			result = append(result, ' ', ' ', ' ', ' ')
		} else if b == '\r' {
			continue
		} else if (b >= 32 && b <= 127) || b == '\n' { // '\n' is kept if newline needs to be retained
			result = append(result, b)
		}
	}

	return string(result)
}

// TODO: Write tests for this function.
func wrap(files []os.DirEntry, width int, height int, callback func(name string, i, j int)) ([][]string, int, int) {
	// If the directory is empty, return no names, rows and columns.
	if len(files) == 0 {
		return nil, 0, 0
	}

	// If it's possible to fit all files in one column on a third of the screen,
	// just use one column. Otherwise, let's squeeze listing in half of screen.
	columns := len(files) / max(1, height/3)
	if columns <= 0 {
		columns = 1
	}

	// Max number of files to display in one column is 10 or 4 columns in total.
	columnsEstimate := int(math.Ceil(float64(len(files)) / 10))
	columns = max(columns, min(columnsEstimate, 4))

	// For large lists, don't use more than 2 columns.
	if len(files) > 100 {
		columns = 2
	}

	// Fifteenth column is enough for everyone.
	if columns > 15 {
		columns = 15
	}

start:
	// Let's try to fit everything in terminal width with this many columns.
	// If we are not able to do it, decrease column number and goto start.
	rows := int(math.Ceil(float64(len(files)) / float64(columns)))
	names := make([][]string, columns)
	n := 0

	for i := 0; i < columns; i++ {
		names[i] = make([]string, rows)
		maxNameSize := 0 // We will use this to determine max name size, and pad names in column with spaces.
		for j := 0; j < rows; j++ {
			if n >= len(files) {
				break // No more files to display.
			}
			name := ""
			if showIcons {
				info, err := files[n].Info()
				if err == nil {
					icon := icons.getIcon(info)
					if icon != "" {
						name += icon + " "
					}
				}
			}
			name += files[n].Name()
			if callback != nil {
				callback(files[n].Name(), i, j)
			}
			if files[n].IsDir() {
				// Dirs should have a slash at the end.
				name += fileSeparator
			}

			n++ // Next file.

			if maxNameSize < strlen(name) {
				maxNameSize = strlen(name)
			}
			names[i][j] = name
		}

		// Append spaces to make all names in one column of same size.
		for j := 0; j < rows; j++ {
			names[i][j] += Repeat(" ", maxNameSize-strlen(names[i][j]))
		}
	}

	// Let's verify was all columns have at least one file.
	for i := 0; i < columns; i++ {
		if names[i] == nil {
			columns--
			goto start
		}
		columnHaveAtLeastOneFile := false
		for j := 0; j < rows; j++ {
			if names[i][j] != "" {
				columnHaveAtLeastOneFile = true
				break
			}
		}
		if !columnHaveAtLeastOneFile {
			columns--
			goto start
		}
	}

	for j := 0; j < rows; j++ {
		row := make([]string, columns)
		for i := 0; i < columns; i++ {
			row[i] = names[i][j]
		}
		if strlen(Join(row, separator)) > width && columns > 1 {
			// Yep. No luck, let's decrease number of columns and try one more time.
			columns--
			goto start
		}
	}
	return names, rows, columns
}

func emitCO2(duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)

	var wg sync.WaitGroup

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
		}()
	}

	wg.Wait()
}

func (m *model) dontDoPendingDeletions() {
	for _, toDelete := range m.toBeDeleted {
		fmt.Fprintf(os.Stderr, "Was not deleted: %v\n", toDelete.path)
	}
}

func (m *model) performPendingDeletions() {
	for _, toDelete := range m.toBeDeleted {
		remove(toDelete.path)
	}
	m.toBeDeleted = nil
}

func fileInfo(path string) os.FileInfo {
	fi, err := os.Stat(path)
	if err != nil {
		panic(err)
	}
	return fi
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

func remove(path string) {
	go func() {
		cmd, ok := os.LookupEnv("WALK_REMOVE_CMD")
		if !ok {
			_ = os.RemoveAll(path)
		} else {
			_ = exec.Command(cmd, path).Run()
		}
	}()
}

func usage(out io.Writer, full bool) {
	if full {
		_, _ = fmt.Fprintf(out, "\n  "+bold.Render("walk "+Version)+"\n\n  Usage: walk [path]\n\n")
	}
	w := tabwriter.NewWriter(out, 0, 8, 2, ' ', 0)
	put := func(s string) {
		_, _ = fmt.Fprintln(w, s)
	}
	put("    arrows, hjkl\tMove cursor")
	put("    enter\tEnter directory")
	put("    backspace\tExit directory")
	put("    space\tToggle preview")
	put("    esc, q\tExit with cd")
	put("    ctrl+c\tExit without cd")
	put("    /\tFuzzy search")
	put("    d, delete\tDelete file or dir")
	put("    y\tCopy to clipboard")
	put("    .\tHide hidden files")
	put("    ?\tShow help")
	if full {
		put("\n  Flags:\n")
		put("    --icons\tdisplay icons")
		put("    --dir-only\tshow dirs only")
		put("    --hide-hidden\thide hidden files")
		put("    --preview\tdisplay preview")
		put("    --with-border\tpreview with border")
		put("    --fuzzy\tfuzzy mode")
	}
	_ = w.Flush()
	_, _ = fmt.Fprintf(out, "\n")
}

func version() {
	fmt.Printf("%s\n", Version)
	os.Exit(0)
}
