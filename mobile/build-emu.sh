#!/usr/bin/env bash
# Compila para o emulador (x86_64), instala e abre o Blup Notes.
set -e
cd "$(dirname "$0")"
export ANDROID_HOME="${ANDROID_HOME:-$HOME/Android/Sdk}"
# Com o celular também ligado no cabo, o adb precisa saber qual aparelho usar.
export ANDROID_SERIAL="${ANDROID_SERIAL:-emulator-5554}"
mkdir -p bin
wails3 task android:build ARCH=amd64 >bin/build.log 2>&1 || { tail -20 bin/build.log; exit 1; }
wails3 task android:assemble:apk >>bin/build.log 2>&1 || { tail -20 bin/build.log; exit 1; }
"$ANDROID_HOME/platform-tools/adb" install -r bin/blup-notes.apk | tail -1
"$ANDROID_HOME/platform-tools/adb" shell am force-stop io.getblup.notes
"$ANDROID_HOME/platform-tools/adb" shell am start -n io.getblup.notes/com.wails.app.MainActivity >/dev/null
