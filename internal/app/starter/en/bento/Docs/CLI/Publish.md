---
title: Publish
description: Cloudflare Pages or GitHub Pages, for free. One command for each update.
weight: 3
---
Run `blup publish`. The first time, choose where. Both are free.

![Blup launching the site](../../Attachments/blup-publicar.jpg)

## Cloudflare Pages

Fast worldwide, with an address like `mysite.pages.dev`.

1. Create a free account at dash.cloudflare.com, if you don't have one.
2. Blup opens the access key form. Check the **Cloudflare Pages: Edit** permission, create the key and paste it in the terminal.
3. Choose the project name: it becomes the address.

## GitHub Pages

An address like `yourname.github.io/mysite`.

1. Create a free account at github.com, if you don't have one.
2. Blup opens the key form already filled in. Under **Repository access**, choose **All repositories**, generate the key and paste it in the terminal.
3. Choose the repository name: it becomes the end of the address.

## From then on

`blup publish` sends only what changed and shows the address. Keys stay only on your computer; if one expires, Blup asks for another.

## Custom domain

To have the site answer at an address of yours, like `mysite.com` or `blog.mysite.com`, run `blup customize` and choose **Custom domain**. Blup:

1. asks for the domain;
2. connects it where the site is published, on Cloudflare Pages or GitHub Pages;
3. shows the exact DNS record to create where you bought the domain, if one is missing;
4. turns on HTTPS when everything is ready.

DNS can take from minutes to a few hours. To check, open **Custom domain** again and choose **Check if it is working**; every `blup publish` checks too. The same option changes the domain or goes back to the free address.

> [!tip] Root domain on Cloudflare Pages
> A root domain (`mysite.com`, without `www` or `blog` in front) on Cloudflare Pages has to be in your Cloudflare account, which is free. If you'd rather not, use `www.mysite.com`.
