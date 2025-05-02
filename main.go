package main

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	. "strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/quick"
	"github.com/antonmedv/clipboard"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
	"github.com/sahilm/fuzzy"
)

var Version = "v1.13.0"

var separator = "    " // Separator between columns.

var (
	fileSeparator  = string(filepath.Separator)
	showIcons      = false
	dirOnly        = false
	fuzzyByDefault = false
	withBorder     = false
	withHighlight  = true
	strlen         = runewidth.StringWidth
)

func main() {
	startPath, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if s, ok := os.LookupEnv("WALK_OPEN_WITH"); ok {
		parseOpenWith(s)
	}

	if color, ok := os.LookupEnv("WALK_MAIN_COLOR"); ok {
		mainColor = lipgloss.Color(color)
	}

	if _, ok := os.LookupEnv("WALK_NO_HIGHLIGHT"); ok {
		withHighlight = false
	}

	initStyles()

	m := &model{
		termWidth:  80,
		termHeight: 60,
		positions:  make(map[string]position),
	}

	initExtra(m)

	if statusBar, ok := os.LookupEnv("WALK_STATUS_BAR"); ok {
		m.statusBar = compile(statusBar)
	}

	argsWithoutFlags := make([]string, 0)
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--help" || os.Args[1] == "-h" {
			usage(os.Stderr, true)
			os.Exit(1)
		}
		if os.Args[i] == "--version" || os.Args[1] == "-v" {
			fmt.Printf("%s\n", Version)
			os.Exit(0)
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
		if os.Args[i] == "--dirs-first" {
			dirsFirst = true
			continue
		}
		if os.Args[i] == "--preview" {
			m.previewMode = true
			continue
		}
		if os.Args[i] == "--fuzzy" {
			fuzzyByDefault = true
			continue
		}
		if os.Args[i] == "--hide-hidden" {
			m.hideHidden = true
			continue
		}
		if os.Args[i] == "--with-border" {
			withBorder = true
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

	m.path = startPath
	m.list()

	opts := []tea.ProgramOption{
		tea.WithOutput(os.Stderr),
	}
	if m.previewMode {
		opts = append(opts, tea.WithAltScreen())
	}

	p := tea.NewProgram(m, opts...)
	lastM, err := p.Run()
	if err != nil {
		panic(err)
	}

	m = lastM.(*model)
	if m.exitCode == 0 {
		if fileEntry, ok := m.currentFile(); ok {
			fmt.Println(fileEntry.dirPath) // Write to cd.
		}
	}

	os.Exit(m.exitCode)
}

type model struct {
	path                  string              // Current dir path we are looking at.
	files                 []fileEntry         // Files we are looking at.
	err                   error               // Error while listing files.
	currenFileIndex       int                 // The index of the currently highlighted file (i.e. the current file).
	columns, rows         int                 // Displayed amount of rows and columns.
	termWidth, termHeight int                 // Terminal size.
	firstFileIndex        int                 // Scroll position.
	positions             map[string]position // Map of cursor positions per path.
	search                string              // Type to select files with this value.
	searchMode            bool                // Whether type-to-select is active.
	searchId              int                 // Search id to indicate what search we are currently on.
	matchedIndexes        []int               // List of char found indexes.
	prevName              string              // Full path of previous directory before "up".
	findPrevName          bool                // On View(), set c&r to point to prevName.
	exitCode              int                 // Exit code.
	previewMode           bool                // Whether preview is active.
	previewContent        string              // Content of preview.
	deleteCurrentFile     bool                // Whether to delete current file.
	toBeDeleted           []toDelete          // Map of files to be deleted.
	yankedFilePath        string              // Show yank info
	hideHidden            bool                // Hide hidden files
	showHelp              bool                // Show help
	statusBar             *vm.Program         // Status bar program.
	quitting              bool                // Whether we are quitting the program.
	fileInfoProg          *vm.Program         // File info program.
	extra                 extraModel
}

type position struct {
	firstDisplayedFileIndex int
	cursorFileIndex         int
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
	//log.Println("UPD: msg: ", reflect.TypeOf(msg).Name())
	if mm, cmd, updated := extraUpdate(m, msg); updated {
		return mm, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		if m.termHeight < 3 {
			m.termHeight = 3
		}
		// Reset position history as c&r changes.
		m.positions = make(map[string]position)
		// Keep cursor at same place.
		filePath, ok := m.filePath()
		if ok {
			m.prevName = filePath
			m.findPrevName = true
		}
		// Also, m.currenFileIndex no longer point to the correct index.
		m.currenFileIndex = 0
		m.columns = -1
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
			m.quitting = true
			m.exitCode = 2
			m.dontDoPendingDeletions()
			return m, tea.Quit

		case key.Matches(msg, keyQuit, keyQuitQ):
			m.quitting = true
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
				enterDirectory(m, filePath)
				return m, nil
			} else {
				// Open file. This will block until complete.
				return m, m.open()
			}

		case key.Matches(msg, keyUpDir):
			m.search = ""
			m.searchMode = false
			m.prevName = m.path
			m.path = filepath.Join(m.path, "..")
			if p, ok := m.positions[m.path]; ok {
				m.currenFileIndex = p.cursorFileIndex
				m.firstFileIndex = p.firstDisplayedFileIndex
			} else {
				m.findPrevName = true
			}
			m.list()
			return m, nil

		case key.Matches(msg, keyUp):
			m.moveUp()

		case key.Matches(msg, keyPageUp):
			m.movePageUp()

		case key.Matches(msg, keyPageDown):
			m.movePageDown()

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
			// Reset position history as currentFileIndex changes.
			m.positions = make(map[string]position)
			// Keep cursor at same place.
			filePath, ok := m.filePath()
			if !ok {
				return m, nil
			}
			m.prevName = filePath
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
		names[i] = fi.dirEntry.Name()
	}
	matches := fuzzy.Find(m.search, names)
	if len(matches) > 0 {
		m.matchedIndexes = matches[0].MatchedIndexes
		index := matches[0].Index
		m.setCurrentFileIndex(index)
	}
	m.saveCursorPosition()
}

