---
title: Links and images
description: Links between notes, links to other sites, images, videos and attachments.
weight: 2
---
## Links between notes

In Blup Studio, type `[[`, pick the note and you're done. Or use `/` and **Link to another note**. See [Writing in the editor](../Blup%20Studio/Writing%20in%20the%20editor.md).

The link is saved like this, with a relative path:

```md
[Formatting and callouts](Formatting%20and%20callouts.md)
```

On the site it becomes a link to the page: [Formatting and callouts](Formatting%20and%20callouts.md). Links to a heading in the note (`Note.md#heading`) work too.

> [!tip] Renamed or moved? The links follow
> In the Studio, **Rename** and **Move to…** fix the links in other notes on their own.

> [!tip] Older notes with wikilinks
> Run `blup convert`. Blup shows how many links it will change and, if you confirm, turns every wikilink in the site folder into the standard format. It keeps a copy of the affected notes first.

## Links to other sites

Write them as usual: `[Blup](https://getblup.io)`. In the Studio, the link button (or `Ctrl+K`) puts in the format and selects the address so you can paste it.

## Images

In the Studio, paste (`Ctrl+V`) or drag the image into the editor. It goes to **Attachments** and shows up right there.

When publishing, Blup:

- makes smaller WebP versions of each image;
- sends the browser only the size it needs;
- uses the text between the brackets as the description (good for accessibility and Google).

![Blup in the terminal](../../Attachments/blup-terminal.jpg)

## YouTube videos

Paste the video link as an image: `![How to publish](https://www.youtube.com/watch?v=...)`. On the site it becomes the player. With the cookie banner on, the video waits for consent; see [Cookies and statistics](../Site%20setup/Cookies%20and%20statistics.md).

## Other attachments

PDFs, audio, videos and ZIP files can be linked too, like `[PDF manual](../../Attachments/manual.pdf)`. They are published along with the site. See the [Attachments](../Blup%20Studio/Attachments.md) screen.
