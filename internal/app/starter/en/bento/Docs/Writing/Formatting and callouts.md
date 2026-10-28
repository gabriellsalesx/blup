---
title: Formatting and callouts
description: Text, lists, tables and Obsidian's callout boxes.
weight: 3
---
## Text

**Bold**, *italic*, ~~strikethrough~~, ==highlight== and `code`. Footnotes too.[^1]

[^1]: They go to the end of the page, with a link back.

## Lists

- Items with a hyphen
  - and sub-items

1. Numbered steps
2. In order

- [x] Done task
- [ ] Pending task

## Tables

| Section | Folder | Property |
| --- | --- | --- |
| Docs | Docs | `blup: docs` |
| Roadmap | Roadmap | `blup: roadmap` |

## Headings

Level 2 (`##`) and level 3 (`###`) headings show up in **On this page**, next to the text, and get their own link.

## Callouts

> [!note] Note
> Write `> [!note] Title` and the text on the following lines, starting with `>`.

> [!info]
> Without a title, the callout shows the type name.

> [!tip] Tip
> Types: note, abstract, info, todo, tip, success, question, warning, failure, danger, bug, example and quote. Obsidian's aliases (hint, important, faq, caution, error...) work too.

> [!success] Success
> Each type has its own color and icon.

> [!question] Question
> Handy for FAQs.

> [!warning] Warning
> For heads-ups.

> [!danger] Danger
> For things that can't be undone.

> [!faq]- Folded callout
> A `-` after the type starts the callout folded; click to open. A `+` starts it open, but foldable.
