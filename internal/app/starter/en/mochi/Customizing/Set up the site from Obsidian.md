---
title: Set up the site from Obsidian
date: {{date-6}}
description: Name, description, menu, button, social links, logo and favicon live in the _index note.
tags: [Customizing]
image: "[[blup-configurar.jpg]]"
---
The site settings are **properties of the _index note** of the site folder. Change them in Obsidian and the site changes. If you prefer the terminal, `blup customize` asks the same questions and saves to the same properties.

## Name and description

```yaml
site_name: My Blog
site_description: Notes about writing, travel and books.
```

The name shows up in the browser tab, in the footer (© year and name) and on Google. The description shows up on Google and when someone shares the site.

## Top menu

One item per line:

```yaml
menu:
  - Home
  - Writing
  - Reading | Books
  - Shop | https://myshop.com
```

- A name on its own looks for a **folder or note** with that name: `Writing` leads to the Writing section.
- `Text shown | folder or note` changes the item's text: `Reading | Books` shows "Reading" and leads to the Books folder.
- `Name | https://...` becomes a link to another site.
- `Home` is the home page.
- Without the property, the menu is automatic: the folders at the root of the site.

## Button at the top

```yaml
button_text: Subscribe
button_link: https://mynewsletter.com
```

## Social links

```yaml
social:
  - https://instagram.com/yourname
  - https://github.com/yourname
  - mailto:you@example.com
```

The icons show up in the footer. Blup recognizes the network from the address: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp and e-mail.

## Logo and favicon

Put the images in the vault (in the **Attachments** folder, for example) and point to them:

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
```

The favicon is the browser tab icon. As SVG, it stays sharp at any size; as PNG, Blup creates the sizes browsers and iPhones ask for.

> [!tip] Want to see it in practice?
> Open the **_index** note of this site folder: this blog's menu is there.
