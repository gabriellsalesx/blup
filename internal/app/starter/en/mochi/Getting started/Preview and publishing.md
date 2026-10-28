---
title: Preview and publishing
date: {{date-2}}
description: See the site while you write and publish it for free on Cloudflare Pages or GitHub Pages.
tags: [Getting started, CLI]
image: "[[blup-publicar.jpg]]"
---
## The preview

Run `blup preview` (or pick the preview in the menu). The site opens in the browser and **updates by itself** every time you save a note in Obsidian. To stop, press Ctrl+C in the terminal.

The preview also shows drafts, so you can check them before publishing.

## Publish

Run `blup publish`. The first time, Blup asks where to publish. Both are free:

- **Cloudflare Pages**: fast worldwide, with an address like `yoursite.pages.dev`.
- **GitHub Pages**: an address like `yourname.github.io/yoursite`.

Blup opens the right page to create the access key, already filled in, and explains what to click. You paste the key in the terminal once; it stays only on your computer.

From then on, `blup publish` only sends what changed.

> [!tip] Publishing is always the same
> You wrote, checked the preview, ran `blup publish`. There is no server to maintain and no database.

## Custom domain

Want an address of your own, like `mysite.com`? Run `blup customize` and choose **Custom domain**. Blup connects the domain where the site is published, shows the DNS record to create where you bought the domain, and turns on HTTPS when everything is ready.

## What goes online

- Every note in the site folder, except:
  - notes with `draft: true`;
  - folders starting with a dot, the templates folder (Templates) and the writers folder, which only becomes profile data.
- The images and attachments the notes use, optimized.
- The search, the RSS feed, the sitemap and the share card of each page.
