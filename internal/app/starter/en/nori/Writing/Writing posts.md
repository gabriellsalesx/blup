---
title: Writing posts
date: {{date-3}}
description: Creating a post, properties, links between notes, images and sections.
tags: [Writing, Studio]
image: "[[blup-escrever.jpg]]"
---
To create a post, create a note in the site folder and write. That's all. The rest is optional.

In the Studio, click **New note** in the Files panel, give it a name and press Enter. The preview next to it shows the post as it will look on the site, while you write.

## Commands with /

At the start of a line (or after a space), type `/` and choose: heading, list, task list, quote, callout, table, code block, divider, link to another note, external link, image from the computer or today's date. Enter or Tab picks one.

Type `@` to mention another note or a date, like today, tomorrow or next Friday.

The **formatting bar**, above the text, has the same commands as buttons. Right clicking in the editor also gives you the **Format** and **Insert** menus.

## The properties

Date, tags, cover and the rest live in the **Properties** panel, above the editor. Click the panel's name to open or close it. The Studio saves everything at the top of the note itself.

| In the Studio | In the note | What it's for |
| --- | --- | --- |
| **Title** | `title` | The post's title. Empty, the note's name is used. |
| **Description** | `description` | Summary in lists, on Google and when shared. |
| **Date** | `date` | The post's date. |
| **Tags** | `tags` | Topics. Each tag gets its own page. |
| **Authors** | `authors` | Who wrote it. Empty, the site's author is used. |
| **Cover** | `image` | The cover image, like `"[[my-cover.jpg]]"`. |
| **Draft** | `draft: true` | Keeps the note off the site. It shows only in the preview. |
| **Hidden** | `hidden: true` | Publishes it, but keeps it out of lists and Google. |
| **Password** | `password` | Asks for a password to read. See [Password protected posts](Password%20protected%20posts.md). |

In the note's file, the same properties sit in a block at the top, between two `---` lines.

## Links between notes

In the Studio, type `[[` and pick the note by name (Tab picks), or use **Link to another note** in the `/` menu. In the text, the link shows just the name, in blue. Saved in the note, it looks like this:

```md
[Formatting that works](Formatting%20that%20works.md)
```

On the site, it becomes the link to the page. Example: [Formatting that works](Formatting%20that%20works.md).

For a link to another website, use **Link** (Ctrl+K).

> [!tip] Renamed a note?
> In the Studio, rename without worry: it updates the links that point to the note by itself.

Already have old notes with `[[wikilinks]]`? Run `blup convert`: Blup turns the old links into the standard format and shows what will change first.

## Images

Paste the image right into the note, or drag it, or use **Image from the computer**. The Studio saves the file in the **Attachments** folder by itself and the image shows up in place, inside the text.

Blup makes smaller, modern versions (WebP) of each image, and the browser downloads only the size it needs.

![Blup writing on a note card](../Attachments/blup-escrever.jpg)

The text in square brackets is the image's description: it helps people who use screen readers, and Google.

The **Attachments** screen, in the Studio's side menu, shows all the images and files the notes use. The **Unused** filter finds what's left over.

## Organize into sections

Create a folder in the site folder, like **Travel**, and put posts in it: the site gets a Travel section, with its own page and a menu item. To give the section a title and description, create an **_index** note inside it, like the one in this section.

## Spell checker

The Studio underlines misspelled words while you write. Right click the word to see suggestions or add the word to the dictionary. The checker runs on your computer: the text never leaves it.
