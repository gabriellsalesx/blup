---
title: Writing posts
date: {{date-3}}
description: The properties the site understands, links between notes and images.
tags: [Writing]
image: "[[blup-escrever.jpg]]"
---
To create a post, click **New note** in the **Files** panel and give it a name. That's it. Everything else is optional and lives in the note's **Properties**, a form the Studio shows above the text.

## The properties

| Property | What it's for |
| --- | --- |
| **Title** | Post title. Without it, the note name is used. |
| **Description** | Summary in lists, on Google and when someone shares the post. |
| **Date** | Post date. Without it, the file date is used. |
| **Tags** | Topics. Type one and press Enter. Each tag gets a page. |
| **Cover** | The post image, at the top and in lists. |
| **Draft** | Only in the preview, never published. |
| **Hidden** | Published, but out of lists and Google. |
| **Password** | Asks for a password to read it. See [Password protected posts](Password%20protected%20posts.md). |
| **Authors** | Who wrote it. Without it, the site author. It's under **More options**. |

Open this post in the Studio to see its properties filled in.

> [!info] Where they are saved
> The properties are saved at the top of the note, between two lines of `---`. The Studio shows them as a form, so you never need to touch that block.

## Links between notes

Right-click in the text and choose **Link to another note**, then find the note by name. Or type `[[` and pick the note from the list. On the site, it becomes the page link. Example: [Formatting that works](Formatting%20that%20works.md).

For a link to another website, select the text and use **Link** in the formatting bar (or Ctrl+K).

> [!tip] Renaming is safe
> Renaming a note in the **Files** panel updates every link that points to it.

## Images

Paste or drag the image into the note. The Studio saves it in **Attachments** by itself. You can also use **Image from the computer** in the formatting bar. Blup creates smaller, modern versions (WebP) of each image, and the browser downloads only the size it needs.

![Blup writing on a note card](../Attachments/blup-escrever.jpg)

The text between the brackets is the image description: it helps people using screen readers, and Google.

**Attachments**, in the side menu, lists every image and file the notes use. There you can rename them (the notes are updated) and find the ones no note uses.

## Organize into sections

Click **New folder** in the **Files** panel, name it **Travel** and put posts in it: the site gets a Travel section, with its own page and menu item. To give the section a title and description, create an **_index** note inside it, like the one in this section.
