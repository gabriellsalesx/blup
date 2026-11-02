---
title: Social links
description: The footer icons, with a button for each network.
weight: 3
---
The site's profiles show as icons in the footer. Authors can have their own social links too, on each one's page.

## In Blup Studio

On the **Site** screen, in the **Social links** block:

1. Click the network's button: **Instagram**, **X**, **LinkedIn**, **YouTube**, **Discord**, **GitHub**, **Facebook**, **WhatsApp** or **e-mail**.
2. The Studio adds a line with the start of the address, like `https://instagram.com/`, and puts the cursor at the end.
3. Finish it with your profile (or the e-mail).

If the rest of the address is missing, the Studio warns you: a line with only the start does not go to the site.

On the **Authors** screen, each person's profile has the same buttons.

## In Obsidian

In the **_index** note of the site folder:

```yaml
social:
  - https://instagram.com/acme
  - https://github.com/acme
  - https://wa.me/5585999999999
  - mailto:hi@acme.com
```

Blup recognizes the network from the address and uses the right icon. Addresses it does not know get a link icon.

## From the command line

`blup customize` > social links asks for the addresses one by one.
