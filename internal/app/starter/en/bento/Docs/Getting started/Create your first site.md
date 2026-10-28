---
title: Create your first site
description: The wizard asks one thing at a time. When in doubt, press Enter.
weight: 2
---
Run `blup new` (or just `blup`, the first time). Each question shows up on its own and comes with a suggested answer.

## The questions

1. **Obsidian vault**: Blup lists the vaults it found.
2. **Folder**: the whole vault or just one folder, like "Site".
3. **Theme**:
   - **Mochi**: personal blog, with posts, tags and authors.
   - **Bento**: product site, this one.
4. **Sections** (Bento only): Docs, Roadmap, Changelog, Help center and Blog. Tick the ones you want; you can add more later.
5. **Site name**, **one sentence about it** and **your name**.
6. **Site language** and **where to save** the site files on your computer.
7. **Summary** to confirm.
8. **Obsidian adjustment**: Markdown links with relative paths, the format websites understand. Blup keeps a copy of the old settings.
9. **Keep the notes on GitHub**: optional, to use them on another computer. See [Sync](../CLI/Sync.md).
10. **Open the preview**.

## What the wizard creates

- In the vault: this tutorial (if the folder is empty), your author note and the **Templates** folder with note templates.
- In a separate folder: **blup.toml** and the files the site generator (Hugo) uses. You don't need to open that folder.

> [!tip] Already had notes in the folder?
> Nothing is deleted or overwritten. Blup only creates the **_index** note if it is missing, and the folders of the sections you chose.
