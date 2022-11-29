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

## Getting Started
- [Installation](#install)
- [Usage](#usage)
- [Configuration](#configuration)
  - [Editor](#editor)
  - [Key Bindings](#key-bindings)
- [License](#license)

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
</tr>
<tr>
  <th colspan="2"> PowerShell </th>
</tr>
<tr>
<td colspan="2">

```powershell
function ll() {
  cd $(llama $args | Out-String -Stream | Select-Object -Last 1)
}
```
See [issues/30](https://github.com/antonmedv/llama/issues/30) for more details.

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

Preview mode:

<img src=".github/images/preview-mode.gif" width="600" alt="Llama Preview Mode">

Delete file or directory:

<img src=".github/images/rm-demo.gif" width="600" alt="Llama Deletes a File">

## Command-line options

* `--icons`: display icons

  To get the icons to render properly you should download and install a Nerd font from https://www.nerdfonts.com/.

  Then, select that font as your font for the terminal.

  <img src=".github/images/demo-icons.gif" width="600" alt="Llama Icons Support">

## Configuration
### Editor
The editor used for opening files from llama can be configured using the `EDITOR` or `LLAMA_EDITOR` environment variable.
<table>
<tr>
  <th> Bash </th>
  <th> Fish </th>
  <th> PowerShell </th>
</tr>
<tr>
<td>

```bash
export EDITOR=vim
```

</td>
<td>

```fish
set -gx EDITOR vim
```

</td>
<td>

```powershell
$env:EDITOR = "vim"
```

</td>
</tr>
</table>

### Key Bindings
Key bindings can be configured via json. By default, llama will search for a configuration file at `~\.config\llama\config.json` where `~` is the user's home directory, but this may be overridden using the `LLAMA_CONFIG` environment variable.

For example:
```json5
{
  "bindings": [
    // {
    //   action   : string
    //   keys     : string[]
    //   disabled : boolean
    //   help     : { key : string, desc: string }
    // }
    {
      "action": "keyQuit",
      "keys": [ "q", "tab" ],  // Bind keyQuit to activate with 'q' or 'tab' instead of `esc`
    },
    {
      "action": "keyDelete",
      "disabled": true         // Disable the keyDelete action
    }
  ]
}
```

Note that the `action` property must match one of the following actions, and that the configurations provided to the action will fully override the default configuration for that action. All actions are enabled by default.
| Action       | Default     |
| ------------ | ----------- |
| keyForceQuit | ctrl+c      |
| keyQuit      | esc         |
| keyOpen      | enter       |
| keyBack      | backspace   |
| keyUp        | up          |
| keyDown      | down        |
| keyLeft      | left        |
| keyRight     | right       |
| keyTop       | shift+up    |
| keyBottom    | shift+down  |
| keyLeftmost  | shift+left  |
| keyRightmost | shift+right |
| keyVimUp     | k           |
| keyVimDown   | j           |
| keyVimLeft   | h           |
| keyVimRight  | l           |
| keyVimTop    | g           |
| keyVimBottom | G           |
| keySearch    | /           |
| keyPreview   | space       |
| keyDelete    | d           |
| keyUndo      | u           |

## License

[MIT](LICENSE)
