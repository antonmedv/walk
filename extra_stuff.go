package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/antonmedv/walk/overlay"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/bubbles/key"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

type appConfig struct {
	Keys           *keysConfig            `json:"keys,omitempty"`
	Colors         *colorsConfig          `json:"colors,omitempty"`
	Layout         *layoutConfig          `json:"layout,omitempty"`
	Editor         *string                `json:"editor,omitempty"`
	CustomCommands *[]customCommandConfig `json:"customCommands,omitempty"`
}

type keysConfig struct {
	ForceQuit *string `json:"forceQuit,omitempty"`
	Quit      *string `json:"quit,omitempty"`
	QuitQ     *string `json:"quitQ,omitempty"`
	UpDir     *string `json:"upDir,omitempty"`
	OpenDir   *string `json:"openDir,omitempty"`
	OpenTree  *string `json:"openTree,omitempty"`
	CloseTree *string `json:"closeTree,omitempty"`
	Back      *string `json:"back,omitempty"`
	Select    *string `json:"select,omitempty"`
}

type colorsConfig struct {
	Main         *colorConfig `json:"main,omitempty"`
	Cursor       *colorConfig `json:"cursor,omitempty"`
	StatusBar    *colorConfig `json:"statusBar,omitempty"`
	Directory    *colorConfig `json:"directory,omitempty"`
	Symlink      *colorConfig `json:"symlink,omitempty"`
	Executable   *colorConfig `json:"executable,omitempty"`
	SelectedFile *colorConfig `json:"selectedFile,omitempty"`
}

type layoutConfig struct {
	StatusBar       *string `json:"statusBar,omitempty"`
	FileInfo        *string `json:"fileInfo,omitempty"`
	MaxColumns      *int    `json:"maxColumns,omitempty"`
	LongListColumns *int    `json:"longListColumns,omitempty"`
	LongListLimit   *int    `json:"longListLimit,omitempty"`
	ColumnSeparator *string `json:"columnSeparator,omitempty"`
	SelectionMark   *string `json:"selectionMark,omitempty"`
}

type colorConfig struct {
	Foreground *string `json:"fg,omitempty"`
	Background *string `json:"bg,omitempty"`
}

type customCommandConfig struct {
	Description      string `json:"description"`
	Key              string `json:"key,omitempty"`
	Command          string `json:"cmd"`
	Prompt           string `json:"prompt,omitempty"`
	CompletedMessage string `json:"completedMessage,omitempty"`
	Args             string `json:"args"`
}

// Represents an file entry in the file system.
type fileEntry struct {
	dirPath    string      // The path of the parent directory.
	treeDepth  int         // Depth relative to model.path (i.e. the current "root" directory).
	dirEntry   os.DirEntry // The dir entry of the file.
	icon       string
	brief      string
	padding    string
	isSelected bool
}

const (
	argTypeCurrentDir            int = 0
	argTypeCurrentFile           int = 1
	argTypeSelectedFiles         int = 2
	argTypeSelectedOrCurrentFile int = 3
	argTypeInput                 int = 4
)

type customCommand struct {
	description      string
	key              key.Binding
	cmd              string
	prompt           string
	completedMessage string
	argType          int
}

type askInputForCommandMsg struct{}
type textInputAcceptedMsg struct{}
type cmdMenuAcceptedMsg struct{}

type extraModel struct {
	customCommandsMenu table.Model // Menu of custom commands.

	textInput    textinput.Model // For asking text input from the user.
	textInputCmd *customCommand  // A command that's waiting for the text input.

	statusMessage string // A status message that's shown at the bottom.

	maxDislayNameLength int // The longest display name in the file list.

	// Set of opened directories in the tree view.
	// We use this map as a "set". The bool value is ignored.
	openTreeDirs map[string]bool
}

