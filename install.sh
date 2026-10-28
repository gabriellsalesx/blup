#!/bin/sh
# Installs Blup on Linux and macOS:
#   curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
# Optional: BLUP_INSTALL_DIR (default ~/.local/bin), BLUP_VERSION (default latest).
set -eu

repo="gabriellsalesx/blup"
dir="${BLUP_INSTALL_DIR:-$HOME/.local/bin}"

say() { printf '%s\n' "$*"; }
fail() { say "blup: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || fail "'$1' is required / '$1' é necessário"; }

need curl
need tar

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) fail "unsupported system: $(uname -s) / sistema não suportado" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "unsupported processor: $(uname -m) / processador não suportado" ;;
esac

version="${BLUP_VERSION:-}"
if [ -z "$version" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n 1)"
  [ -n "$version" ] || fail "could not find the latest version / não achei a última versão"
fi
version="${version#v}"

asset="blup_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/v$version"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

say "Downloading Blup $version ($os/$arch)…"
curl -fsSL -o "$tmp/$asset" "$base/$asset" || fail "download failed / falha no download"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "download failed / falha no download"

want="$(grep " $asset\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then
  got="$(sha256sum "$tmp/$asset" | cut -d' ' -f1)"
else
  got="$(shasum -a 256 "$tmp/$asset" | cut -d' ' -f1)"
fi
[ -n "$want" ] && [ "$want" = "$got" ] || fail "checksum mismatch, nothing installed / verificação falhou, nada foi instalado"

tar -xzf "$tmp/$asset" -C "$tmp" blup
mkdir -p "$dir"
mv "$tmp/blup" "$dir/blup"
chmod 755 "$dir/blup"

say "Blup $version installed in $dir/blup"
case ":$PATH:" in
  *":$dir:"*) say "Run: blup" ;;
  *)
    say ""
    say "Add $dir to your PATH / Adicione $dir ao PATH:"
    say "  echo 'export PATH=\"$dir:\$PATH\"' >> ~/.profile && . ~/.profile"
    ;;
esac
