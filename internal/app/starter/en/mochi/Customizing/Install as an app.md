---
title: Install as an app
date: {{date-10}}
description: The blog installs as an app on phones and computers and opens offline.
tags: [Customizing]
---
Every site Blup publishes can be **installed as an app**, on phones and computers, and **opens offline**. Nothing to turn on.

## Install

- **Android and desktop (Chrome, Edge):** an **Install app** button shows in the site's footer, and the browser also offers the install icon in the address bar.
- **iPhone and iPad (Safari):** tap **Share**, then **Add to Home Screen**.

The app opens full screen, with the site's icon and name.

## The icon

Blup makes the icon of every device from the site's **favicon**: Android, iPhone, Android's circle-cropped icon and `favicon.ico`. To change it, change the favicon (see Set up the site from Obsidian). A square image of at least 512 pixels, or an SVG, looks best.

## Offline

On the first visit, the browser keeps every page, the styles and the fonts. From then on, the site opens even without internet.

- **Photos** are kept as people see them, so they don't all download at once.
- **Comments and reactions** need the internet; without it, the rest of the page works as usual.
- After each publish, the browser fetches the new version by itself.

The preview (`blup preview`) doesn't become an app, since it changes all the time; only the published site does.
