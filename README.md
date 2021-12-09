# llama

<p align="center"><img src="https://medv.io/assets/llama/llama.gif" width="520" align="center" alt="Llama Screenshot"></p>

Llama — a terminal file manager.

## Install

```
go get github.com/antonmedv/llama
```

Or download [prebuild binaries](https://github.com/antonmedv/llama/releases).

Put the next function into **~/.bashrc**:

```bash
function ll {
  llama "$@" 2> /tmp/path
  if [[ -d `cat /tmp/path` ]]; then
    cd `cat /tmp/path`
  fi
}
```

Use **ll** to navigate the filesystem. Note: we need a such helper as the child
process can't modify the working directory of the parent process.

## Usage

| Key binding | Description     |
|-------------|-----------------|
| `Arrows`    | Move cursor     |
| `Enter`     | Enter directory |
| `Backspace` | Exit directory  |
| `[A-Z]`     | Fuzzy search    |
| `Esc`       | Exit with cd    |
| `Ctrl+C`    | Exit with noop  |

Use `LLAMA_EDITOR` environment variable to specify program for opening files.

```bash
export LLAMA_EDITOR=vim
```

## License

[MIT](LICENSE)
