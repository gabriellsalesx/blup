---
title: Math, code and comments
description: Math, code blocks with colors and notes only you can see.
weight: 4
---
## Math

Between `$` in the middle of the text, like $a^2 + b^2 = c^2$, or between `$$` as a block:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

Formulas become MathML: the browser draws them, with nothing to download.

## Code

Three backticks, the language name and the code:

```js
function hello(name) {
  return `Hello, ${name}!`;
}
```

The colors follow the language: `go`, `js`, `py`, `sh`, `yaml`, `html`, `css` and many others.

## Comments

Text between `%%` and `%%` is an Obsidian comment: it shows while you edit and never goes to the site. %%Like this bit, which only exists in the note.%%

Use them for reminders, drafts and instructions for whoever edits, like in the **_index** note of the site folder.

In the Studio, the formatting bar has the **Comment (stays off the site)** button. Narration skips comments too.

> [!tip] Code blocks in the Studio
> In the Studio editor, a code block gets a background, the language name and a **Copy** button. The three backticks only show with the cursor inside the block.
