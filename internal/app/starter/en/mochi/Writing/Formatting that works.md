---
title: Formatting that works
date: {{date-4}}
description: Everything you write in Obsidian that shows up on the site, with examples.
tags: [Obsidian]
image: "[[blup-formatacao.jpg]]"
---
This post is a showcase. Open the note in Obsidian and see how each block was written.

## Text

**Bold**, *italic*, ~~strikethrough~~, ==highlight==, `code` and an [external link](https://obsidian.md). A footnote works too.[^1]

[^1]: Footnotes go to the end of the post, with a link back.

## Lists

- A list
- with items
  - and sub-items

1. Numbered list
2. Second step

- [x] Done task
- [ ] Pending task

## Table

| Theme | For whom |
| --- | --- |
| Mochi | Personal blog |
| Bento | Product site |

## Quote

> Writing is thinking twice.

## Callouts

Obsidian callouts become colored boxes, with the same icon:

> [!note] Note
> The type goes between brackets: `> [!note]`. The title is optional.

> [!tip] Tip
> There are also info, todo, success, question, warning, failure, danger, bug, example, quote and abstract.

> [!warning] Warning
> Without a title, the callout shows the type name.

> [!danger]
> This one has no title.

> [!example]- Click to open
> A `-` after the type starts the callout folded. A `+` starts it open, but foldable.

## Code

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, Blup!")
}
```

The colors follow the language written after the three backticks.

## Math

Math inline, like $E = mc^2$, and as a block:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

## Comments

Text between `%%` and `%%` is an Obsidian comment: you see it while editing and it disappears on the site. Use it for reminders and drafts inside the post. %%This bit, for example, only exists in Obsidian.%%

## Horizontal rule

---

Three hyphens on a line of their own draw the line above.
