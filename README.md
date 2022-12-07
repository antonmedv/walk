# 🦙 llama

<p align="center">
  <br>
  <img src=".github/images/demo.gif" width="600" alt="Llama Demo">
  <br>
</p>

Llama — a terminal file manager.

Why another file manager? I wanted something simple and minimalistic.
Something to help me with faster navigation in the filesystem; a `cd` and `ls`
replacement. So I build "llama". It allows for quick navigation with fuzzy
searching. `cd` integration is quite simple. And you can open `vim` right from
the llama. That's it. As simple and dumb as a llama.

## Install

```
brew install llama
```

```
snap install llama
```

```
pkg_add llama
```

```
go install github.com/antonmedv/llama@latest
```

Or download [prebuild binaries](https://github.com/antonmedv/llama/releases).

Put the next function into the **.bashrc** or a similar config:

<table>
<tr>
  <th> Bash </th>
  <th> Fish </th>
  <th> PowerShell </th>
</tr>
<tr>
<td>

```bash
function ll {
  cd "$(llama "$@")"
}
```

</td>
<td>

```fish
function ll
  set loc (llama $argv); and cd $loc;
end
```

</td>
<td>

```powershell
function ll() {
  cd "$(llama $args)"
}
```

</td>
</tr>
</table>


Note: we need a such helper as the child process can't modify the working
directory of the parent process.

## Usage

| Key binding      | Description        |
|------------------|--------------------|
| `Arrows`, `hjkl` | Move cursor        |
| `Enter`          | Enter directory    |
| `Backspace`      | Exit directory     |
| `Space`          | Toggle preview     |
| `Esc`            | Exit with cd       |
| `Ctrl+C`         | Exit without cd    |
| `/`              | Fuzzy search       |
| `dd`             | Delete file or dir |

The `EDITOR` or `LLAMA_EDITOR` environment variable used for opening files from
the llama.

```bash
export EDITOR=vim
```

Preview mode:

<img src=".github/images/preview-mode.gif" width="600" alt="Llama Preview Mode">

Delete file or directory:

<img src=".github/images/rm-demo.gif" width="600" alt="Llama Deletes a File">


## License

[MIT](LICENSE)
