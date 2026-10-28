---
title: Authors and profiles
date: {{date-7}}
description: One note per author, with photo, role, social links and bio. Posts with more than one author.
tags: [Customizing]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
This post was "written" by **Blup**, the mascot. Notice the photo and name at the top: they come from Blup's note in the **Writers** folder.

## The writers folder

The **Writers** folder has an **_index** note with the property `blup: authors`. It tells Blup that the notes in there are profiles, not posts: the folder doesn't become a page and doesn't show up in the menu.

Each author is a note named after the person:

```yaml
---
photo: "[[ana.jpg]]"
role: Editor
social:
  - https://instagram.com/ana
  - https://github.com/ana
---
Ana writes about books and travel.
```

- **photo**: the profile photo.
- **role**: the role or job title.
- **social**: profile links, with automatic icons.
- The **text of the note** is the bio.

When you created the site, Blup already created your note in this folder. Fill in the photo and the bio.

## In posts

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

Without the property, the post belongs to the site author. With more than one author, the photos show up side by side and the names together.

## Each author's page

Click the author's name on a post: their page shows photo, role, bio, social links and every post they wrote.
