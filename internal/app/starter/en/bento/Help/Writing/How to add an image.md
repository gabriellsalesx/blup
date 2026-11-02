---
title: How do I add an image?
description: Paste or drag the image into the note.
weight: 2
---
## In Blup Studio

Paste the image (or drag the file) into the note. The Studio saves the image in the attachments folder and puts the link in the text. The image shows up in the editor right away.

For a post's **cover**, pick the image in **Properties**, in the image field.

## By hand

Put the image in the attachments folder and link it in the note. For the cover, use the `image` property:

```yaml
image: "[[my-cover.jpg]]"
```

Blup resizes and converts images on its own. Covers look good at 1600x900.
