---
title: Folder structure
description: How folders and notes become the pages of the site.
weight: 5
---
## The map

```text
Site/                     the folder with the site's notes
├── _index.md             home page and site settings
├── Docs/                 documentation section (blup: docs)
│   ├── _index.md         docs opening page
│   └── Getting started/  a group in the side menu
├── Roadmap/              roadmap board (blup: roadmap)
├── Changelog/            list of versions (blup: changelog)
├── Help/                 help center (blup: help)
├── Blog/                 blog (blup: blog)
├── Feedback/             request board (blup: feedback)
├── Writers/              author profiles (blup: authors)
└── Attachments/          images, audio and other files
```

## The rules

- Each **note** becomes a page.
- Each **folder** becomes a group or a section.
- A folder's **_index** note gives it a title, a description and an opening text.
- In Bento, the **blup** property in the _index of a top-level folder says what kind of section it is: `docs`, `roadmap`, `changelog`, `help`, `blog`, `feedback` or `authors`. You can name the folder anything you like.
- A folder called `en`, `es` or `pt` next to a note holds its **translation**. See [Multilingual sites](../Site%20setup/Multilingual%20sites.md).

## What stays off the site

- Notes with `draft: true` (they only show in the preview).
- Folders that start with a dot, like `.obsidian` and `.blup`.
- The Obsidian templates folder (Templates).
- The authors folder: it only becomes the profile data.
- The guides for AI agents (`AGENTS.md`, `CLAUDE.md` and `GEMINI.md`), which the Studio creates when you open the terminal.
- Attachments that no note uses.

## Folders Blup uses behind the scenes

- **.blup**, inside the notes folder: holds the `site.toml` file, the part of the settings that travels with sync.
- **The site folder** (outside your notes): `blup.toml`, the generator's files, the Studio's Trash and the sync state. You never need to open it.

> [!note] Page addresses
> The address comes from the folder and note names, without accents and with hyphens: the note "Folder structure" in "Getting started" becomes `/docs/getting-started/folder-structure/`.
