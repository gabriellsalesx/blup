---
title: Install and create the site
date: {{date-1}}
description: One command to install, and a wizard that asks one thing at a time.
tags: [Getting started, CLI]
image: "[[blup-terminal.jpg]]"
---
Blup is a terminal program, but you don't need to memorize anything: it shows a menu and asks one question at a time.

## Install

On macOS or Linux, open the Terminal and paste:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

On Windows, open PowerShell and paste:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Then run `blup`. The first time, it asks for the language and opens the new site wizard.

## The wizard

Every question comes with a suggested answer; when in doubt, press Enter.

1. **Which Obsidian vault** becomes a site.
2. **Which folder** of it: the whole vault or just one folder, like "Blog".
3. **Theme**: Mochi (personal blog, this one) or Bento (product site, with docs, roadmap, changelog, help center and blog).
4. **Site name**, **one sentence about it** and **your name**.
5. **Site language** and **where to save** the site files on your computer.
6. **Obsidian adjustments**: Blup asks to use Markdown links with relative paths, the format websites understand.
7. **Keep the notes on GitHub**, if you want to use them on another computer.
8. **Open the preview**.

> [!info] Where things live
> Your notes stay in your vault, as always. In a separate folder, Blup keeps the **blup.toml** file (where to publish, general password) and the files Hugo uses to build the site. You don't need to open that folder.

## The commands

Running just `blup` opens the menu with everything. If you prefer, each action has a command:

| Command | What it does |
| --- | --- |
| `blup new` | Creates a new site |
| `blup preview` | Opens the preview in the browser |
| `blup publish` | Puts the site online |
| `blup customize` | Menu, social links, logo, favicon and author |
| `blup sync` | Keeps the notes in a private GitHub repository |
| `blup convert` | Turns `[[wikilinks]]` into standard links |
| `blup build` | Builds the site into the public folder, with no questions |
| `blup language` | Changes Blup's language |
| `blup update` | Updates Blup |

The Portuguese commands work too: `novo`, `ver`, `publicar`, `personalizar`, `sincronizar`, `converter`, `gerar`, `idioma` and `atualizar`.

## Updates

Blup checks once a day for a new version and **always asks** before updating. If something goes wrong, `blup update --rollback` goes back to the previous version.
