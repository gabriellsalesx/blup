---
title: Blog
description: Posts with covers, tags, authors and related posts.
weight: 5
---
The blog is a folder whose **_index** has `blup: blog`. Each note is a post, with the same look as the Mochi theme.

```yaml
---
title: How this site was built
date: 2026-10-25
description: A look inside the model that comes with Blup.
tags: [Behind the scenes]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
```

- The newest post becomes the **featured** one at the top; to pick another, use `featured: true`.
- **image** is the cover. Without it, the first image in the text is used.
- **tags** become filters at the top of the list and get their own pages.
- **authors** shows a photo and a name; each author gets a page. See [Authors](../Site%20setup/Authors.md).
- Each post has a table of contents, a reading bar, share buttons and related posts by tag.
- With a **narration**, the post gets a player at the top. See [Post narration](../AI%20features/Post%20narration.md).

> [!tip] In Blup Studio
> Create the post with **New note** in the blog folder and fill in **Date**, **Tags**, **Cover** and **Authors** in the properties. You can pick the cover from Attachments or paste it from your computer.
