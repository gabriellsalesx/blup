#!/usr/bin/env bash
# Pins the tools packages built by .github/workflows/voice-tools.yml in
# internal/kokoro/install.go: writes each one's SHA-256 and size in its
# line of the platforms table. A system without a package keeps its empty
# checksum, and the local voice stays off there.
#
# Usage:
#   gh run download <run id> -R gabriellsalesx/blup -D /tmp/voice-tools
#   studio/build/voice/pin-tools.sh /tmp/voice-tools
# It then prints the command that publishes the release with those files.
set -euo pipefail

dir=${1:?usage: pin-tools.sh <folder with the downloaded artifacts>}
root=$(cd "$(dirname "$0")/../../.." && pwd)
go_file=$root/internal/kokoro/install.go
tag=$(sed -nE 's|^const toolsRelease = ".*/download/([^/]+)/"$|\1|p' "$go_file")

sha256() {
  if command -v sha256sum >/dev/null; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

files=()
while IFS= read -r f; do
  name=$(basename "$f")
  sum=$(sha256 "$f")
  size=$(wc -c <"$f" | tr -d ' ')
  if ! grep -q "\"$name\", \"[0-9a-f]*\", [0-9]*}" "$go_file"; then
    echo "$name: no line for it in install.go" >&2
    exit 1
  fi
  sed -E -i.bak "s|(\"$name\", )\"[0-9a-f]*\", [0-9]+\}|\1\"$sum\", $size}|" "$go_file"
  rm -f "$go_file.bak"
  echo "$name  $sum  $size"
  files+=("$f")
done < <(find "$dir" -type f -name 'blup-voice-tools-*.tar.gz' | sort)

if [ ${#files[@]} -eq 0 ]; then
  echo "no blup-voice-tools-*.tar.gz in $dir" >&2
  exit 1
fi
echo
echo "To publish them:"
printf 'gh release create %s -R gabriellsalesx/blup --prerelease --latest=false --title "Voice tools %s" --notes "eSpeak NG and LAME for the local voice of Blup Studio, built from their official sources by studio/build/voice/build-tools.sh."' "$tag" "${tag#voice-tools-}"
printf ' %q' "${files[@]}"
echo
