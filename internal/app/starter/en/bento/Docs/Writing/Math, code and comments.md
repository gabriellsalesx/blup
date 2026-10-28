---
title: Math, code and comments
description: Math, code blocks with colors and notes only you can see.
weight: 4
---
## Math

Between `$` inline, like $a^2 + b^2 = c^2$, or between `$$` as a block:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

Math becomes MathML: the browser draws it, with nothing to download.

## Code

Three backticks, the language name and the code:

```js
function hello(name) {
  return `Hello, ${name}!`;
}
```

The colors follow the language: `go`, `js`, `py`, `sh`, `yaml`, `html`, `css` and many more.

## Comments

Text between `%%` and `%%` is an Obsidian comment: you see it while editing and it doesn't go to the site. %%Like this bit, which only exists in the note.%%

Use it for reminders, drafts and instructions for whoever edits, like in the **_index** note of the site folder.
