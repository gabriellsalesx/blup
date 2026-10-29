---
title: Formatting that works
date: {{date-4}}
description: Everything you write in the Studio that shows up on the site, with examples.
tags: [Writing]
image: "[[blup-formatacao.jpg]]"
---
This post is a showcase. Open the note in the Studio and see how each block was written.

Almost everything here comes from the **formatting bar** at the top of the editor: select the text and click the button. The rest is in the menu that opens when you right-click the text (**Format**, **Paragraph** and **Insert**).

## Text

**Bold**, *italic*, ~~strikethrough~~, ==highlight==, `code` and an [external link](https://github.com/gabriellsalesx/blup). A footnote works too.[^1]

[^1]: Footnotes go to the end of the post, with a link back.

Bold, italic, strikethrough and highlight have their own buttons; code is in the bar too. For an external link, select the text and click **Link** (or Ctrl+K). For a footnote, type `[^1]` where it goes and, on a line of its own, `[^1]:` followed by the note.

## Lists

- A list
- with items
  - and sub-items

1. Numbered list
2. Second step

- [x] Done task
- [ ] Pending task

The bar has **List**, **Numbered list** and **Task list**.

## Table

| Theme | For whom |
| --- | --- |
| Nori | Personal blog, a single column |
| Mochi | Personal blog |
| Bento | Product site |

Click **Table** in the bar. Right-click a cell to add or delete rows and columns.

## Quote

> Writing is thinking twice.

Use **Quote** in the bar.

## Callouts

Callouts are colored boxes, each with its own icon. Click **Callout** in the bar and pick the type:

> [!note] Note
> The type goes between brackets: `> [!note]`. The title is optional.

> [!tip] Tip
> There are also info, todo, success, question, warning, failure, danger, bug, example, quote and abstract.

> [!warning] Warning
> Without a title, the callout shows the type name.

> [!danger]
> This one has no title.

> [!example]- Click to open
> To make it start folded, tick **Starts folded** in the callout menu. In the text, that is a `-` after the type; a `+` starts it open, but foldable.

## Code

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, Blup!")
}
```

Right-click and choose **Insert**, then **Code block**. The colors follow the language written after the three backticks.

## Math

Math inline, like $E = mc^2$, and as a block:

$$
x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}
$$

Type it between `$` signs for inline math, or between `$$` lines for a block.

## Comments

Text between `%%` and `%%` is a comment: you see it while editing and it disappears on the site. Use it for reminders and drafts inside the post. To add one, right-click and choose **Format**, then **Comment (stays off the site)**. %%This bit, for example, only exists in the notes.%%

## Divider

---

Click **Divider** in the bar to draw the line above.
