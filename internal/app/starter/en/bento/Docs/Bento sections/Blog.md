---
title: Blog
description: Posts with cover, tags, authors and related posts.
weight: 5
---
The blog is a folder whose **_index** has `blup: blog`. Each note is a post, with the same look as the Mochi theme.

```yaml
---
title: How this site was built
date: 2026-10-25
description: Inside the template that comes with Blup.
tags: [Behind the scenes]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
```

- The newest post becomes the **featured** one at the top; to choose another, use `featured: true`.
- **image** is the cover. Without it, the first image in the text is used.
- **tags** become filters at the top of the list and pages of their own.
- **authors** shows photo and name; each author gets a page. See [Authors](../Configuration/Authors.md).
- Each post has a table of contents, reading progress bar, share buttons and related posts by tag.
