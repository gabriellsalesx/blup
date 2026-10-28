---
title: Authors
description: One note per person, with photo, role, social links and bio.
weight: 3
---
Profiles live in a folder whose **_index** has `blup: authors`, here called **Writers**. That folder doesn't become a page: its notes are just profile data.

## The profile

A note named after the person, like `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Product designer
social:
  - https://github.com/ana
  - https://x.com/ana
---
Ana designs the product and writes about how it's made.
```

- **photo**: the photo. Without it, initials show up (or the mascot, for the site author).
- **role**: the role.
- **social**: profiles, with automatic icons.
- The note's **text** is the bio.

The new site wizard already creates your note. `blup customize` > **Name, description and author** edits name, role, bio and photo.

## In posts

```yaml
authors: [Ana Lima]
authors: [Ana Lima, Bruno Costa]
```

Without **authors**, the post belongs to the site author. With more than one, the photos show up side by side and the names together ("Ana Lima and Bruno Costa").

Each author gets a page with photo, role, bio, social links and their posts.
