#!/bin/sh
set -euo pipefail

owner='antonmedv'
name='walk'
version='v1.13.0'

os=$(uname -s | tr '[:upper:]' '[:lower:]')
machine=$(uname -m)

case $os in
linux | darwin)
  ext=''
  ;;
windows)
  os=windows
  ext='.exe'
  ;;
*)
  echo "Unsupported OS: $os" >&2
  exit 1
  ;;
esac

case $machine in
x86_64 | amd64)
  arch=amd64
  ;;
arm64 | aarch64)
  arch=arm64
  ;;
*)
  echo "Unsupported architecture: $machine" >&2
  exit 1
  ;;
esac

asset="${name}_${os}_${arch}${ext}"
echo "Installing ${name} ${version} (${asset})"
curl -Lfs "https://github.com/${owner}/${name}/releases/download/${version}/${asset}" -o "${name}"

chmod +x "${name}"
sudo mv "${name}" "/usr/local/bin/${name}"

printf "Run 'sudo mv ${name} /usr/local/bin/${name}'? [Y/n]: "
read -r answer
case "$answer" in
  [Yy]* | "" )
    sudo mv "$bin_path" "/usr/local/bin/${name}"
    echo "${name} installed successfully!"
    ;;
esac
