---
title: Notes and properties
description: Create a note and it becomes a page. Properties control the rest.
weight: 1
---
To create a page, create a note in the right folder. **Properties** (the block at the top of the note) are optional and control how it shows up. In Obsidian, type `---` on the first line or use the **Add property** command.

## On every page

| Property | What it's for |
| --- | --- |
| `title` | Page title. Without it, the note name is used. |
| `linkTitle` | Short title for menus and lists. |
| `description` | Summary in lists, on Google and on social networks. |
| `weight` | Order in the sidebar and lists: lower comes first. |
| `draft` | `true` keeps the note off the site (it shows only in the preview). |
| `hidden` | `true` publishes it, but keeps it out of lists and asks Google not to show it. |
| `password` | Protects the page with a password. See [Passwords and hidden pages](../Configuration/Passwords%20and%20hidden%20pages.md). |

## In blog posts and the changelog

| Property | What it's for |
| --- | --- |
| `date` | Date. Without it, `created` is used. |
| `updated` | Date of the last update. |
| `tags` | Topics; each tag gets a page. |
| `image` | Cover, like `"[[my-cover.jpg]]"`. |
| `authors` | Who wrote it, like `[Ana, Bruno]`. |

## In the roadmap and help center

| Property | What it's for |
| --- | --- |
| `status` | Roadmap column: planned, in progress or done. |
| `category` | Label on the roadmap card. |
| `icon` | Icon of a help center collection (on its _index). |

> [!tip] Ready-made templates
> The vault's **Templates** folder has templates for a docs page, a roadmap item and a changelog version. Use Obsidian's **Templates: Insert template** command to start a note with the right properties.
