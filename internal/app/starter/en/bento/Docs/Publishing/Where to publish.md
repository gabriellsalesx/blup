---
title: Where to publish
description: Blup Cloud, Cloudflare Workers or GitHub Pages. Two of them are free.
weight: 1
---
Blup builds the site on your computer and sends the finished pages to one of these places:

| Destination | Price | Address | What you need |
| --- | --- | --- | --- |
| **Blup Cloud** | On paid plans (Pro and up) | `yoursite.getblup.site` | Just your Blup account. Nothing to set up |
| **Cloudflare Workers** | Free | `yoursite.youraccount.workers.dev` | A free Cloudflare account |
| **GitHub Pages** | Free | `username.github.io/site` | A free GitHub account |

![Blup launching the site](../../Attachments/blup-publicar.jpg)

## Which one?

- **Want the simplest?** Blup Cloud: you pick the address and click Publish. See [Sites on Blup Cloud](../Blup%20Cloud/Sites%20on%20Blup%20Cloud.md).
- **Want it free and fast all over the world?** Cloudflare Workers.
- **Already use GitHub?** GitHub Pages.

With all three, the site is fast, has HTTPS and works as an app.

## Custom domain

Want `mysite.com` instead of the service's address? A custom domain is linked only through Blup Cloud, on the Max and Ultra plans. With Cloudflare and GitHub, the site stays at the free address. See [Custom domain](Custom%20domain.md).

## What goes online

A site is public by nature. Only what you published goes up:

- Drafts (`draft: true`) stay out.
- Comments between `%%` stay out.
- Of the attachments, only the ones some note uses go up.
- Posts with a password leave your computer encrypted.
- Notes in note vaults are never published.

## How to publish

- **In Blup Studio**: see [Publish from the Studio](Publish%20from%20the%20Studio.md).
- **On the command line**: see [Publish](../Command%20line/Publish.md). Blup Cloud only works from the Studio.
