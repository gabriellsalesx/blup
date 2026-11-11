#!/usr/bin/env bash
# Makes the .flatpak of the GitHub release from the Studio already built
# (studio/bin/blup-studio), on the GNOME runtime. Not for Flathub: Flathub
# builds from source with its own manifest, and does not take manifests made
# or assisted by AI (see docs).
#
#   build/linux/flatpak/bundle.sh <version> <out.flatpak>
#
# Run `go run ./scripts/flatpak-tools -out studio/bin` from the repository
# root first. Run this script from studio/. Needs flatpak and the runtime
# (flatpak --user install flathub org.gnome.Platform//51).
set -euo pipefail

version="${1:?version}"
out="${2:?output .flatpak}"
app=io.getblup.Studio
runtime="${RUNTIME:-51}"
# A local test can supply temporary metadata without adding an unreleased
# version to the source repository. Releases use the checked-in file.
metainfo="${BLUP_FLATPAK_METAINFO:-build/linux/$app.metainfo.xml}"
if ! grep -Fq "<release version=\"$version\"" "$metainfo"; then
  echo "Add <release version=\"$version\"> to $metainfo first." >&2
  exit 1
fi
for tool in blup-studio hugo pagefind; do
  if [[ ! -x "bin/$tool" ]]; then
    echo "Missing bin/$tool: build the Studio and prepare flatpak-tools first." >&2
    exit 1
  fi
done
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Nothing is compiled in the sandbox: the runtime stands for the SDK too.
flatpak build-init "$work/build" "$app" org.gnome.Platform org.gnome.Platform "$runtime"
files="$work/build/files"
install -Dm755 bin/blup-studio "$files/bin/blup-studio"
install -Dm755 bin/hugo "$files/bin/hugo"
install -Dm755 bin/pagefind "$files/bin/pagefind"
# Inside the sandbox the desktop file, icon and GTK window carry the app
# id. The Studio reads it from FLATPAK_ID.
sed -e "s/^Icon=.*/Icon=$app/" build/linux/blup-studio.desktop >"$work/app.desktop"
echo "StartupWMClass=$app" >>"$work/app.desktop"
install -Dm644 "$work/app.desktop" "$files/share/applications/$app.desktop"
for size in 128 256 512; do
  install -Dm644 "build/linux/icons/${size}x${size}/blup-studio.png" "$files/share/icons/hicolor/${size}x${size}/apps/$app.png"
done
sed -e "s#<launchable type=\"desktop-id\">blup-studio.desktop#<launchable type=\"desktop-id\">$app.desktop#" \
  "$metainfo" >"$work/metainfo.xml"
install -Dm644 "$work/metainfo.xml" "$files/share/metainfo/$app.metainfo.xml"
install -Dm644 ../THIRD_PARTY_NOTICES.md "$files/share/licenses/$app/THIRD_PARTY_NOTICES.md"
install -Dm644 ../LICENSE "$files/share/licenses/$app/LICENSE"

# Export the AppStream catalogue as well as the original MetaInfo. Software
# centres and the repository linter read share/app-info/xmls/<app-id>.xml.gz.
compose=(--no-net --origin=flatpak --prefix=/ --components="$app"
  --data-dir="$files/share/app-info/xmls"
  --icons-dir="$files/share/app-info/icons/flatpak")
if appstreamcli compose --help | grep -q -- '--image-format'; then
  compose+=(--image-format=png)
fi
appstreamcli compose "${compose[@]}" "$files"
mv "$files/share/app-info/xmls/flatpak.xml.gz" "$files/share/app-info/xmls/$app.xml.gz"

# Permissions, each for a reason:
#   wayland, fallback-x11, ipc, dri: the window (WebKit draws with the GPU).
#   network: publishing, Blup Cloud and optional remote AI services.
#   pulseaudio: the posts' narration plays in the window.
#   xdg-documents, ~/Blup: where Blup makes sites and note vaults. Other
#     folders (notes kept elsewhere) are picked through the file
#     chooser portal.
#   Notifications: calendar reminders. StatusNotifier: the tray icon.
#   Single instance uses the app id's namespace, allowed by Flatpak.
flatpak build-finish "$work/build" \
  --command=blup-studio \
  --socket=wayland --socket=fallback-x11 --share=ipc --device=dri \
  --share=network \
  --socket=pulseaudio \
  --filesystem=xdg-documents --filesystem=~/Blup:create \
  --talk-name=org.freedesktop.Notifications \
  --talk-name=org.kde.StatusNotifierWatcher --own-name=org.kde.*

flatpak build-export "$work/repo" "$work/build" stable
flatpak build-bundle --runtime-repo=https://dl.flathub.org/repo/flathub.flatpakrepo \
  "$work/repo" "$out" "$app" stable
echo "made $out"