var (
	config appConfig

	keyEnter = key.NewBinding(key.WithKeys("enter"))
	keyEsc   = key.NewBinding(key.WithKeys("esc"))

	keyUpDir     = key.NewBinding(key.WithKeys("ctrl+left"))
	keyOpenDir   = key.NewBinding(key.WithKeys("ctrl+right"))
	keySelect    = key.NewBinding(key.WithKeys("insert"))
	keyCmdMenu   = key.NewBinding(key.WithKeys("f2"))
	keyOpenTree  key.Binding
	keyCloseTree key.Binding

	customCommands []customCommand

	directoryStyle = lipgloss.NewStyle().Background(lipgloss.NoColor{}).Foreground(lipgloss.NoColor{})
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.NoColor{}).Foreground(lipgloss.NoColor{})
	symlinkStyle   = lipgloss.NewStyle().Background(lipgloss.NoColor{}).Foreground(lipgloss.NoColor{})
	exeStyle       = lipgloss.NewStyle().Background(lipgloss.NoColor{}).Foreground(lipgloss.NoColor{})

	maxColumns      = 15
	longListLimit   = 100
	longListColumns = 4
	dirsFirst       = false
	selectionMark   = "+"
	symlinkMark     = "@"
	exeMark         = "*"
	treeIndent      = "    "
	moveWrapsAround = false
)

func initExtra(m *model) {
	// Init model
	m.extra.openTreeDirs = make(map[string]bool)

	// Logging
	initLogToFile()

	config = readConfig()

	// Keys
	if config.Keys != nil {
		if config.Keys.ForceQuit != nil {
			keyForceQuit = key.NewBinding(key.WithKeys(*config.Keys.ForceQuit))
		}
		if config.Keys.Quit != nil {
			keyQuit = key.NewBinding(key.WithKeys(*config.Keys.Quit))
		}
		if config.Keys.QuitQ != nil {
			keyQuitQ = key.NewBinding(key.WithKeys(*config.Keys.QuitQ))
		}
		if config.Keys.UpDir != nil {
			keyUpDir = key.NewBinding(key.WithKeys(*config.Keys.UpDir))
		}
		if config.Keys.OpenDir != nil {
			keyOpenDir = key.NewBinding(key.WithKeys(*config.Keys.OpenDir))
		}
		if config.Keys.OpenTree != nil {
			keyOpenTree = key.NewBinding(key.WithKeys(*config.Keys.OpenTree))
		}
		if config.Keys.CloseTree != nil {
			keyCloseTree = key.NewBinding(key.WithKeys(*config.Keys.CloseTree))
		}
		if config.Keys.Back != nil {
			keyBack = key.NewBinding(key.WithKeys(*config.Keys.Back))
		}
		if config.Keys.Select != nil {
			keySelect = key.NewBinding(key.WithKeys(*config.Keys.Select))
		}
	}

	// Style (colors)
	if config.Colors != nil {
		initStyleFromConfig(&cursor, config.Colors.Cursor)
		initStyleFromConfig(&bar, config.Colors.StatusBar)
		initStyleFromConfig(&directoryStyle, config.Colors.Directory)
		initStyleFromConfig(&symlinkStyle, config.Colors.Symlink)
		initStyleFromConfig(&exeStyle, config.Colors.Executable)
		initStyleFromConfig(&selectedStyle, config.Colors.SelectedFile)
	}

	// Layout
	if config.Layout != nil {
		if config.Layout.StatusBar != nil {
			m.statusBar = compile(*config.Layout.StatusBar)
		}
		if config.Layout.FileInfo != nil {
			m.fileInfoProg = compile(*config.Layout.FileInfo)
		}
		if config.Layout.MaxColumns != nil {
			maxColumns = *config.Layout.MaxColumns
		}
		if config.Layout.LongListColumns != nil {
			longListColumns = *config.Layout.LongListColumns
		}
		if config.Layout.LongListLimit != nil {
			longListLimit = *config.Layout.LongListLimit
		}
		if config.Layout.ColumnSeparator != nil && len(*config.Layout.SelectionMark) > 0 {
			selectionMark = (*config.Layout.SelectionMark)[0:1]
		}
		if config.Layout.ColumnSeparator != nil && len(*config.Layout.ColumnSeparator) > 0 {
			separator = " " + (*config.Layout.ColumnSeparator)[0:1] + "  "
		}
	}

	// Custom commands
	initCustomCommands(m, &config)

	// Text input
	ti := textinput.New()
	ti.CharLimit = 156
	ti.Width = 40
	m.extra.textInput = ti
}

