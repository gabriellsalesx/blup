#!/usr/bin/env bash
# Checks the direct bundle's exported application with the official linter.
# It does not validate a source-build manifest or request linter exceptions.
set -euo pipefail
bundle="$(realpath "${1:?bundle .flatpak}")"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
flatpak --user run --filesystem="$work" --command=ostree org.flatpak.Builder \
  --repo="$work/repo" init --mode=archive-z2
flatpak build-import-bundle "$work/repo" "$bundle"
flatpak --user run --filesystem="$work:ro" --command=flatpak-builder-lint \
  org.flatpak.Builder repo "$work/repo"
