---
title: Links and images
description: Links between notes, links to other sites, images and attachments.
weight: 2
---
## Links between notes

Type `[[` and pick the note, or drag the note into the text. With the adjustment Blup made in Obsidian, the link is saved in the standard format, with a relative path:

```md
[Formatting and callouts](Formatting%20and%20callouts.md)
```

On the site it becomes the page link: [Formatting and callouts](Formatting%20and%20callouts.md). Links to a heading of a note (`Note.md#heading`) work too.

> [!tip] Old notes with [[wikilinks]]
> Run `blup convert`. Blup shows how many links it will change and, if you confirm, turns every `[[link]]` in the site folder into the standard format.

## Links to other sites

Write them as usual: `[Obsidian](https://obsidian.md)`. They open in place, safely (`noopener`).

## Images

Drag the image into the note. Blup:

- creates smaller WebP versions of each image;
- gives the browser only the size it needs;
- uses the text between the brackets as the description (good for accessibility and Google).

![Blup at the terminal](../../Attachments/blup-terminal.jpg)

## YouTube videos

Paste the video link as an image: `![How to publish](https://www.youtube.com/watch?v=...)`. On the site it becomes the player. With the cookie banner on, the video waits for consent; see [Cookies and statistics](../Configuration/Cookies%20and%20statistics.md).

## Other attachments

PDFs, audio, video and ZIP files from the vault can be linked too: `[PDF manual](../../Attachments/manual.pdf)`. They are published along with the site.
