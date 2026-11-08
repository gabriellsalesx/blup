#!/usr/bin/env bash
# Builds the tools of Blup Studio's local voice (Kokoro) for this system:
# eSpeak NG (text to phonemes) and LAME (MP3), from their official source
# releases, statically linked where the system allows it (fully on Linux
# and Windows; on macOS only Apple's system libraries stay dynamic). The
# result, blup-voice-tools-<os>-<arch>.tar.gz, is published with Blup's
# releases and downloaded by the Studio (internal/kokoro) only when the user
# picks the local voice. The tools run as separate programs: they are not
# part of Blup's binary.
#
# Usage: build/voice/build-tools.sh [work dir]   (a short path: eSpeak NG
#   keeps the path of its data in a fixed size buffer while it builds)
#   Linux, macOS, or Windows in an MSYS2 UCRT64 shell. The CI workflow
#   .github/workflows/voice-tools.yml runs it on every system.
#   On Windows, VCRT_DIR may name a folder with Microsoft's Visual C++
#   runtime (msvcp140.dll, vcruntime140.dll...): those files go into the
#   package, because ONNX Runtime's Windows build needs them and not every
#   Windows has them installed.
set -euo pipefail

ESPEAK_VERSION=1.52.0
ESPEAK_SHA256=bb4338102ff3b49a81423da8a1a158b420124b055b60fa76cfb4b18677130a23
LAME_VERSION=3.100
LAME_SHA256=ddfe36cab873794038ae2c1210557ad34857a4b6bdc515785d1da9e175b1da1e

here=$(cd "$(dirname "$0")" && pwd)
work=${1:-$(mktemp -d)}
mkdir -p "$work"
cd "$work"

case $(uname -s) in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  MINGW* | MSYS* | CYGWIN*) os=windows ;;
  *) echo "unsupported system: $(uname -s)" >&2; exit 1 ;;
esac
case $(uname -m) in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac
x= # executable suffix
[ "$os" = windows ] && x=.exe
# Fully static programs on Linux and Windows (libtool wants -all-static at
# link time). macOS has no static system library: there the programs only
# link Apple's, checked below.
static=() # (expanded with ${static[@]+...}: macOS bash 3.2 rejects an empty array under set -u)
[ "$os" != darwin ] && static=(LDFLAGS=-all-static)
# The oldest macOS the tools run on.
[ "$os" = darwin ] && export MACOSX_DEPLOYMENT_TARGET=${MACOSX_DEPLOYMENT_TARGET:-11.0}
jobs=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 4)

sha256() { # file: prints its SHA-256
  if command -v sha256sum >/dev/null; then
    sha256sum "$1" | cut -d' ' -f1
  else
    shasum -a 256 "$1" | cut -d' ' -f1
  fi
}

fetch() { # url file sha256
  [ -f "$2" ] || curl -fsSL --retry 3 -o "$2" "$1"
  got=$(sha256 "$2")
  if [ "$got" != "$3" ]; then
    echo "$2: SHA-256 $got, expected $3" >&2
    exit 1
  fi
}

fetch "https://github.com/espeak-ng/espeak-ng/archive/refs/tags/$ESPEAK_VERSION.tar.gz" "espeak-ng-$ESPEAK_VERSION.tar.gz" "$ESPEAK_SHA256"
fetch "https://downloads.sourceforge.net/project/lame/lame/$LAME_VERSION/lame-$LAME_VERSION.tar.gz" "lame-$LAME_VERSION.tar.gz" "$LAME_SHA256"

rm -rf "espeak-ng-$ESPEAK_VERSION" "lame-$LAME_VERSION" bundle
tar xzf "espeak-ng-$ESPEAK_VERSION.tar.gz"
tar xzf "lame-$LAME_VERSION.tar.gz"

espeak_src=espeak-ng-$ESPEAK_VERSION
if [ "$os" = windows ]; then
  # On Windows eSpeak NG builds with CMake (its autotools files expect a
  # Unix system); with MinGW its CMake file links the program statically.
  # SONIC_LIB and SONIC_INC are set only so CMake does not download
  # libsonic, which is turned off anyway.
  cmake -S "$espeak_src" -B espeak-build -G Ninja -Wno-dev \
    -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF -DBUILD_TESTING=OFF \
    -DUSE_ASYNC=OFF -DUSE_MBROLA=OFF -DUSE_LIBSONIC=OFF -DUSE_LIBPCAUDIO=OFF \
    -DUSE_KLATT=OFF -DUSE_SPEECHPLAYER=OFF -DSONIC_LIB=unused -DSONIC_INC=unused \
    -DEXTRA_ru=OFF -DEXTRA_cmn=OFF -DEXTRA_yue=OFF >/dev/null
  # One job: its data steps race each other in parallel.
  cmake --build espeak-build -j 1 >/dev/null
  espeak_bin=espeak-build/src/espeak-ng$x
  espeak_data=espeak-build/espeak-ng-data