func (m *model) View() string {
	if m.showHelp {
		out := &Builder{}
		out.WriteString(bar.Render("help") + "\n\n")
		usage(out, false)
		return out.String()
	}

	width := m.termWidth
	if m.previewMode {
		width = m.termWidth / 2
	}
	height := m.listHeight()

	if m.columns == -1 {
		m.columns, m.rows = calculateColumnCount(m.extra.maxDislayNameLength, m.files, width, height)
	}
	if m.findPrevName {
		fileIndex, ok := findFileIndex(m.prevName, m.files)
		if ok {
			m.setCurrentFileIndex(fileIndex)
		}
		m.findPrevName = false
		m.saveCursorPosition()
	}

	// After we have updated offset and saved cursor position, we can
	// preview currently selected file.
	m.preview()

	// Get output rows width before coloring.
	outputWidth := width
	// TODO: Update/fix this (if it's really needed):
	// outputWidth := strlen(path.Base(m.path)) // Use current dir name as default.
	// if m.previewMode {
	// 	row := make([]string, m.columns)
	// 	for i := 0; i < m.columns; i++ {
	// 		if len(m.fileList[i]) > 0 {
	// 			row[i] = createDisplayName(&m.fileList[i][0], true, false, nil)
	// 		} else {
	// 			outputWidth = width
	// 		}
	// 	}
	// 	outputWidth = max(outputWidth, strlen(Join(row, separator)))
	// } else {
	// 	outputWidth = width
	// }

	// Let's add colors to file names.
	output := make([]string, m.rows)
	for r := 0; r < m.rows; r++ {
		row := make([]string, m.columns)
		for c := 0; c < m.columns; c++ {
			fileIndex := m.firstFileIndex + (c * m.rows) + r
			if fileIndex >= len(m.files) {
				break
			}
			if fileIndex == m.currenFileIndex {
				if m.deleteCurrentFile {
					row[c] = createDisplayName(&m.files[fileIndex], true, true, &danger)
				} else {
					row[c] = createDisplayName(&m.files[fileIndex], true, true, &cursor)
				}
			} else {
				row[c] = createDisplayName(&m.files[fileIndex], true, true, nil)
			}
		}
		output[r] = Join(row, separator)
	}

	// Preview pane.
	fileName, _ := m.currentFileName()
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

	if m.showStatusBar() {
		// Only show one status bar.
		// TODO: Show most recent status bar.
		if len(m.toBeDeleted) > 0 {
			toDelete := m.toBeDeleted[len(m.toBeDeleted)-1]
			timeLeft := int(toDelete.at.Sub(time.Now()).Seconds())
			deleteBar := fmt.Sprintf("%v deleted. (u)ndo %v", path.Base(toDelete.path), timeLeft)
			main += "\n" + danger.Render(deleteBar)
		} else if m.yankedFilePath != "" {
			yankBar := fmt.Sprintf("copied: %v", m.yankedFilePath)
			main += "\n" + bar.Render(yankBar)
		} else if m.statusBar != nil {
			f, ok := m.currentFile()
			if ok {
				env := Env{
					DirPath:     m.path,
					Files:       nil,
					CurrentFile: f.dirEntry,
				}
				statusBar, err := expr.Run(m.statusBar, env)
				if err != nil {
					main += "\n" + err.Error()
				} else {
					main += "\n" + bar.Render(fmt.Sprintf("%v", statusBar))
				}
			}
		}
	}

	view := main
	if m.previewMode {
		previewStyle := previewPlain
		if withBorder {
			previewStyle = previewSplit
		}
		view = lipgloss.JoinHorizontal(
			lipgloss.Top,
			main,
			previewStyle.
				MaxHeight(m.termHeight).
				Render(previewPane),
		)
	}

	view = extraView(m, view)

	if m.quitting {
		view += "\n" // Keep the last line from disappearing.
	}

	return view
}

