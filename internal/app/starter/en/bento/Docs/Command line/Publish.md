---
title: Publish
description: Cloudflare Pages or GitHub Pages, for free. One command for each update.
weight: 3
---
Run `blup publish`. The first time, pick where: Cloudflare Pages or GitHub Pages. Both are free. Publishing with Blup Cloud is done in the Studio; see [Where to publish](../Publishing/Where%20to%20publish.md).

![Blup launching the site](../../Attachments/blup-publicar.jpg)

## Cloudflare Pages

Fast all over the world, with an address like `mysite.pages.dev`.

1. Create a free account at dash.cloudflare.com, if you don't have one.
2. Blup opens the access key form. Check the **Cloudflare Pages: Edit** permission, create the key and paste it in the terminal.
3. Pick the project name: it becomes the address.

## GitHub Pages

An address like `yourname.github.io/mysite`.

1. Create a free account at github.com, if you don't have one.
2. Blup opens the key form, already filled in. In **Repository access**, pick **All repositories**, generate the key and paste it in the terminal.
3. Pick the repository name: it becomes the end of the address.

## The next times

`blup publish` sends only what changed and shows the address. The keys are kept only on your computer; if one expires, Blup asks for another.

## Custom domain

To have the site answer at an address of your own, like `mysite.com` or `blog.mysite.com`, run `blup customize` and pick **Custom domain**. Blup:

1. asks for the domain;
2. connects the domain where the site is published, on Cloudflare Pages or GitHub Pages;
3. shows the exact DNS record to create where you bought the domain, if one is missing;
4. turns on HTTPS when everything is ready.

DNS can take from a few minutes to a few hours. To check, go back to **Custom domain** and pick **Check if it is working**; every `blup publish` checks too. With the same option you can switch domains or go back to the free address.

> [!tip] Root domain on Cloudflare Pages
> A root domain (`mysite.com`, with no `www` or `blog` in front) on Cloudflare Pages has to be in your Cloudflare account, which is free. If you'd rather not change that, use `www.mysite.com`.