func initCustomCommands(m *model, config *appConfig) {

	menuRows := []table.Row{}

	if config.CustomCommands != nil {
		for _, c := range *config.CustomCommands {
			menuRow := []string{}
			menuRow = append(menuRow, c.Description)
			menuRow = append(menuRow, "") // key

			var customCommand customCommand
			customCommand.description = c.Description
			if len(c.Key) != 0 {
				menuRow[1] = c.Key
				customCommand.key = key.NewBinding(key.WithKeys(c.Key))
			}
			customCommand.cmd = c.Command
			customCommand.prompt = c.Prompt
			customCommand.completedMessage = c.CompletedMessage
			switch c.Args {
			case "currentDir":
				customCommand.argType = argTypeCurrentDir
			case "currentFile":
				customCommand.argType = argTypeCurrentFile
			case "selectedFiles":
				customCommand.argType = argTypeSelectedFiles
			case "selectedOrCurrentFile":
				customCommand.argType = argTypeSelectedOrCurrentFile
			case "input":
				customCommand.argType = argTypeInput
			default:
				log.Println("Invalid command args: ", c.Args)
			}

			customCommands = append(customCommands, customCommand)
			menuRows = append(menuRows, menuRow)
		}
	}

	menuColumns := []table.Column{
		{Title: "Command", Width: 20},
		{Title: "Key", Width: 20},
	}
	cmdMenu := table.New(
		table.WithColumns(menuColumns),
		table.WithRows(menuRows),
		table.WithHeight(7),
	)
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(false)
	s.Selected = cursor
	cmdMenu.SetStyles(s)

	m.extra.customCommandsMenu = cmdMenu
}

func ensureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		return nil
	}
	return os.MkdirAll(dirPath, os.ModePerm)
}

func readConfig() appConfig {
	var config appConfig

	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		log.Println("HOME env var is not defined.")
		return config
	}

	jsonPath := path.Join(homeDir, ".config", "walk.json")

	if err := ensureDirExists(path.Dir(jsonPath)); err != nil {
		log.Println("Cannot create directory for ", jsonPath)
		return config
	}

	jsonFile, err := os.Open(jsonPath)
	if err != nil {
		log.Println("Cannot read config file: ", jsonPath)
		return config
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		log.Printf("Cannot parse %s: %s", jsonPath, err)
	}

	return config
}

func initLogToFile() {
	homeDir, ok := os.LookupEnv("HOME")
	if !ok {
		log.Println("HOME env var is not defined.")
		panic(ok)
	}

	logPath := path.Join(homeDir, ".cache", "walk.log")

	if err := ensureDirExists(path.Dir(logPath)); err != nil {
		log.Println("Cannot create directory for ", logPath)
		panic(err)
	}

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
}

func initStyleFromConfig(style *lipgloss.Style, colorConfig *colorConfig) {
	if colorConfig == nil {
		return
	}
	*style = lipgloss.NewStyle()
	if colorConfig.Foreground != nil {
		*style = (*style).Foreground(lipgloss.Color(*colorConfig.Foreground))
	}
	if colorConfig.Background != nil {
		*style = (*style).Background(lipgloss.Color(*colorConfig.Background))
	}
}

func executeCommand(m *model, command *customCommand, filePaths ...string) tea.Cmd {
	commandSlice := append(strings.Split(command.cmd, " "), filePaths...)
	execCmd := exec.Command(commandSlice[0], commandSlice[1:]...)
	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		// Note: we could return a message here indicating that editing is
		// finished and altering our application about any errors. For now,
		// however, that's not necessary.

		if len(command.completedMessage) > 0 {
			m.extra.statusMessage = command.completedMessage
		}

		// Refresh the list. Files may have been created/deleted.
		m.list()

		return nil
	})
}

