---
title: Formatting that works
date: {{date-4}}
description: Everything you write in the Studio that shows up on the site, with examples.
tags: [Writing]
image: "[[blup-formatacao.jpg]]"
---
This post is a showcase. Open the note in the Studio and see how each block was written. In the Studio, almost everything has a button on the **formatting bar** and an item in the `/` menu, and the editor already shows the result: the marks only appear when the cursor passes over them.

## Text

**Bold**, *italic*, ~~strikethrough~~, ==highlight==, `code` and [a link out](https://github.com/gabriellsalesx/blup). A footnote works too.[^1]

[^1]: Footnotes go to the end of the post, with a link back.

## Lists

- A list
- with items
  - and sub items

1. Numbered list
2. Second step

- [x] Done task
- [ ] Pending task

## Table

| Theme | For whom |
| --- | --- |
| Mochi | A blog with cards and covers |
| Nori | A single column blog, with the text up front |
| Bento | A product site |

## Quote

> Writing is thinking twice.

## Callouts

Callouts become colored boxes, with an icon. In the Studio, use **Callout**:

> [!note] Note
> The type goes in square brackets: `> [!note]`. The title is optional.

> [!tip] Tip
> There are also info, todo, success, question, warning, failure, danger, bug, example, quote and abstract.

> [!warning] Heads up
> With no title, the callout shows the type's name.

> [!danger]
> This one has no title.

> [!example]- Click to open
> A `-` after the type keeps the callout closed. A `+` keeps it open, but foldable.

## Code

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, Blup!")
}
```

The colors follow the language written after the three backticks.

## Formulas

Formulas in the middle of the text, like $E = mc^2$, and as a block:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

## Comments

Text between `%%` and `%%` is a comment: it shows while you edit and disappears on the site. It's for reminders and drafts inside the post. %%This part, for example, only exists in the notes.%%

## Divider

---

Three hyphens on a line by themselves draw the line above.
