---
title: Notes and properties
description: Create a note and it becomes a page. Properties handle the rest.
weight: 1
---
To create a page, create a note in the right folder. **Properties** (the block at the top of the note, between `---` lines) are optional and control how it shows up.

- **In Blup Studio**: click **New note** (or `Ctrl+N`) and fill in the **Properties** panel, above the text. See [Note properties](../Blup%20Studio/Note%20properties.md).
- **By hand**: type `---` on the first line, write the properties and close the block with another `---`.

## On every page

| Property | What it's for |
| --- | --- |
| `title` | Page title. Without it, the note's name is used. |
| `linkTitle` | Short title for menus and cards. |
| `description` | Summary in lists, on Google and on social media. |
| `weight` | Order in the side menu and in lists: lower comes first. |
| `draft` | `true` keeps the note off the site (it only shows in the preview). |
| `hidden` | `true` publishes it, but takes it out of lists and asks Google not to show it. |
| `password` | Protects the page with a password. See [Passwords and hidden pages](../Site%20setup/Passwords%20and%20hidden%20pages.md). |

## In blog posts and the changelog

| Property | What it's for |
| --- | --- |
| `date` | Date. Without it, `created` is used. |
| `updated` | Date of the last update. |
| `tags` | Topics; each tag gets its own page. |
| `image` | Cover, like `"[[my-cover.jpg]]"`. |
| `authors` | Who wrote it, like `[Ana, Bruno]`. |
| `featured` | `true` picks the featured post. |

## In the roadmap and the help center

| Property | What it's for |
| --- | --- |
| `status` | Roadmap column: planned, in progress or done. |
| `category` | The card's label on the roadmap. |
| `icon` | Icon of a help center collection (in its _index). |

## Filled in by Blup on its own

| Property | Who fills it in |
| --- | --- |
| `audio`, `audio_hash`, `audio_seconds` | Narration. See [Post narration](../AI%20features/Post%20narration.md). |
| `translationKey`, `translation_hash`, `ai_translated` | Translation. See [Multilingual sites](../Site%20setup/Multilingual%20sites.md). |
| `created`, `updated` | The Studio, in vault notes. |

> [!tip] Ready-made templates
> Sites created from the command line get a **Templates** folder, with templates for a docs page, a roadmap item and a changelog version. Copy one to start a note with the right properties.
