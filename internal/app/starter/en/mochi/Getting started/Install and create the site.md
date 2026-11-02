---
title: Install and create the site
date: {{date-1}}
description: Download Blup Studio and create the site by answering one question at a time. Or use the command line.
tags: [Getting started, Studio]
image: "[[blup-dicas.jpg]]"
---
## Install Blup Studio

**Blup Studio** downloads from **getblup.io**, for Linux, macOS and Windows. Install it like any other app and open it.

The first time, the Studio says hello and asks a few quick questions: the language, whether you want to connect your GitHub account now, and a general password for password protected posts. You can skip whatever you like and come back to it later, in **Settings** (the gear, at the bottom of the side menu).

## Create a site

On the home screen, click **Create a new site**. The Studio asks one question at a time, and you can change everything later.

1. **What kind of site**: a personal blog or a product site.
2. **Which theme**: Mochi (this one, with cards and covers) or Nori, for a blog; Bento, for a product site.
3. **What's the site called**: the name, one sentence about it, your name and the site's language.
4. **Where do the notes live**: in a new folder (the simplest way) or in a folder you pick.
5. **All set?** Check everything and create the site.

If the folder is empty, Blup puts in the theme's example site: this tutorial.

> [!tip] A folder that already has notes?
> Leave **Adjust the folder so links work on the site** checked. Links between notes then work on the site.

## The site window

- **Files**, on the left: the notes and folders. Use **New note** and **New folder** to create them, and the search to find a note.
- **The editor**, in the middle: where you write. Above it is the note's **Properties** panel.
- **Preview**, on the right: the real site, updated while you write.
- **The side menu**: Write, Attachments, Authors, Site, Sections, Languages, Comments, Cookies and statistics, Publish and Trash, plus the terminal.

There is no save button: the Studio saves on its own while you write.

Right click a note to create another one there, make a copy, move it, copy the link, rename, export or delete it. **Rename** updates the links that point to the note by itself, and **Delete** sends the note to the **Trash**, where you can restore it.

## Through the command line

Prefer the terminal? Install the `blup` command line.

On macOS or Linux, open the Terminal and paste:

```sh
curl -fsSL https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.sh | sh
```

On Windows, open PowerShell and paste:

```powershell
irm https://raw.githubusercontent.com/gabriellsalesx/blup/main/install.ps1 | iex
```

Then run `blup`. The first time it asks for the language and opens the new site wizard: which Obsidian vault, which folder in it, the theme (Mochi or Bento), the site's name and where to save it. Each question comes with a suggested answer; when in doubt, press Enter.

| Command | What it does |
| --- | --- |
| `blup new` | Creates a new site |
| `blup preview` | Opens the preview in the browser |
| `blup publish` | Puts the site online |
| `blup customize` | Menu, social links, logo, comments, cookies and domain |
| `blup sync` | Keeps the notes in a private GitHub repository |
| `blup convert` | Turns `[[wikilinks]]` into standard links |
| `blup update` | Updates Blup |

The same commands also work in Portuguese: `novo`, `ver`, `publicar`, `personalizar`, `sincronizar`, `converter` and `atualizar`.

> [!info] Both together
> A site created in the Studio opens in the command line, and the other way around too. The notes are the same; only where you click changes.
