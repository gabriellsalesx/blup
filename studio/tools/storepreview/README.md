# Native Linux screenshots

This helper opens the real Studio with disposable example notes. It gives the
preview a separate instance ID and temporary XDG directories, without copying
accounts, credentials or existing notes. Closing the launcher removes its
temporary files. The example text and helper were prepared with AI assistance;
the screenshots are unedited captures of the running Linux application.

From `studio/`, after compiling the frontend, run in a Linux desktop session:

```sh
wails3 task store:preview VIEW=notes THEME=light
```

Use `THEME=dark` for dark mode or `VIEW=site` for the site editor and live preview.
`VIEW=library` opens the example library. Close each preview before starting the
next one; this command stops only the dedicated preview executable:

```sh
wails3 task store:stop
```

On Hyprland with its Lua dispatch API and `grim` installed, capture from a second
terminal:

```sh
wails3 task store:capture OUTPUT=bin/store-preview.png
```

The capture task identifies the preview by its exact executable and process ID,
focuses and sizes its window, and restores the previous focus afterward. It
refuses to capture if another window has focus. On other desktops, use the
system's window screenshot tool. Review every image before publishing it.

An agent runner without desktop environment variables can use `SESSION` to read
a local JSON file containing only desktop connection references. The helper
accepts only `DISPLAY`, `WAYLAND_DISPLAY`, `XDG_RUNTIME_DIR`, `XDG_SESSION_TYPE`,
`DBUS_SESSION_BUS_ADDRESS` and `HYPRLAND_INSTANCE_SIGNATURE`; it ignores other
keys. Keep that file outside the repository. Never commit session references,
temporary profiles, logs or the preview executable. The local state file and
default capture destination are under the ignored `studio/bin/` directory.

Reviewed captures live in `.github/assets/screenshots/`. Capture dimensions in
AppStream describe the PNG pixels, which can differ from the logical window
size on scaled displays. These native captures do not validate a Flatpak build
or its desktop portals.
