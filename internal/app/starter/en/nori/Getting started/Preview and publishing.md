---
title: Preview and publishing
date: {{date-2}}
description: See the site while you write and publish it through Blup Cloud, Cloudflare Pages or GitHub Pages.
tags: [Getting started, Studio]
image: "[[blup-publicar.jpg]]"
---
## The preview

In the Studio, click **Preview**, at the top. The site shows up next to the editor and **updates by itself** while you write. Switch notes and the preview goes to that note's page; click a link in the preview and the editor opens the note for that page.

The preview also shows drafts, so you can check them before publishing.

> [!info] In the command line
> Run `blup preview`. The site opens in the browser and updates every time you save a note. To stop, press Ctrl+C in the terminal.

## Where to publish

Click **Publish**, at the top. The first time, choose where the site will live:

- **Blup Cloud**: an address like `yoursite.getblup.site`, with nothing to set up. Comes with the paid plans, starting with Pro.
- **Cloudflare Pages**: free, with an address like `yoursite.pages.dev`. You need a Cloudflare account.
- **GitHub Pages**: free, with an address like `yourname.github.io/yoursite`. You need a GitHub account.

### Through Blup Cloud

Sign in with your Blup account and type the address you want. While you type, the Studio checks whether the name is free and, if it isn't, suggests similar ones. The address stays fixed after the first publish.

Each plan has a number of sites: Pro has 1, Max has 5 and Ultra has 10. Custom domains for Blup Cloud sites are on the way, on the Max and Ultra plans.

### Through Cloudflare or GitHub

Connect the account once:

1. Click **Connect the account** and then **Open in the browser**. The key form opens with the right permissions already set.
2. Create the key and copy it.
3. Paste the key into the Studio and click **Connect**. It is kept only on your computer.

Done: click **Publish now**. When it finishes, the Studio shows the site's address.

> [!info] In the command line
> Run `blup publish`. The first time it asks where to publish (Cloudflare Pages or GitHub Pages), opens the key page and asks you to paste the key in the terminal.

From then on, only what changed is sent.

> [!tip] Publishing is always the same
> You wrote, checked the preview, clicked Publish. There is no server to maintain and no database.

## Custom domain

Want an address of your own, like `mysite.com`? With the site online through Cloudflare or GitHub, go to the **Publish** screen, under **Custom domain**, and type the address. Blup connects the domain where the site is published and, if something is missing, shows the DNS record to create where you bought the domain. In the command line, the way is `blup customize` and then **Custom domain**.

## What goes online

The site is public: anyone with the address can read it. That is why only what you published goes:

- All the notes in the site folder, except the ones marked as **Draft** (`draft: true`).
- Comments between `%%` are left out.
- Only the attachments the notes use, optimized.
- Password protected posts leave your computer encrypted.
- The search, the RSS feed, the sitemap and the share card for each page.
