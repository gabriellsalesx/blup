---
title: Menu and button
description: Choose the items of the top menu and the highlighted button.
weight: 2
---
## Automatic menu

Without the **menu** property, the top menu shows the site's sections, ordered by each one's **weight**.

## Chosen menu

In the **_index** note of the site folder, one item per line:

```yaml
menu:
  - Docs
  - Support | Help center
  - What's new | Changelog
  - Pricing | https://acme.com/pricing
```

| Line | Becomes |
| --- | --- |
| `Docs` | A link to the folder (or note) with that name or title. |
| `Support \| Help center` | The text "Support", leading to the "Help center" folder. |
| `Pricing \| https://...` | A link to another site. |
| `Home` | The home page. |
| `Blog \| /blog/` | An address on the site itself. |

The item of the current page is highlighted.

## The button

```yaml
button_text: Sign up
button_link: https://app.acme.com/signup
```

It shows up at the top right, highlighted. Without both properties, there is no button.
