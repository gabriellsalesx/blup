# Native Linux screenshots

This helper opens the real Studio with disposable example notes. Native
previews use a separate instance ID and temporary XDG directories, without copying
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

## Flatpak GUI tests

After preparing the runtime and a direct bundle, open the installed application
with disposable examples:

```sh
wails3 task store:preview:flatpak VIEW=site FLATPAK_BUNDLE=bin/blup-studio_1.1.0-test.1_x86_64.flatpak
```

This uses the real application ID, refuses to activate an already running Blup
Flatpak, and installs into its own temporary installation. It grants access only
to its temporary examples in addition to the bundle's permissions. A shell sets
the isolated XDG directories inside the sandbox after Flatpak's own setup, before
starting Studio. The test expires after eight minutes and removes its installation
and examples. `store:stop` stops only the recorded, verified instance.

On Hyprland, from another terminal:

```sh
wails3 task store:inspect
wails3 task store:capture OUTPUT=bin/flatpak-site.png
wails3 task store:exercise:flatpak
wails3 task store:probe:flatpak
wails3 task store:stop
```

`store:exercise:flatpak` requires `wtype` and a supported virtual keyboard
protocol. It focuses the verified example window, types a test paragraph, saves
it, creates an example note and runs a command in Studio's terminal. Before each
keyboard operation it checks that the same window still has focus. Do not type
into the preview while this short test runs. It checks the resulting files only
under the temporary example directory and restores the previous focus.

The probe runs a compiled test helper inside the same installation. It verifies
the update channel, packaged Hugo/Pagefind and rejection of executable downloads.
It lists available portal versions and attempts to open and cancel a FileChooser
request. This does not select a real folder or verify persistent Documents portal
permissions. A failed portal probe remains a failed check and needs investigation;
available interface versions alone do not demonstrate desktop integration.

For a runner without desktop environment variables, pass the same local `SESSION`
file to each task. Window identity checks use both process metadata and the
installation recorded in `/.flatpak-info`, since Flatpak's wrapper process and
the process owning the GTK window have different host PIDs. Temporary profiles,
probe executables, captures and logs stay outside commits.
