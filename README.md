# 🥾 walk

<p align="center">
  <br>
  <img src=".github/images/demo.gif" width="600" alt="walk demo">
  <br>
</p>

**Walk** — a terminal navigator.

Why another terminal navigator? I wanted something simple and minimalistic.
Something to help me with faster navigation in the filesystem; a `cd` and `ls`
replacement. So I build **walk**. It allows for quick navigation with fuzzy
searching, `cd` integration is quite simple. And you can open `vim` right from
the walk. That's it.

## Install

```
brew install walk
```

```
pkg_add walk
```

```
go install github.com/antonmedv/walk@latest
```

Or download [prebuild binaries](https://github.com/antonmedv/walk/releases).

Put the next function into the **.bashrc** or a similar config:

<table>
<tr>
  <th> Bash/Zsh </th>
  <th> Fish </th>
  <th> PowerShell </th>
</tr>
<tr>
<td>

```bash
function lk {
  cd "$(walk "$@")"
}
```

</td>
<td>

```fish
function lk
  set loc (walk $argv); and cd $loc;
end
```

</td>
<td>

```powershell
function lk() {
  cd $(walk $args)
}
```

</td>
</tr>
</table>


Now use `lk` command to start walking.

### Preview mode

Press `Space` to toggle preview mode.

<img src=".github/images/preview-mode.gif" width="600" alt="Walk Preview Mode">

### Delete file or directory

Press `dd` to delete file or directory. Press `u` to undo.

<img src=".github/images/rm-demo.gif" width="600" alt="Walk Deletes a File">

### Display icons

Install [Nerd Fonts](https://www.nerdfonts.com) and add `--icons` flag.

<img src=".github/images/demo-icons.gif" width="600" alt="Walk Icons Support">

### Image preview

No additional setup is required.

<img src=".github/images/images-mode.gif" width="600" alt="Walk Image Preview">

## Usage

| Key binding      | Description        |
|------------------|--------------------|
| `arrows`, `hjkl` | Move cursor        |
| `enter`          | Enter directory    |
| `backspace`      | Exit directory     |
| `space`          | Toggle preview     |
| `esc`, `q`       | Exit with cd       |
| `ctrl+c`         | Exit without cd    |
| `/`              | Fuzzy search       |
| `d`, `delete`    | Delete file or dir |
| `y`              | yank current dir   |
| `.`              | Hide hidden files  |

## Configuration

The `EDITOR` or `WALK_EDITOR` environment variable used for opening files from
the walk.

```bash
export EDITOR=vim
```

The `WALK_REMOVE_CMD` environment variable can be used to specify a command to
be used to remove files. This is useful if you want to use a different
command to remove files than the default `rm`.

```bash
export WALK_REMOVE_CMD=trash
```

Flags can be used to change the default behavior of the program.

| Flag         | Description                |
|--------------|----------------------------|
| `--icons`    | Show icons                 |
| `--dir-only` | Show dirs only             |
| `--preview`  | Start with preview mode on |
| `--fuzzy`    | Start with fuzzy search on |

## License

[MIT](LICENSE)
