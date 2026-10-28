---
title: Writing posts in Obsidian
date: {{date-3}}
description: The properties the site understands, links between notes, images and templates.
tags: [Obsidian]
image: "[[blup-escrever.jpg]]"
---
To create a post, create a note in the site folder. That's it. Everything else is optional and lives in the **properties**, the block at the top of the note (in Obsidian, type `---` on the first line or use the "Add property" command).

## The properties

| Property | What it's for |
| --- | --- |
| `title` | Post title. Without it, the note name is used. |
| `date` | Post date. Without it, `created` or the file date is used. |
| `description` | Summary on cards, on Google and on social networks. |
| `tags` | Topics. Each tag gets a page. |
| `image` | Post cover, like `"[[my-cover.jpg]]"`. |
| `authors` | Who wrote it, like `[Ana, Bruno]`. Without it, the site author. |
| `draft` | `true` keeps the note off the site (it shows only in the preview). |
| `hidden` | `true` publishes it, but keeps it out of lists and Google. |
| `password` | Protects the post with a password. See [Password protected posts](Password%20protected%20posts.md). |
| `updated` | Date of the last update, when you want to show it. |

Open this post in Obsidian to see its properties at the top.

## Links between notes

Drag a note into another one, or type `[[` and pick the note. With the adjustment Blup made in Obsidian, the link is saved like this:

```md
[Formatting that works](Formatting%20that%20works.md)
```

On the site, it becomes the page link. Example: [Formatting that works](Formatting%20that%20works.md).

> [!tip] Already have notes with [[wikilinks]]?
> Run `blup convert`: Blup turns the old `[[links]]` into the standard format in every note of the site folder, and shows what will change first.

## Images

Drag the image into the note, as always. Blup creates smaller, modern versions (WebP) of each image, and the browser downloads only the size it needs.

![Blup writing on a note card](../Attachments/blup-escrever.jpg)

The text between the brackets is the image description: it helps people using screen readers, and Google.

## Templates

Blup created the **Templates** folder in your vault with a post template and set up Obsidian's Templates plugin. To start a post with the right properties, use the **Templates: Insert template** command. The templates folder never goes to the site.

## Organize into sections

Create a subfolder in the site folder, like **Travel**, and put posts in it: the site gets a Travel section, with its own page and menu item. To give the section a title and description, create an **_index** note inside it, like the one in this section.
