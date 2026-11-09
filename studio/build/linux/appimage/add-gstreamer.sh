#!/usr/bin/env bash
# Sound in the AppImage. Wails bundles WebKitGTK with the GStreamer
# libraries but none of GStreamer's plugins, and the bundled GStreamer looks
# for them where Ubuntu keeps them (/usr/lib/x86_64-linux-gnu/...). On Arch,
# Fedora and others nothing is there: the first <audio> (a post's narration
# in the Studio) found no sound output and WebKit aborted, closing the
# window. This adds the plugins audio playback needs, built for the same
# GStreamer as the bundled libraries (run it on the machine that built the
# AppImage), with what they link to, and makes AppRun point GStreamer at
# them, with a registry of its own so the system's plugins never mix in.
#
#   add-gstreamer.sh <in.AppImage> <out.AppImage>
#
# Needs: gstreamer1.0-plugins-base, gstreamer1.0-plugins-good and
# gstreamer1.0-pulseaudio installed, and appimagetool (APPIMAGETOOL).
set -euo pipefail

in="$(readlink -f "$1")"
out="$(readlink -m "$2")"
tool="${APPIMAGETOOL:-appimagetool}"
triplet="$(gcc -dumpmachine 2>/dev/null || echo x86_64-linux-gnu)"
gst="/usr/lib/$triplet/gstreamer-1.0"
scanner="/usr/lib/$triplet/gstreamer1.0/gstreamer-1.0/gst-plugin-scanner"

# Playing a file: reading and typing it, demuxing, decoding the usual
# formats (mp3, aac in mp4, ogg/opus/vorbis, wav, flac, webm), converting,
# and the outputs (PulseAudio, which PipeWire also serves, and ALSA).
plugins=(coreelements typefindfunctions playback app autodetect pulseaudio alsa
	audioconvert audioresample volume audioparsers audiorate
	mpg123 isomp4 matroska ogg opus vorbis wavparse flac
	videoconvertscale videorate)

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cd "$work"
"$in" --appimage-extract >/dev/null
app="$work/squashfs-root"
lib="$app/usr/lib"
dest="$lib/gstreamer-1.0"
mkdir -p "$dest"

for p in "${plugins[@]}"; do
	f="$gst/libgst$p.so"
	if [ ! -e "$f" ]; then echo "missing GStreamer plugin: $f" >&2; exit 1; fi
	cp -L "$f" "$dest/"
done
cp -L "$scanner" "$dest/gst-plugin-scanner"

# What the plugins and the scanner link to and the AppImage lacks. The
# system's own base libraries stay out (the AppImage never carries them).
skip='^(linux-vdso|ld-linux|libc|libm|libdl|libpthread|librt|libresolv|libutil)\.so'
for f in "$dest"/*; do
	ldd "$f" | awk '/=> \//{print $3}'
done | sort -u | while read -r dep; do
	name="$(basename "$dep")"
	if [[ "$name" =~ $skip ]] || [ -e "$lib/$name" ]; then continue; fi
	cp -L "$dep" "$lib/$name"
done

# AppRun: GStreamer looks only at the bundled plugins.
grep -q GST_PLUGIN_SYSTEM_PATH_1_0 "$app/AppRun" || sed -i 's|^cd "\$usr/lib"$|# GStreamer: only the bundled plugins (build/linux/appimage/add-gstreamer.sh).\
export GST_PLUGIN_SYSTEM_PATH_1_0="$usr/lib/gstreamer-1.0"\
export GST_PLUGIN_PATH_1_0="$usr/lib/gstreamer-1.0"\
export GST_PLUGIN_SCANNER_1_0="$usr/lib/gstreamer-1.0/gst-plugin-scanner"\
gst_cache="${XDG_CACHE_HOME:-$HOME/.cache}/blup-studio"\
mkdir -p "$gst_cache" 2>/dev/null || true\
export GST_REGISTRY_1_0="$gst_cache/gstreamer-registry.bin"\
cd "$usr/lib"|' "$app/AppRun"
grep -q GST_PLUGIN_SYSTEM_PATH_1_0 "$app/AppRun" || { echo "AppRun not patched" >&2; exit 1; }

ARCH="${ARCH:-$(uname -m)}" "$tool" --no-appstream "$app" "$out" >/dev/null
echo "AppImage with $(ls "$dest" | wc -l) GStreamer files: $out"
