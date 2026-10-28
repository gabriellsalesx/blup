---
title: Folder structure
description: How the folders and notes of the vault become the pages of the site.
weight: 3
---
## The map

```text
Site/                     the site folder
├── _index.md             home page and site settings
├── Docs/                 docs section (blup: docs)
│   ├── _index.md         docs opening page
│   └── Getting started/  group in the sidebar
├── Roadmap/              roadmap board (blup: roadmap)
├── Changelog/            list of versions (blup: changelog)
├── Help/                 help center (blup: help)
├── Blog/                 blog (blup: blog)
├── Writers/              author profiles (blup: authors)
└── Attachments/          images
```

## The rules

- Each **note** becomes a page.
- Each **folder** becomes a group or section.
- A folder's **_index** note gives it a title, description and opening text.
- In Bento, the **blup** property on the _index of a root folder says what kind of section it is: `docs`, `roadmap`, `changelog`, `help`, `blog` or `authors`. The folder name is up to you.

## What stays off the site

- Notes with `draft: true`.
- Folders starting with a dot, like `.obsidian`.
- The templates folder (Templates).
- The writers folder: it only becomes profile data.

> [!note] Page addresses
> The address comes from the folder and note names, lowercase and with hyphens: the note "Folder structure" in "Getting started" becomes `/docs/getting-started/folder-structure/`.