func updateTextInput(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if !m.extra.textInput.Focused() {
		return m, nil, false
	}

	// Handle keyboard input.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keyEsc) {
			m.extra.textInput.Blur()
			return m, nil, true
		} else if key.Matches(msg, keyEnter) {
			m.extra.textInput.Blur()
			return m, func() tea.Msg { return textInputAcceptedMsg{} }, true
		}
	}

	var cmd tea.Cmd
	m.extra.textInput, cmd = m.extra.textInput.Update(msg)
	return m, cmd, true
}

func updateCmdMenu(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if !m.extra.customCommandsMenu.Focused() {
		return m, nil, false
	}

	// Handle keyboard input.
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keyEsc) {
			m.extra.customCommandsMenu.Blur()
			return m, nil, true
		} else if key.Matches(msg, keyEnter) {
			m.extra.customCommandsMenu.Blur()
			return m, func() tea.Msg { return cmdMenuAcceptedMsg{} }, true
		}
	}

	var cmd tea.Cmd
	m.extra.customCommandsMenu, cmd = m.extra.customCommandsMenu.Update(msg)
	return m, cmd, true
}

func executeCustomCommand(m *model, customCommand *customCommand) (tea.Cmd, bool) {
	if customCommand == nil {
		return nil, false
	}
	if customCommand.cmd == "" {
		return nil, false
	}

	switch customCommand.argType {
	case argTypeCurrentDir:
		{
			if fileEntry, ok := m.currentFile(); ok {
				return executeCommand(m, customCommand, fileEntry.dirPath), true
			} else if len(m.files) == 0 {
				return executeCommand(m, customCommand, m.path), true
			} else {
				return nil, true
			}
		}
	case argTypeCurrentFile:
		{
			currentFilePath, ok := m.filePath()
			if !ok {
				return nil, true
			}
			return executeCommand(m, customCommand, currentFilePath), true
		}
	case argTypeSelectedFiles:
		{
			selectedFilePaths := getSelectedFilePaths(m)
			if len(selectedFilePaths) == 0 {
				return nil, true
			}
			return executeCommand(m, customCommand, selectedFilePaths...), true
		}
	case argTypeSelectedOrCurrentFile:
		{
			selectedFilePaths := getSelectedFilePaths(m)
			if len(selectedFilePaths) == 0 {
				currentFilePath, ok := m.filePath()
				if !ok {
					return nil, true
				}
				selectedFilePaths = append(selectedFilePaths, currentFilePath)
			}
			return executeCommand(m, customCommand, selectedFilePaths...), true
		}
	case argTypeInput:
		{
			m.extra.textInputCmd = customCommand
			return func() tea.Msg { return askInputForCommandMsg{} }, true
		}
	}
	log.Println("Invalid command arg type: ", customCommand.argType)
	return nil, false
}

func enterDirectory(m *model, dirPath string) {
	isEnteringDirectSubDir := m.path == path.Dir(dirPath)

	// Enter subdirectory.
	m.path = dirPath
	if p, ok := m.positions[m.path]; ok {
		m.currenFileIndex = p.cursorFileIndex
		m.firstFileIndex = p.firstDisplayedFileIndex
	} else {
		m.currenFileIndex = 0
		m.firstFileIndex = 0
	}

	m.search = ""
	m.searchMode = false

	if !isEnteringDirectSubDir {
		// If not a direct subdirectory (e.g. we enter an opened tree-view directory),
		// then it takes too much effort to update the model.poistions.
		// So let's just reset it.
		clear(m.positions)
	}

	clear(m.extra.openTreeDirs)
	m.list()
}

