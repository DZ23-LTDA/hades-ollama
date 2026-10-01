#!/bin/sh
# Ollama Full source installer.
#
# This script intentionally builds the checked-out fork from source. It does
# not download Ollama upstream binaries, installers, models, or Docker images.
# Signed release artifacts are a separate, currently unsupported workflow.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
install_dir=${OLLAMA_INSTALL_DIR:-"$HOME/.local/bin"}
binary_name=${OLLAMA_BINARY_NAME:-ollama-full}
output="$install_dir/$binary_name"

if ! command -v go >/dev/null 2>&1; then
  echo "ERROR: Go is required to build Ollama Full from source." >&2
  exit 1
fi
if [ ! -f "$repo_root/go.mod" ]; then
  echo "ERROR: run this script from a checked-out Ollama Full repository." >&2
  exit 1
fi
case "$install_dir" in
  /*) ;;
  *) echo "ERROR: OLLAMA_INSTALL_DIR must be an absolute path." >&2; exit 1 ;;
esac

mkdir -p "$install_dir"
printf '%s\n' "Building Ollama Full from $repo_root"
(cd "$repo_root" && go build -trimpath -o "$output" .)
chmod 0755 "$output"
printf '%s\n' "Installed source-built binary at $output"
printf '%s\n' "Run it with: $output serve"
printf '%s\n' "No signed installer, release binary, model download, or upstream service was used."
