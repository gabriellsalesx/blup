#!/usr/bin/env bash
# Installs a direct bundle into a disposable installation and checks its
# runtime libraries and publishing tools offline. Does not launch the GUI
# or load Studio accounts, notes or preferences.
set -euo pipefail

bundle="$(realpath "${1:?bundle .flatpak}")"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
export FLATPAK_USER_DIR="$work/installation"
export XDG_CONFIG_HOME="$work/config"
export XDG_DATA_HOME="$work/data"
export XDG_CACHE_HOME="$work/cache"
export XDG_STATE_HOME="$work/state"
if [[ -n "${BLUP_FLATPAK_RUNTIME_DIR:-}" ]]; then
  # Read the build's private runtime installation as the system installation
  # while keeping the test application in its separate temporary user one.
  export FLATPAK_SYSTEM_DIR="$(realpath "$BLUP_FLATPAK_RUNTIME_DIR")"
fi

# Dependencies must already be installed. This check never downloads a runtime
# and cannot replace a package in the person's normal Flatpak installation.
flatpak --user install --bundle --no-deps --no-related --noninteractive -y "$bundle"
flatpak --user run --unshare=network --command=sh io.getblup.Studio -ec '
  dependencies="$(ldd /app/bin/blup-studio 2>&1)" || {
    printf "%s\n" "$dependencies"
    exit 1
  }
  if printf "%s\n" "$dependencies" | grep -q "not found"; then
    printf "%s\n" "$dependencies"
    exit 1
  fi
  echo "Studio runtime libraries resolved"
  /app/bin/hugo version
  /app/bin/pagefind --version

  site="$(mktemp -d)"
  mkdir -p "$site/content" "$site/layouts/_default"
  printf "baseURL = \"https://example.org/\"\ntitle = \"Flatpak test\"\ndisableKinds = [\"home\", \"taxonomy\", \"term\", \"rss\", \"sitemap\"]\n" > "$site/hugo.toml"
  printf "<!doctype html><html lang=\"en\"><head><title>{{ .Title }}</title></head><body><main data-pagefind-body>{{ .Content }}</main></body></html>\n" > "$site/layouts/_default/single.html"
  printf -- "---\ntitle: Offline publishing test\n---\n# Notes from the sandbox\n\nA disposable test of the bundled publishing tools.\n" > "$site/content/check.md"
  /app/bin/hugo --source "$site" --destination "$site/public"
  test -s "$site/public/check/index.html"
  cd "$site"
  /app/bin/pagefind --site "$site/public"
  test -s "$site/public/pagefind/pagefind-entry.json"
  echo "Offline Hugo render and Pagefind index passed inside Flatpak"
'
