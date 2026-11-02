---
title: How do I install Blup?
description: Download Blup Studio or install the command line, on Linux, macOS or Windows.
weight: 2
---
## Blup Studio (recommended)

1. Open **getblup.io** in your browser.
2. Download the version for your system: Linux, macOS or Windows.
3. Install and open **Blup Studio**. The first time, it asks for your language and helps with the rest.

## Command line

If you prefer the terminal (or write in Obsidian), you can install the `blup` command.

On macOS and Linux, open the **Terminal** and paste:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

On Windows, open **PowerShell** and paste:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Then run `blup`.

> [!info] Do I need to install anything else?
> No. Blup downloads whatever it needs to build the site on its own.
