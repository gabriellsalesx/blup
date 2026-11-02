---
title: Installation
description: Download Blup Studio at getblup.io. The command line is optional.
weight: 1
---
There are two ways to use Blup, and they get along well:

- **Blup Studio** (recommended): a desktop app with an editor, live preview, settings screens and one-button publishing.
- **The `blup` command line** (optional): for people who prefer the terminal.

## Requirements

- Linux, macOS or Windows.
- That's it. Blup downloads what it needs to build the site on its own.

## Install Blup Studio

1. Open **getblup.io** in your browser.
2. Download the version for your system (Linux, macOS or Windows).
3. Install it like any other app and open **Blup Studio**.

The first time, the Studio says hello, asks for your language and offers to sign in to your Blup Cloud account. The account is **optional**: everything works on this computer without it. Then go on to [Create your first site](Create%20your%20first%20site.md).

> [!info] Studio updates
> In **Settings > About**, the Studio looks for a new version once a day (or when you click **Check for updates**). When there is one, a blue dot shows at the bottom of the settings, along with the **Download** button.

## Install the command line (optional)

You only need this if you will use `blup` in the terminal. The Studio does not depend on it.

### macOS and Linux

Open the Terminal and paste:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

### Windows

Open PowerShell and paste:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

### Check it

Run `blup`. The first time, it asks for your language (English, Portuguese or Spanish) and opens the new site wizard.

> [!info] Command line updates
> `blup` checks once a day for a new version and always asks before updating. See [Update](../Command%20line/Update.md).