func (m *model) moveUp() {
	m.setCurrentFileIndex(m.currenFileIndex - 1)
}

func (m *model) moveDown() {
	m.setCurrentFileIndex(m.currenFileIndex + 1)
}

func (m *model) moveLeft() {
	m.setCurrentFileIndex(m.currenFileIndex - m.rows)
}

func (m *model) moveRight() {
	m.setCurrentFileIndex(m.currenFileIndex + m.rows)
}

func (m *model) moveStart() {
	m.setCurrentFileIndex(0)
}

func (m *model) moveEnd() {
	m.setCurrentFileIndex(len(m.files) - 1)
}

func (m *model) movePageUp() {
	screenCapacity := m.columns * m.rows
	m.setCurrentFileIndex(m.currenFileIndex - (screenCapacity - 1))
}

func (m *model) movePageDown() {
	screenCapacity := m.columns * m.rows
	m.setCurrentFileIndex(m.currenFileIndex + (screenCapacity - 1))
}
func (m *model) setCurrentFileIndex(index int) {
	m.currenFileIndex = index
	m.sanitizeFirstIndexAndCurrentIndex()
}

func (m *model) sanitizeFirstIndexAndCurrentIndex() {
	if m.currenFileIndex < 0 {
		m.currenFileIndex = 0
	}
	if m.currenFileIndex < m.firstFileIndex {
		m.firstFileIndex = m.currenFileIndex
	}

	if m.currenFileIndex >= len(m.files) {
		m.currenFileIndex = len(m.files) - 1
	}
	screenCapacity := m.columns * m.rows
	lastScreenIndex := m.firstFileIndex + (screenCapacity - 1)
	if m.currenFileIndex > lastScreenIndex {
		m.firstFileIndex = m.currenFileIndex - (screenCapacity - 1)
	}
}

func (m *model) list() {
	m.columns = -1
	m.rows = -1
	m.files, m.err = listDir(m, m.path, 0)
}

func (m *model) listHeight() int {
	h := m.termHeight - 1 // Subtract 1 for location bar.
	if m.showStatusBar() {
		h--
	}
	return h
}

func (m *model) showStatusBar() bool {
	if len(m.toBeDeleted) > 0 {
		return true
	}
	if m.yankedFilePath != "" {
		return true
	}
	if m.statusBar != nil {
		return true
	}
	return false
}

func (m *model) saveCursorPosition() {
	m.positions[m.path] = position{
		cursorFileIndex:         m.currenFileIndex,
		firstDisplayedFileIndex: m.firstFileIndex,
	}
}

func (m *model) currentFile() (*fileEntry, bool) {
	i := m.currenFileIndex
	if i >= len(m.files) || i < 0 {
		return nil, false
	}
	return &m.files[i], true
}

func (m *model) currentFileName() (string, bool) {
	f, ok := m.currentFile()
	if !ok {
		return "", false
	}
	return f.dirEntry.Name(), true
}

func (m *model) filePath() (string, bool) {
	fileEntry, ok := m.currentFile()
	if !ok {
		return "", false
	}
	return path.Join(fileEntry.dirPath, fileEntry.dirEntry.Name()), true
}

func (m *model) open() tea.Cmd {
	filePath, ok := m.filePath()
	if !ok {
		return nil
	}

	var commandString string
	if commandString, ok = openWith[extension(filePath)]; ok {
	} else {
		commandString = lookup([]string{"WALK_EDITOR", "EDITOR"}, "less")
	}

	commandSlice := append(Split(commandString, " "), filePath)
	execCmd := exec.Command(commandSlice[0], commandSlice[1:]...)
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

	width := m.termWidth / 2
	height := m.termHeight - 1 // Subtract 1 for name bar.

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

	if isImage(filePath) {
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
		previewContent := leaveOnlyAscii(content)
		m.previewContent = previewContent

		if withHighlight {
			var buf bytes.Buffer
			lexer := lexers.Match(filePath)
			if lexer == nil {
				lexer = lexers.Fallback
			}
			if err := quick.Highlight(&buf, previewContent, lexer.Config().Name, "terminal256", "friendly"); err == nil {
				m.previewContent = buf.String()
			}
		}
	default:
		m.previewContent, ok = generateFilePreview(filePath)
		if !ok {
			m.previewContent = warning.Render("No preview available")
		}
	}
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
	if len(files) > longListLimit {
		columns = longListColumns
	}

	// Fifteenth column is enough for everyone.
	if columns > maxColumns {
		columns = maxColumns
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
