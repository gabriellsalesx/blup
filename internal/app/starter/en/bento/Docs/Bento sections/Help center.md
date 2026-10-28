---
title: Help center
description: Collections of articles with icons and search, like support centers.
weight: 4
---
The help center is a folder whose **_index** has `blup: help`. Its entry page has a search box and the **collections** as cards.

![Blup with a life buoy](../../Attachments/blup-ajuda.jpg)

## Collections and articles

- Each **subfolder** is a collection.
- Each **note** inside it is an article.
- The collection's **_index** gives it a title, description and icon.

```yaml
---
title: Getting started
description: The basics to begin.
icon: rocket
weight: 1
---
```

## Icons

The **icon** property accepts any name from the [Lucide](https://lucide.dev/icons) library, with more than 2,000 icons: `rocket`, `pen-line`, `globe`, `life-buoy`, `credit-card`, `shield-check`, `users`... Copy the name from the Lucide site. Without an icon, Blup picks one.

## Title and menu

On the help center's _index, **title** is the big question on the entry page ("How can we help?") and **linkTitle** is the short name in the menu ("Help center").
