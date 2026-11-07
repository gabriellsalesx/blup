---
title: Studio and command line
description: Two ways to use the same notes. The Studio covers the whole path, from text to a live site.
weight: 4
---
Blup notes are **plain Markdown files** in a folder on your computer. Blup Studio opens that folder, and so does the `blup` command line. You can switch whenever you like.

## The recommended way: Blup Studio

The Studio puts everything in one app: writing, seeing the site, setting it up and publishing. If you are just starting, it is the shortest path.

## Also fine: the command line

If you prefer the terminal, use `blup` to preview (`blup preview`), publish (`blup publish`) and change the settings (`blup customize`). See [Commands](../Command%20line/Commands.md).

> [!tip] Use both
> You can edit the notes in any text editor and open the Studio only to preview, set things up and publish. The Studio notices when a note changes outside it and refreshes the screen.

## What only the Studio has

| Feature | Where to read |
| --- | --- |
| Editor with a live preview of the site, side by side | [Live preview](../Blup%20Studio/Live%20preview.md) |
| Site, Sections, Authors, Languages, Comments, Cookies, Podcast and Publish screens | [Site settings](../Site%20setup/Site%20settings.md) |
| Submenus in the top menu, by dragging | [Menu, submenus and button](../Site%20setup/Menu,%20submenus%20and%20button.md) |
| Translating notes into other languages | [Translation](../AI%20features/Translation.md) |
| Narrating posts as audio | [Post narration](../AI%20features/Post%20narration.md) |
| Feedback board in Bento | [Feedback](../Bento%20sections/Feedback.md) |
| Note vaults with daily note, map of notes, canvas, drawings and databases | [What is a vault](../Note%20vaults/What%20is%20a%20vault.md) |
| Sync and publishing with Blup Cloud | [Sync your notes](../Blup%20Cloud/Sync%20your%20notes.md) |
| Terminal with AI agents, and undoing what the AI changed | [Terminal and AI agents](../Blup%20Studio/Terminal%20and%20AI%20agents.md) |
| Trash, version history and a daily copy of your notes | [Trash and version history](../Blup%20Studio/Trash%20and%20history.md) |

## What works in both

- Writing in Markdown, with properties, callouts, tables, math and `%%` comments.
- Building the site, previewing it and publishing to Cloudflare Workers or GitHub Pages.
- Syncing your notes with a private GitHub repository.
- Passwords, hidden pages, drafts, cookies, comments and reactions.

## Links between notes

Sites understand Markdown links with relative paths, like `[Installation](Installation.md)`. The Studio always saves links this way, even when you type `[[`. Older notes with `[[wikilinks]]` can be converted with `blup convert`. See [Links and images](../Writing/Links%20and%20images.md).
