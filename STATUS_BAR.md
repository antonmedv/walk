# Status Bar

Status bar is a program that is executed every time the cursor is moved. It is used to display additional information
about the current file.

For example, to simulate the `ls -l` command, you can use the following status bar program:

```bash
export WALK_STATUS_BAR='[Mode(), Owner(), Size() | PadLeft(7), ModTime() | PadLeft(12)] | join(" ")'
```

## Syntax

The syntax of the status bar program is based on the [expr](https://expr-lang.org/docs/language-definition) language.

## Variables

### `current_file`

Current file, type: `fs.DirEntry`

### `files`

List of files in the current directory, type: `[]fs.DirEntry`

## Functions

### `Sprintf(format, a...)`

Returns a formatted string.

### `PadLeft(s, n)`

Returns a string padded with spaces on the left.

### `PadRight(s, n)`

Returns a string padded with spaces on the right.

### `Size()`

Returns the size of the current file in human-readable format.

### `Mode()`

Returns the file mode of the current file. Like `drwxr-xr-x`.

### `ModTime()`

Returns the modification time of the current file in human-readable format.

### `Owner()`

Returns owner's username and group name.
