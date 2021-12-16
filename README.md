# 🦙 llama

<p align="center">
  <br>
  <img src="https://medv.io/assets/llama/llama.gif" width="520" alt="Llama Screenshot">
  <br>
</p>

Llama — a terminal file manager.

Why another file manager? I wanted something simple and minimalistic,
something to help me with faster navigation in the filesystem. A cd &
ls replacement. So I build "llama". It allows to quickly navigate
with fuzzy searching; cd integration is quite simple. Open vim right
from the llama. That's it. Simple and dumb as a llama.

## Install

```
go get github.com/antonmedv/llama
```

Or download [prebuild binaries](https://github.com/antonmedv/llama/releases).


Put the next function into **~/.bashrc** or **~/.config/fish/functions/ll.fish**:

<table>
<tr>
  <th> Bash </th>
  <th> Fish </th>
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
    cd (llama $argv);
end
```

</td>
</tr>
</table>


Note: we need a such helper as the child process can't modify the working directory of the parent process.

## Usage

| Key binding | Description     |
|-------------|-----------------|
| `Arrows`    | Move cursor     |
| `Enter`     | Enter directory |
| `Backspace` | Exit directory  |
| `[A-Z]`     | Fuzzy search    |
| `Esc`       | Exit with cd    |
| `Ctrl+C`    | Exit with noop  |

The `EDITOR` or `LLAMA_EDITOR` environment variable used for openning files from the llama.

```bash
export EDITOR=vim
```

### Vim Keybindings

Use `LLAMA_VIM_KEYBINDINGS` environment variable to enable vim keybindings.
```bash
export LLAMA_VIM_KEYBINDINGS=true
```
| Key binding | Description                 |
|-------------|-----------------------------|
| `hjkl`      | Move cursor                 |
| `/[A-Z]`    | Fuzzy search                |
| `Esc`       | Exit search or Exit with cd |

## License

[MIT](LICENSE)
