---
title: Installation
description: One command in the terminal and Blup is ready.
weight: 1
---
## Requirements

- [Obsidian](https://obsidian.md) with a vault.
- macOS, Linux or Windows.
- That's it. Blup downloads what it needs to build the site by itself.

## macOS and Linux

Open the Terminal and paste:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

## Windows

Open PowerShell and paste:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

## Check

Run `blup`. The first time, it asks for the language (English or Portuguese) and opens the new site wizard.

> [!info] Updates
> Blup checks once a day for a new version and always asks before updating. See [Update](../CLI/Update.md).
