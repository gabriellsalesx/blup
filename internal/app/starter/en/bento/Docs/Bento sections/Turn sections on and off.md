---
title: Turn sections on and off
description: Every section is optional. Add, remove and restore them whenever you like.
weight: 9
---
## In Blup Studio

Click **Sections** in the side bar (Bento only). The screen shows each part of the product site, with one line about it:

| Section | What it is |
| --- | --- |
| Docs | Pages with a sidebar and table of contents |
| Roadmap | A board with planned, in progress and done |
| Changelog | What changed in each version |
| Help center | Collections of articles with search |
| Blog | News with covers and tags |
| Feedback | A board where visitors ask for features, report bugs and vote. Blup Studio only |

- **Add** creates the section's folder with example notes that teach you how to use it.
- The ones already on the site show as **On the site**. Removing a section sends its folder to the **Trash**, and the notice has **Undo**.
- A removed section can be restored from the Trash.

The preview updates by itself after each change.

## From the command line

Run `blup customize` and pick **Site sections (docs, roadmap, changelog)**. Blup shows the ones that are missing and creates each one's folder, with example notes. To remove one, delete the folder. Feedback does not show in this list.

## By hand

A folder at the root of the site with an **_index** note that has the **blup** property:

| Section | Property |
| --- | --- |
| Docs | `blup: docs` |
| Roadmap | `blup: roadmap` |
| Changelog | `blup: changelog` |
| Help center | `blup: help` |
| Blog | `blup: blog` |
| Feedback | `blup: feedback` |
| Author profiles | `blup: authors` |

## Rename and reorder

- The folder name is up to you: "Docs", "Guide", "Manual"... what matters is the **blup** property.
- The _index's **title** is the name in the menu and at the top of the section.
- The _index's **weight** sorts the sections in the automatic menu and in the home page cards.
