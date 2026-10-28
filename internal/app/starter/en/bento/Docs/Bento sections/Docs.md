---
title: Docs
description: Pages organized in groups, with a sidebar, table of contents and navigation.
weight: 1
---
The docs are the section you are reading. They live in a root folder whose **_index** note has `blup: docs`.

## How it's organized

- Each **note** is a page.
- Each **subfolder** is a group in the sidebar, with the title of its _index.
- The **_index** of the docs folder is the opening page.
- The order follows the **weight** property (lower first) and, without it, the note name.

```text
Docs/
├── _index.md           blup: docs
├── Getting started/
│   ├── _index.md       title: Getting started, weight: 1
│   ├── Installation.md weight: 1
│   └── Create your first site.md   weight: 2
└── Writing/
    └── _index.md       weight: 2
```

## What each page gets

- A **sidebar** with every group; the current group stays open.
- **On this page**: the `##` and `###` headings, next to the text.
- A **breadcrumb** (Home > Docs > group) at the top.
- **Previous and Next** at the end, following the sidebar order.
- **Search** across every page of the site.

> [!tip] A short description helps
> The **description** property shows up under the title and on Google. One sentence is enough.
