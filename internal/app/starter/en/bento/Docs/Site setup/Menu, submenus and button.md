---
title: Menu, submenus and button
description: Pick the items of the top menu, group them in submenus and highlight a button.
weight: 2
---
## Automatic menu

When no menu is chosen, the top menu shows the site's sections, in the order of each one's **weight**.

## In Blup Studio

On the **Site** screen, in the **Top menu** block:

1. **Add page** picks a folder or note of the site.
2. **Add link** makes an item for another address: a name and the `https://`.
3. Each item's name is up to you. Click it to edit.
4. **Drag by the handle** to change the order.

### Submenus

- **Drag one item on top of another**: it goes into the submenu of the item above. The item above then only opens the submenu, without a link.
- **New submenu** makes an empty entry with a name. Drag items into it.
- The buttons on each item do the same without a mouse: up, down, **Into the submenu above**, **Out of the submenu** and remove.
- There is only one level: a submenu cannot go inside another.

### Top button

In the **Top button** block, write the **Button text** and **Where it leads**. It shows to the right of the menu, highlighted. Leave the text empty for no button.

## In Obsidian

In the **_index** note of the site folder, one item per line:

```yaml
menu:
  - Docs
  - Help | Help center
  - Resources |
  - + What's new | Changelog
  - + Roadmap
  - Pricing | https://acme.com/pricing
button_text: Sign up
button_link: https://app.acme.com/signup
```

| Line | Becomes |
| --- | --- |
| `Docs` | A link to the folder (or note) with that name or title. |
| `Help \| Help center` | The text "Help", leading to the "Help center" folder. |
| `Resources \|` | An entry that only opens the submenu. |
| `+ What's new \| Changelog` | With `+` in front, the item goes into the submenu of the line above. |
| `Pricing \| https://...` | A link to another site. |
| `Home` | The home page. |
| `Blog \| /blog/` | An address on the site itself. |

The item of the current page is highlighted.

## From the command line

`blup customize` > top menu builds the menu by picking folders, tags and links, and asks about the button. Submenus are not rebuilt from the terminal: it tells you and lets you edit them in the Studio or in Obsidian.
