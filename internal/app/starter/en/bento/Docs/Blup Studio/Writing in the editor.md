---
title: Writing in the editor
description: Commands with /, links with [[, mentions with @, pasted images and spell check.
weight: 2
---
The Studio editor writes Markdown, but shows the text already styled: bold looks bold, headings look big, callouts show their icon. The marks (like `**` or `#`) come back only while the cursor is on them, so you can edit.

![Blup writing](../../Attachments/blup-escrever.jpg)

## The formatting bar

Select some text and use the bar: bold, italic, strikethrough, highlight, code, headings, lists, tasks, quote, callout, link, image, table, divider and comment. The alignment button centers, aligns right or justifies.

## Commands with /

At the start of a line (or after a space), type `/` and pick with the arrow keys and Enter (or Tab):

- Heading 1, Heading 2 and Heading 3
- List, Numbered list and Task list
- Quote and Callout
- Table, Code block and Divider
- Link to another note (opens the note search) and External link
- Image from the computer
- Today's date

In a note vault, `/` also has **New drawing** and **New canvas**: the Studio creates the file and puts the link in the note.

## Links with [[

Type `[[` and start writing a note's name. Pick it from the list and the Studio saves a relative Markdown link, the format the site understands.

In the editor, the link shows only as its text, blue and underlined. One click opens the note. For web links, hold `Ctrl` and click: a plain click keeps editing.

The link button (or `Ctrl+K`) inserts an external link and selects the address so you can paste it.

## Mentions with @

At the start of a line or after a space, type `@`:

- **Notes**: pick a note and the Studio adds a link to it, with the `@` in the text.
- **Dates**: today, tomorrow, yesterday and the coming days of the week by name. It turns into the written date.

In a vault, the mentioned note shows who cited it in the **Mentioned in** panel.

## Images

Paste (`Ctrl+V`) or drag an image into the editor. The Studio saves the file in **Attachments** and puts the image in the text. It shows up right there, inside the editor.

Images from the internet wait for a click on "Show image from…", because loading the image tells its server that the note was opened.

## Tables

Create one with `/` or the bar. Right-click the table to insert or delete rows and columns, or to **Edit as text (Markdown)**.

## Headings that fold

Hover to the left of a heading: the little arrow hides everything up to the next heading of the same level. Shortcuts: `Ctrl+Shift+[` folds, `Ctrl+Shift+]` shows, `Ctrl+Alt+[` folds all and `Ctrl+Alt+]` shows all.

## Spell checker

Misspelled words get underlined. Right-click to see suggestions or to add the word to the dictionary. The spell checker runs on your computer: your text is not sent to any service.

In **Settings > Editor** you turn the spell checker on or off, choose the dictionary language (Portuguese, English or Spanish) and see the words you added.

## Paste without formatting

The right-click menu has **Paste as plain text**, handy when the text comes from a website with styles.

## Editor shortcuts

| Action | Keys |
| --- | --- |
| Save | `Ctrl+S` |
| Undo and redo | `Ctrl+Z` and `Ctrl+Shift+Z` |
| Bold and italic | `Ctrl+B` and `Ctrl+I` |
| Link | `Ctrl+K` |
| Accept the suggestion | Tab |

> [!tip] Comments that stay off the site
> Text between `%%` and `%%` stays only in your notes. Use it for reminders and drafts. See [Math, code and comments](../Writing/Math,%20code%20and%20comments.md).
