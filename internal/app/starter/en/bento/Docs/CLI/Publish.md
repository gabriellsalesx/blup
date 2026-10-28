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

Write the domain in the site folder's **blup.toml**:

```toml
domain = "mysite.com"
```

- On GitHub Pages, Blup sets the domain up by itself; point the domain's DNS to GitHub.
- On Cloudflare Pages, Blup reminds you: add the domain under **Custom domains** in the project, in the Cloudflare dashboard.
