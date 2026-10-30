#!/usr/bin/env bash
# Builds the tools of Blup Studio's local voice (Kokoro) for this system:
# eSpeak NG (text to phonemes) and LAME (MP3), from their official source
# releases, statically linked so they run on any distribution. The result,
# blup-voice-tools-<os>-<arch>.tar.gz, is published with Blup's releases
# and downloaded by the Studio (internal/kokoro) only when the user picks
# the local voice. The tools run as separate programs: they are not part of
# Blup's binary.
#
# Usage: build/voice/build-tools.sh [work dir]   (Linux or macOS)
set -euo pipefail

ESPEAK_VERSION=1.52.0
ESPEAK_SHA256=bb4338102ff3b49a81423da8a1a158b420124b055b60fa76cfb4b18677130a23
LAME_VERSION=3.100
LAME_SHA256=ddfe36cab873794038ae2c1210557ad34857a4b6bdc515785d1da9e175b1da1e

here=$(cd "$(dirname "$0")" && pwd)
work=${1:-$(mktemp -d)}
mkdir -p "$work"
cd "$work"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
# Fully static programs on Linux (libtool wants -all-static at link time).
static=()
[ "$os" = linux ] && static=(LDFLAGS=-all-static)
jobs=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)

fetch() { # url file sha256
  [ -f "$2" ] || curl -fsSL -o "$2" "$1"
  echo "$3  $2" | sha256sum -c - >/dev/null
}

fetch "https://github.com/espeak-ng/espeak-ng/archive/refs/tags/$ESPEAK_VERSION.tar.gz" "espeak-ng-$ESPEAK_VERSION.tar.gz" "$ESPEAK_SHA256"
fetch "https://downloads.sourceforge.net/project/lame/lame/$LAME_VERSION/lame-$LAME_VERSION.tar.gz" "lame-$LAME_VERSION.tar.gz" "$LAME_SHA256"

rm -rf "espeak-ng-$ESPEAK_VERSION" "lame-$LAME_VERSION" bundle
tar xzf "espeak-ng-$ESPEAK_VERSION.tar.gz"
tar xzf "lame-$LAME_VERSION.tar.gz"

(
  cd "espeak-ng-$ESPEAK_VERSION"
  ./autogen.sh >/dev/null 2>&1
  ./configure --quiet --disable-shared --enable-static --with-async=no --with-mbrola=no \
    --with-sonic=no --with-pcaudiolib=no --with-klatt=no --with-speechplayer=no \
    --with-extdict-ru=no --with-extdict-cmn=no --with-extdict-yue=no
  # One job: its data steps race each other in parallel.
  make "${static[@]}" >/dev/null 2>&1
)
(
  cd "lame-$LAME_VERSION"
  # No termcap/ncurses (only colors in a terminal; it would not link
  # statically): its configure has no switch, so its checks are answered.
  ./configure --quiet --disable-shared --enable-static --disable-gtktest --disable-decoder \
    ac_cv_header_termcap_h=no ac_cv_header_ncurses_termcap_h=no \
    ac_cv_lib_termcap_initscr=no ac_cv_lib_curses_initscr=no ac_cv_lib_ncurses_initscr=no
  make -j"$jobs" "${static[@]}" >/dev/null
)

# Only what Portuguese and English need from eSpeak NG's data.
b=bundle
data=$b/espeak-ng-data
mkdir -p "$data" "$b/licenses"
src=espeak-ng-$ESPEAK_VERSION/espeak-ng-data
cp "$src"/{phontab,phonindex,phondata,intonations,pt_dict,en_dict} "$data/"
cp -r "$src/lang" "$data/"
cp "espeak-ng-$ESPEAK_VERSION/src/espeak-ng" "$b/"
cp "lame-$LAME_VERSION/frontend/lame" "$b/"
strip "$b/espeak-ng" "$b/lame" 2>/dev/null || true
cp "espeak-ng-$ESPEAK_VERSION/COPYING" "$b/licenses/espeak-ng-GPL-3.0.txt"
cp "lame-$LAME_VERSION/COPYING" "$b/licenses/lame-LGPL-2.0.txt"
cat >"$b/SOURCES.txt" <<EOF
eSpeak NG $ESPEAK_VERSION (GPL-3.0-or-later)
  https://github.com/espeak-ng/espeak-ng/archive/refs/tags/$ESPEAK_VERSION.tar.gz
  sha256 $ESPEAK_SHA256
LAME $LAME_VERSION (LGPL-2.0-or-later)
  https://downloads.sourceforge.net/project/lame/lame/$LAME_VERSION/lame-$LAME_VERSION.tar.gz
  sha256 $LAME_SHA256
Built by Blup's studio/build/voice/build-tools.sh (unmodified sources).
EOF
cp "$here/build-tools.sh" "$b/"

out="blup-voice-tools-$os-$arch.tar.gz"
tar czf "$out" -C "$b" .
sha256sum "$out"
