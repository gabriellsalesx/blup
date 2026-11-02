---
title: Help center
description: Collections of articles with icons and search, like support centers.
weight: 4
---
The help center is a folder whose **_index** has `blup: help`. The entry page has a search box and the **collections** as cards.

![Blup with a life buoy](../../Attachments/blup-ajuda.jpg)

## Collections and articles

- Each **subfolder** is a collection.
- Each **note** inside it is an article.
- The collection's **_index** gives the title, description and icon.

```yaml
---
title: Getting started
description: The basics to get going.
icon: rocket
weight: 1
---
```

## Icons

**In Blup Studio**: open the collection's **_index** note. The properties show **Collection icon**: click it to open the icon grid and search by name or by everyday words (rocket, book, help...). **Automatic** removes the icon and lets the theme pick.

**In Obsidian**: the **icon** property takes any name from the [Lucide](https://lucide.dev/icons) library, with more than 2 thousand icons: `rocket`, `pen-line`, `globe`, `life-buoy`, `credit-card`, `shield-check`, `users`... Copy the name from the Lucide site. Without icon, the theme picks one.

## Title and menu

In the help center's _index, **title** is the big question on the entry page ("How can we help?") and **linkTitle** is the short name in the menu ("Help center").
