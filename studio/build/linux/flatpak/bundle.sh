#!/usr/bin/env bash
# Makes the .flatpak of the GitHub release from the Studio already built
# (studio/bin/blup-studio), on the GNOME runtime. Not for Flathub: Flathub
# builds from source with its own manifest, and does not take manifests made
# or assisted by AI (see docs).
#
#   build/linux/flatpak/bundle.sh <version> <out.flatpak>
#
# Run from studio/. Needs flatpak and org.gnome.Platform//$RUNTIME installed
# (flatpak --user install flathub org.gnome.Platform//51).
set -euo pipefail

version="${1:?version}"
out="${2:?output .flatpak}"
app=io.getblup.Studio
runtime="${RUNTIME:-51}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

# Nothing is compiled in the sandbox: the runtime stands for the SDK too.
flatpak build-init "$work/build" "$app" org.gnome.Platform org.gnome.Platform "$runtime"
files="$work/build/files"
install -Dm755 bin/blup-studio "$files/bin/blup-studio"
# Inside the sandbox the desktop file, icon and GTK window carry the app
# id. The Studio reads it from FLATPAK_ID.
sed -e "s/^Icon=.*/Icon=$app/" build/linux/blup-studio.desktop >"$work/app.desktop"
echo "StartupWMClass=$app" >>"$work/app.desktop"
install -Dm644 "$work/app.desktop" "$files/share/applications/$app.desktop"
for size in 128 256 512; do
  install -Dm644 "build/linux/icons/${size}x${size}/blup-studio.png" "$files/share/icons/hicolor/${size}x${size}/apps/$app.png"
done
sed -e "s/@VERSION@/$version/" -e "s/@DATE@/$(date -u +%F)/" \
  -e "s#<launchable type=\"desktop-id\">blup-studio.desktop#<launchable type=\"desktop-id\">$app.desktop#" \
  "build/linux/$app.metainfo.xml" >"$work/metainfo.xml"
install -Dm644 "$work/metainfo.xml" "$files/share/metainfo/$app.metainfo.xml"
install -Dm644 ../THIRD_PARTY_NOTICES.md "$files/share/licenses/$app/THIRD_PARTY_NOTICES.md"
install -Dm644 ../LICENSE "$files/share/licenses/$app/LICENSE"

# Permissions, each for a reason:
#   wayland, fallback-x11, ipc, dri: the window (WebKit draws with the GPU).
#   network: publishing, Blup Cloud, the AI and downloading Hugo.
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