// Returns true if the key was handled.
func keyMsgHandler(m *model, msg tea.KeyMsg) (tea.Cmd, bool) {
	if key.Matches(msg, keyOpenDir) {
		filePath, ok := m.filePath()
		if !ok {
			return nil, false
		}
		if fi := fileInfo(filePath); fi.IsDir() {
			enterDirectory(m, filePath)
			return nil, true
		}
	}

	if key.Matches(msg, keyOpenTree) {
		fileEntry, ok := m.currentFile()
		if !ok {
			return nil, false
		}
		if fileEntry.dirEntry.IsDir() {
			filePath := path.Join(fileEntry.dirPath, fileEntry.dirEntry.Name())
			m.extra.openTreeDirs[filePath] = true
			m.list()
			return nil, true
		}
	}

	if key.Matches(msg, keyCloseTree) {
		fileEntry, ok := m.currentFile()
		if !ok {
			return nil, false
		}
		filePath := path.Join(fileEntry.dirPath, fileEntry.dirEntry.Name())
		if _, isOpen := m.extra.openTreeDirs[filePath]; isOpen {
			delete(m.extra.openTreeDirs, filePath)
		} else {
			// This entry is not open. Maybe it's a file or directory that's not open.
			// Close the parent dir.
			parentDirPath := path.Dir(filePath)
			// The cursor is on a file that will disappear.
			// After closing, let's put the cursor on the parent dir.
			m.findPrevName = true
			m.prevName = parentDirPath
			delete(m.extra.openTreeDirs, parentDirPath)
		}
		m.list()
		return nil, true
	}

	if key.Matches(msg, keySelect) {
		m.files[m.currenFileIndex].isSelected = !m.files[m.currenFileIndex].isSelected
		m.moveDown()
		return nil, true
	}

	if key.Matches(msg, keyCmdMenu) {
		m.extra.customCommandsMenu.SetCursor(0)
		m.extra.customCommandsMenu.Focus()
		return nil, true
	}

	for _, customCommand := range customCommands {
		if key.Matches(msg, customCommand.key) {
			if cmd, handled := executeCustomCommand(m, &customCommand); handled {
				return cmd, true
			}
		}
	}

	return nil, false
}

func extraView(m *model, view string) string {
	dialogStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

	if m.extra.textInput.Focused() {
		view = overlay.PlaceOverlay(5, 1, dialogStyle.Render(m.extra.textInput.View()), view)
		//view += "\n" + m.extra.textInput.View()
	}

	if len(m.extra.statusMessage) > 0 {
		view += "\n" + bar.Render(m.extra.statusMessage)
	}

	if m.extra.customCommandsMenu.Focused() {
		view = overlay.PlaceOverlay(5, 1, dialogStyle.Render(m.extra.customCommandsMenu.View()), view)
	}

	return view
}

func extraUpdate(m *model, msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	if mm, cmd, handled := updateTextInput(m, msg); handled {
		return mm, cmd, handled
	}

	if mm, cmd, handled := updateCmdMenu(m, msg); handled {
		return mm, cmd, handled
	}

	switch msg := msg.(type) {
	case askInputForCommandMsg:
		if len(m.extra.textInputCmd.prompt) > 0 {
			m.extra.textInput.Prompt = m.extra.textInputCmd.prompt
		} else {
			m.extra.textInput.Prompt = "Enter input: "
		}
		m.extra.textInput.SetValue("")
		m.extra.textInput.Focus()
		return m, nil, true
	case textInputAcceptedMsg:
		{
			inputText := strings.TrimSpace(m.extra.textInput.Value())
			if len(inputText) > 0 && m.extra.textInputCmd != nil {
				if currentFile, ok := m.currentFile(); ok {
					return m, executeCommand(m, m.extra.textInputCmd, currentFile.dirPath, inputText), true
				} else if len(m.files) == 0 {
					return m, executeCommand(m, m.extra.textInputCmd, m.path, inputText), true
				}
			}
			return m, nil, true
		}
	case cmdMenuAcceptedMsg:
		{
			cmdIndex := m.extra.customCommandsMenu.Cursor()
			if cmd, handled := executeCustomCommand(m, &customCommands[cmdIndex]); handled {
				return m, cmd, true
			}
		}
	case tea.KeyMsg:
		// Clear the status message when any key is pressed.
		m.extra.statusMessage = ""

		if cmd, handled := keyMsgHandler(m, msg); handled {
			// KeyMsg got handled.
			return m, cmd, true
		}
	}

	return nil, nil, false
}

