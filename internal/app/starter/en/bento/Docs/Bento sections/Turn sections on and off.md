---
title: Turn sections on and off
description: Every section is optional. Add them with Blup, remove them by deleting the folder.
weight: 6
---
## Add

Run `blup customize` and choose **Site sections**. Blup shows the missing ones and creates a folder for each, with example notes.

You can also create one by hand: a folder at the root of the site with an **_index** note that has the **blup** property:

| Section | Property |
| --- | --- |
| Docs | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Help center | `blup: help` |
| Blog | `blup: blog` |
| Author profiles | `blup: authors` |

## Remove

Delete the folder in Obsidian. Done: the section leaves the site and the menu.

## Rename and reorder

- The folder name is up to you: "Docs", "Guide", "Manual"... what matters is the **blup** property.
- The **title** of the _index is the name in the menu and at the top of the section.
- The **weight** of the _index orders the sections in the automatic menu and in the home page cards.
