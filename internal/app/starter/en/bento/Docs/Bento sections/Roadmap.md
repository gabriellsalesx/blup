---
title: Roadmap
description: A board with three columns, one note per item.
weight: 2
---
The roadmap is a folder whose **_index** has `blup: roadmap`. Each note inside it is a card.

## The column

The **status** property picks the column. English or Portuguese both work:

| Column | Write |
| --- | --- |
| Planned | `planned` or `planejado` (also the default, without a status) |
| In progress | `in progress`, `doing`, `em andamento` |
| Done | `done`, `shipped`, `completed`, `concluído` |

## The card

```yaml
---
title: More themes
description: New looks to choose from when creating the site.
status: planned
category: Themes
weight: 1
---
```

- **title** and **description** show up on the card.
- **category** is the label at the bottom of the card. Without it, the first tag is used.
- **weight** orders the cards inside the column.
- The note's text becomes the item's page, opened by clicking the card.

To move an item to another column, change the **status** and save.