func getSelectedFilePaths(m *model) []string {
	var filePaths []string
	for i := 0; i < len(m.files); i++ {
		if m.files[i].isSelected {
			filePaths = append(filePaths, path.Join(m.files[i].dirPath, m.files[i].dirEntry.Name()))
		}
	}
	return filePaths
}

func findFileIndex(filePath string, files []fileEntry) (int, bool) {
	for i := 0; i < len(files); i++ {
		if filePath == path.Join(files[i].dirPath, files[i].dirEntry.Name()) {
			return i, true
		}
	}
	return -1, false
}

func executeExprProgram(prog *vm.Program, dirPath string, file os.DirEntry) string {
	env := Env{
		DirPath:     dirPath,
		Files:       nil,
		CurrentFile: file,
	}
	result := ""
	output, err := expr.Run(prog, env)
	if err != nil {
		result = err.Error()
	} else {
		result = fmt.Sprintf("%v", output)
	}
	return result
}

func getFileMode(e os.DirEntry) os.FileMode {
	info, err := e.Info()
	if err != nil {
		return 0
	}
	return info.Mode()
}

// If withStyle is true but fileNameStyle is nil, then we automatically select a style for you.
func createDisplayName(fileEntry *fileEntry, withPadding bool, withStyle bool, fileNameStyle *lipgloss.Style) string {
	fileMode := getFileMode(fileEntry.dirEntry)

	if withStyle && fileNameStyle == nil {
		if fileEntry.isSelected {
			fileNameStyle = &selectedStyle
		} else if fileEntry.dirEntry.IsDir() {
			fileNameStyle = &directoryStyle
		} else if fileEntry.dirEntry.Type().Type()&fs.ModeSymlink != 0 {
			fileNameStyle = &symlinkStyle
		} else if fileMode&73 != 0 {
			fileNameStyle = &exeStyle
		}
	}

	displayName := ""
	displayName += fileEntry.brief
	if fileEntry.isSelected {
		displayName += " " + selectionMark
	} else {
		displayName += "  "
	}
	if len(fileEntry.icon) > 0 {
		displayName += " " + fileEntry.icon
	}

	fileNameIndent := strings.Repeat(treeIndent, fileEntry.treeDepth)
	if withStyle && fileNameStyle != nil {
		displayName += " " + fileNameIndent + fileNameStyle.Render(fileEntry.dirEntry.Name())
	} else {
		displayName += " " + fileNameIndent + fileEntry.dirEntry.Name()
	}

	if fileEntry.dirEntry.IsDir() {
		displayName += fileSeparator
	} else if fileEntry.dirEntry.Type().Type()&fs.ModeSymlink != 0 {
		displayName += symlinkMark
	} else if fileMode&73 != 0 {
		displayName += exeMark
	}
	if withPadding {
		displayName += fileEntry.padding
	}
	return displayName
}

