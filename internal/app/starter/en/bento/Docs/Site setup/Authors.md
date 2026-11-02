---
title: Authors
description: One profile per person, with photo, role, social links and bio.
weight: 4
---
Each author gets a page with photo, role, bio, social links and their posts. On posts, the photo and the name show up next to the title.

![The authors](../../Attachments/blup-autores.jpg)

## In Blup Studio

Click **Authors** in the side bar. The screen lists the people, with how many posts each one has, and marks the **Site author**.

- **New author** opens the form: **Name** (as it shows on posts), photo, **Role**, **Bio** and **Social links**.
- For the photo, pick one of the ready avatars or **Upload own photo**.
- **Edit** changes everything. Changing the name updates that person's posts.
- **Delete author**: the posts keep the name, without a photo. You can restore it from the Trash.

On posts, pick the authors in the **Authors** field of the [Note properties](../Blup%20Studio/Note%20properties.md).

## In the note's file

The profiles live in a folder whose **_index** has `blup: authors`, here called **Writers**. This folder does not become a page: its notes are only data.

One note named after the person, like `Ana Lima.md`:

```yaml
---
photo: "[[ana.jpg]]"
role: Product designer
social:
  - https://github.com/ana
---
Ana designs the product and writes about how it is made.
```

- **photo**: the photo. Without it, the initials show up (or the mascot, for the site author).
- **role**: the role.
- **social**: the profiles, with icons added automatically.
- The note's **text** is the bio.

On posts:

```yaml
authors: [Ana Lima, Bruno Costa]
```

Without **authors**, the post belongs to the site author. With more than one, the photos show side by side and the names together ("Ana Lima and Bruno Costa").

## From the command line

`blup customize` > **Name, description and author** edits the site author's name, role, bio and photo.

> [!tip] Profiles in other languages
> On a site in several languages, the role and the bio are translated too, in a language folder inside Writers. See [Multilingual sites](Multilingual%20sites.md).
