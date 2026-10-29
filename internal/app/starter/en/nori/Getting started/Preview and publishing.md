---
title: Preview and publishing
date: {{date-2}}
description: See the site while you write and publish it for free on Cloudflare Pages or GitHub Pages.
tags: [Getting started, Studio]
image: "[[blup-publicar.jpg]]"
---
## The preview

The **Preview** panel, on the right of the editor, shows the site as it will look online. It **updates by itself** every time the note is saved, and saving is automatic. You can turn it on and off whenever you want.

The preview also shows drafts, so you can check them before publishing.

## Publish

Open **Publish** in the side menu. The first time, choose where to publish. Both are free:

- **Cloudflare Pages**: fast worldwide, with an address like `yoursite.pages.dev`.
- **GitHub Pages**: an address like `yourname.github.io/yoursite`.

Then connect the account once:

1. Click **Open in the browser**: the form to create the access key opens with the right permissions already ticked.
2. Create the key and copy it.
3. Paste it in the Studio and click **Connect**. The key stays only on your computer.

Click **Publish now**. From then on, publishing only sends what changed.

> [!tip] Publishing is always the same
> You wrote, checked the preview, clicked **Publish now**. There is no server to maintain and no database.

## Custom domain

Want an address of your own, like `mysite.com`? In **Publish**, find **Custom domain**, type the address and click **Connect domain**. Blup connects it where the site is published and, if a DNS record is missing, shows exactly what to create where you bought the domain. Then click **Check**.

## What goes online

- Every note in the site folder, except:
  - notes marked as **Draft** in their properties;
  - folders starting with a dot and the writers folder, which only becomes profile data.
- The images and attachments the notes use, optimized.
- The search, the RSS feed, the sitemap and the share card of each page.
