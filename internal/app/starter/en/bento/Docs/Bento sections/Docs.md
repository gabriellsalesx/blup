---
title: Docs
description: Pages organized in groups, with a side menu, table of contents and navigation.
weight: 1
---
The docs are the section you are reading. They live in a root folder whose **_index** note has `blup: docs`.

## How it is organized

- Each **note** is a page.
- Each **subfolder** is a group in the side menu, with the title of its _index.
- The **_index** of the docs folder is the opening page.
- The order follows the **weight** property (lowest first) and, without it, the note's name.

```text
Docs/
├── _index.md           blup: docs
├── Getting started/
│   ├── _index.md       title: Getting started, weight: 1
│   ├── Installation.md   weight: 1
│   └── Create your first site.md   weight: 2
└── Writing/
    └── _index.md       weight: 2
```

## What each page gets

- A **side menu** with all the groups; the current group stays open.
- **On this page**: the `##` and `###` headings, next to the text.
- A **breadcrumb** (Home > Docs > group) at the top.
- **Previous and Next** at the end, following the menu order.
- **Search** across every page of the site.

## Create a page

- **In Blup Studio**: click a subfolder of Docs, then **New note** (or `Ctrl+N`). Fill in **Title**, **Description** and **Order** in the properties.
- **By hand**: create the note in the subfolder and write `title`, `description` and `weight` at the top.

For a new group, create a subfolder with an **_index** note (title, description and `weight`).

> [!tip] A short description helps
> The **description** property shows under the title and on Google. One sentence is enough.
