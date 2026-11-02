#!/usr/bin/env bash
# Points PKGBUILD at a release: ./update.sh 1.2.0
# Reads the tarball's sha256 from the release's studio-checksums.txt (made
# and signed by .github/workflows/release.yml), then writes .SRCINFO again.
set -euo pipefail
cd "$(dirname "$0")"
version="${1:?usage: ./update.sh <version, like 1.2.0>}"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || { echo "not a version: $version" >&2; exit 1; }
name="blup-studio_${version}_linux_amd64.tar.gz"
sums="$(curl -fsSL "https://github.com/gabriellsalesx/blup/releases/download/v${version}/studio-checksums.txt")"
sum="$(awk -v n="$name" '$2 == n { print $1 }' <<<"$sums")"
[[ "$sum" =~ ^[0-9a-f]{64}$ ]] || { echo "no checksum for $name" >&2; exit 1; }
sed -i -e "s/^pkgver=.*/pkgver=${version}/" -e "s/^pkgrel=.*/pkgrel=1/" \
  -e "s/^sha256sums_x86_64=.*/sha256sums_x86_64=('${sum}')/" PKGBUILD
makepkg --printsrcinfo > .SRCINFO
echo "PKGBUILD and .SRCINFO now at ${version} (${sum})"
