---
title: Publish
description: Cloudflare Workers or GitHub Pages, for free. One command for each update.
weight: 3
---
Run `blup publish`. The first time, pick where: Cloudflare Workers or GitHub Pages. Both are free. Publishing with Blup Cloud is done in the Studio; see [Where to publish](../Publishing/Where%20to%20publish.md).

![Blup launching the site](../../Attachments/blup-publicar.jpg)

## Cloudflare Workers

Fast all over the world, with an address like `mysite.youraccount.workers.dev`.

1. Create a free account at dash.cloudflare.com, if you don't have one.
2. Blup opens the access key form. Check the **Account › Workers Scripts › Edit** permission, create the key and paste it in the terminal.
3. If the account doesn't have a workers.dev address yet, Blup asks for its name. It is chosen once and serves every site of the account.
4. Pick the site name: it starts the address. If the account already has another site with that name, Blup asks for another one.

## GitHub Pages

An address like `yourname.github.io/mysite`.

1. Create a free account at github.com, if you don't have one.
2. Blup opens the key form, already filled in. In **Repository access**, pick **All repositories**, generate the key and paste it in the terminal.
3. Pick the repository name: it becomes the end of the address.

## The next times

`blup publish` sends only what changed and shows the address. The keys are kept only on your computer; if one expires, Blup asks for another.

## Custom domain

With Cloudflare Workers and GitHub Pages, the site stays at the service's free address. A domain of your own, like `mysite.com`, is linked only through Blup Cloud, on the Max and Ultra plans, on the Studio's **Publish** screen. See [Custom domain](../Publishing/Custom%20domain.md).