func generateFilePreview(filePath string) (string, bool) {
	out, err := exec.Command("xxd", "-l", "102400", filePath).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func calculateColumnCount(maxNameSize int, files []fileEntry, width int, height int) (int, int) {
	fileCount := len(files)

	// If the directory is empty, return 0 columns.
	if fileCount == 0 {
		return 0, 0
	}

	// Start with the largest amount of rows and columns.
	rows := height
	columnWidth := maxNameSize + len(separator) // This is just an estimate. Bt it's good enough.
	columns := width / columnWidth

	// For large lists, limit the column count.
	if fileCount > longListLimit {
		columns = longListColumns
	}

	// Obey the max column count limit.
	if columns > maxColumns {
		columns = maxColumns
	}

	// Decrease the row count until the last column is full (or overflows),
	// but don't go lower than the third of the available height.
	//
	// rows*columns-fileCount = Empty rows in last column.
	for rows*columns-fileCount > 0 && rows > height/3 {
		rows--
	}

	// Decrease the column count until the last column is not empty.
	for rows*columns-fileCount >= rows && columns > 1 {
		columns--
	}

	// If we have one column and it's not full, reduce the row count
	// to make it full.
	if columns == 1 && rows > fileCount {
		rows = fileCount
	}

	//log.Println("files ", len(files), ", cols ", columns, ", rows ", rows)
	//log.Println("maxNameSize ", maxNameSize)
	return columns, rows
}

func listDir(m *model, dirPath string, depth int) ([]fileEntry, error) {
	// ReadDir already returns files and dirs sorted by filename.
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	// Filter out some files.
	filteredFiles := []os.DirEntry{}
filter:
	for _, file := range files {
		if m.hideHidden && strings.HasPrefix(file.Name(), ".") {
			continue filter
		}
		if dirOnly && !file.IsDir() {
			continue filter
		}
		for _, toDelete := range m.toBeDeleted {
			if path.Join(dirPath, file.Name()) == toDelete.path {
				continue filter
			}
		}
		filteredFiles = append(filteredFiles, file)
	}
	files = filteredFiles

	// Sort directories first.
	if dirsFirst {
		sort.Slice(files, func(i, j int) bool {
			a := files[i]
			b := files[j]
			return (a.IsDir() && !b.IsDir()) || (a.IsDir() == b.IsDir() && a.Name() < b.Name())
		})
	}

	// Create the final file entries.
	fileEntries := []fileEntry{}
	for i := 0; i < len(files); i++ {
		filePath := path.Join(dirPath, files[i].Name())

		fileEntry := fileEntry{}
		fileEntry.dirPath = dirPath
		fileEntry.treeDepth = depth
		fileEntry.dirEntry = files[i]

		if m.fileInfoProg != nil {
			fileEntry.brief = executeExprProgram(m.fileInfoProg, fileEntry.dirPath, fileEntry.dirEntry)
		}
		if showIcons {
			info, err := fileEntry.dirEntry.Info()
			if err == nil {
				icon := icons.getIcon(info)
				if icon != "" {
					fileEntry.icon = icon
				}
			}
		}

		fileEntries = append(fileEntries, fileEntry)

		// List sub-directories that are open as sub-trees.
		if fileEntry.dirEntry.IsDir() {
			if _, isOpenSubDir := m.extra.openTreeDirs[filePath]; isOpenSubDir {
				if subFiles, err := listDir(m, filePath, depth+1); err == nil {
					fileEntries = append(fileEntries, subFiles...)
				} else {
					log.Println("Cannot list directory: ", filePath)
				}
			}
		}
	}

	// Calculate max display name length, and paddings.
	if depth == 0 {
		maxNameSize := 0
		for _, fileEntry := range fileEntries {
			displayName := createDisplayName(&fileEntry, false, false, nil)
			nameSize := len(displayName)
			if nameSize > maxNameSize {
				maxNameSize = nameSize
			}
		}
		m.extra.maxDislayNameLength = maxNameSize
		for i := 0; i < len(fileEntries); i++ {
			displayName := createDisplayName(&fileEntries[i], false, false, nil)
			fileEntries[i].padding = strings.Repeat(" ", maxNameSize-strlen(displayName))
		}
	}

	return fileEntries, nil
}

/////////////////////////////////////////////////////////
// Status bar functions

func (e Env) FileName() string {
	name := e.CurrentFile.Name()
	target, err := os.Readlink(path.Join(e.DirPath, name))
	if err == nil {
		name += " -> " + target
	}
	return name
}

func (e Env) ModTime2(formatThisYear string, formatOtherYear string) string {
	info, err := e.CurrentFile.Info()
	if err != nil {
		return "???"
	}
	modTime := info.ModTime()
	now := time.Now()
	if modTime.Year() == now.Year() {
		return modTime.Format(formatThisYear)
	}
	return modTime.Format(formatOtherYear)
}
