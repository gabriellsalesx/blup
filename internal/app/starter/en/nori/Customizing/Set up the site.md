---
title: Set up the site
date: {{date-6}}
description: Name, description, logo, menu with submenus, button and social links, on the Studio's Site screen or in the _index note.
tags: [Customizing, Studio, Obsidian]
image: "[[blup-configurar.jpg]]"
---
The site's settings are on the **Site** screen, in the Studio's side menu. Change it there and it changes on the site: the preview shows it right away.

All of this is saved in the properties of the **_index** note in the site folder. So, in Obsidian, you just edit that note. At the end of each part there is the way to write it in Obsidian.

## Name and description

The **Site name** shows at the top, in the browser tab, in the footer (© year and name) and on Google. The **Description** is one sentence about the site: it shows on Google and when someone shares the site.

Under **Author**, pick who signs the posts that don't name an author.

```yaml
site_name: My Blog
site_description: Notes on writing, travel and books.
```

## Logo and favicon

Under **Logo and favicon**, click **Choose**. The image goes into the **Attachments** folder by itself.

- The **Logo** shows at the top of the site. With no logo, the name shows.
- **Show the name next to the logo**: turned off, the top of the site shows only the logo. Good when the logo already has the name in it.
- The **Favicon** is the icon in the browser tab and for the app. As SVG, it stays sharp at any size; as PNG, Blup makes the sizes browsers and the iPhone ask for. Square, ideally 512 pixels.

```yaml
logo: "[[logo.svg]]"
favicon: "[[favicon.png]]"
header_name: false
```

## Top menu

The pages in the menu, in order:

- **Add page** adds a folder or note from the site. A folder, like **Writing**, leads to the whole section.
- **Add link** adds a link to another website, like your shop.
- Each item's name is up to you: write the text that should show.
- Drag by the handle to change the order.

**Submenus**: drag an item on top of another. The item on top then opens a list with the ones below it, with no link of its own. **New submenu** creates an empty entry for you to drag items into. It's one level only.

With no items, the menu is automatic: the folders at the root of the site.

In Obsidian, one item per line. `Name | target` changes the item's text, an address becomes a link out, and `+ ` in front puts the item in the submenu of the line above:

```yaml
menu:
  - Home
  - Writing
  - Reading |
  - + Books | Books
  - + Magazines | Magazines
  - Shop | https://myshop.com
```

## Top button

A highlighted button to the right of the menu, like "Subscribe". Fill in the **Button text** and **Where it leads**. Leave the text empty to have no button.

```yaml
button_text: Subscribe
button_link: https://mynewsletter.com
```

## Social links

Under **Social links**, click the network (Instagram, X, LinkedIn, YouTube, Discord, GitHub, Facebook, WhatsApp or e-mail): the Studio fills in the start of the address, and you finish it with your profile. The icons show in the footer.

```yaml
social:
  - https://instagram.com/yourname
  - https://github.com/yourname
  - mailto:you@example.com
```

> [!tip] Want to see it in practice?
> Open the **_index** note in the site folder: this blog's menu is there. In the command line, `blup customize` asks the same questions and saves to the same properties.
