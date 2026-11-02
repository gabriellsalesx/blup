---
title: How do I link one note to another?
description: Links between notes become links between pages.
weight: 3
---
## In Blup Studio

- Type `[[` and pick the note from the list.
- Or type `@` at the start of a line (or after a space) and pick a note or a date.

The Studio inserts a link in the format the site understands. In the editor, the link shows only as its text, in blue. Click it to open the other note.

## In Obsidian

Use relative Markdown links, like `[Installation](../Getting%20started/Installation.md)`. When you create the site, Blup offers to set Obsidian up to make links like these. If your vault already has `[[wikilinks]]`, run `blup convert` to change them all at once.
