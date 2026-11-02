---
title: Authors and profiles
date: {{date-7}}
description: One profile per author, with photo, role, social links and bio. Posts with more than one author.
tags: [Customizing, Studio]
image: "[[blup-autores.jpg]]"
authors: [Blup]
---
This post was "written" by **Blup**, the mascot. Notice the photo and name at the top: they come from Blup's profile, in the **Writers** folder.

## In the Studio

The **Authors** screen, in the side menu, lists the profiles. For each one:

- **Photo**: a picture of your own or one of the ready made avatars.
- **Role**: the job title, like "Editor".
- **Social networks**: pick the network and finish the profile address.
- **Bio**: a short text about the person.

When you created the site, Blup already created your profile. Fill in the photo and the bio.

To say who wrote a post, use the **Authors** field in the post's **Properties** panel. Start typing and pick the name.

## In the note's file

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

The note's text is the bio. In posts:

```yaml
authors: [Ana]
authors: [Ana, Bruno]
```

## How it shows up

With no author, the post belongs to the site's author. With more than one, the photos show side by side and the names together.

Click the author's name on a post: their page shows the photo, role, bio, social links and every post they wrote. On a blog in several languages, the role and bio are translated too.