else
  (
    cd "$espeak_src"
    ./autogen.sh >/dev/null 2>&1
    ./configure --quiet --disable-shared --enable-static --with-async=no --with-mbrola=no \
      --with-sonic=no --with-pcaudiolib=no --with-klatt=no --with-speechplayer=no \
      --with-extdict-ru=no --with-extdict-cmn=no --with-extdict-yue=no
    # One job: its data steps race each other in parallel.
    make ${static[@]+"${static[@]}"} >/dev/null 2>&1
  )
  espeak_bin=$espeak_src/src/espeak-ng
  espeak_data=$espeak_src/espeak-ng-data
fi
(
  cd "lame-$LAME_VERSION"
  # No termcap/ncurses (only colors in a terminal; it would not link
  # statically) and no iconv (only for ID3 tags, which Blup does not
  # write): its configure has no switch, so its checks are answered.
  ./configure --quiet --disable-shared --enable-static --disable-gtktest --disable-decoder \
    ac_cv_header_termcap_h=no ac_cv_header_ncurses_termcap_h=no \
    ac_cv_lib_termcap_initscr=no ac_cv_lib_curses_initscr=no ac_cv_lib_ncurses_initscr=no \
    am_cv_func_iconv=no
  make -j"$jobs" ${static[@]+"${static[@]}"} >/dev/null
)

# Only what Portuguese, English and Spanish need from eSpeak NG's data.
b=bundle
data=$b/espeak-ng-data
mkdir -p "$data" "$b/licenses"
cp "$espeak_data"/{phontab,phonindex,phondata,intonations,pt_dict,en_dict,es_dict} "$data/"
cp -r "$espeak_src/espeak-ng-data/lang" "$data/"
cp "$espeak_bin" "$b/espeak-ng$x"
cp "lame-$LAME_VERSION/frontend/lame$x" "$b/lame$x"
strip "$b/espeak-ng$x" "$b/lame$x" 2>/dev/null || true
cp "$espeak_src/COPYING" "$b/licenses/espeak-ng-GPL-3.0.txt"
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

case $os in
  darwin)
    # Only Apple's libraries, nothing from Homebrew; then a fresh ad hoc
    # signature (strip changes the file, and Apple Silicon runs only
    # signed programs).
    for f in "$b/espeak-ng" "$b/lame"; do
      if otool -L "$f" | tail -n +2 | grep -vE '^[[:space:]]+/(usr/lib|System/Library)/'; then
        echo "$f links a library outside the system" >&2
        exit 1
      fi
      codesign --force --sign - "$f"
    done
    ;;
  windows)
    # Only Windows' own DLLs (no MinGW runtime to ship along).
    for f in "$b/espeak-ng.exe" "$b/lame.exe"; do
      if objdump -p "$f" | grep -i 'DLL Name' | grep -viE 'DLL Name: (kernel32|user32|advapi32|shell32|ws2_32|msvcrt|ucrtbase|api-ms-win-[a-z0-9-]+)\.dll'; then
        echo "$f needs a DLL Windows does not have" >&2
        exit 1
      fi
    done
    if [ -n "${VCRT_DIR:-}" ]; then
      for dll in msvcp140.dll msvcp140_1.dll vcruntime140.dll vcruntime140_1.dll; do
        cp "$VCRT_DIR/$dll" "$b/$dll"
      done
      cat >>"$b/SOURCES.txt" <<EOF
msvcp140.dll, msvcp140_1.dll, vcruntime140.dll, vcruntime140_1.dll
  Microsoft Visual C++ Redistributable, copied unchanged from Visual Studio's
  redistributable folder; ONNX Runtime's Windows build needs them.
EOF
    fi
    ;;
  linux)
    for f in "$b/espeak-ng" "$b/lame"; do
      if ldd "$f" >/dev/null 2>&1 && ! ldd "$f" 2>&1 | grep -qE 'not a dynamic executable|statically linked'; then
        echo "$f is not static" >&2
        exit 1
      fi
    done
    ;;
esac
cp "$here/build-tools.sh" "$b/"

# The programs run here, as the Studio runs them.
ipa=$(printf 'olá, mundo' | "$b/espeak-ng$x" --path="$b" -q -b 1 --ipa --tie=^ -v pt-br)
echo "eSpeak NG: $ipa"
[ -n "$ipa" ]
head -c 48000 /dev/zero >silence.raw
"$b/lame$x" --quiet -r -s 24 --bitwidth 16 --signed --little-endian -m m -b 64 -q 2 - - <silence.raw >silence.mp3
[ -s silence.mp3 ]
echo "LAME: $(wc -c <silence.mp3) bytes of MP3"

out="blup-voice-tools-$os-$arch.tar.gz"
# COPYFILE_DISABLE keeps macOS's tar from adding ._ files.
COPYFILE_DISABLE=1 tar czf "$out" -C "$b" .
echo "$(sha256 "$out")  $(wc -c <"$out" | tr -d ' ')  $out"
