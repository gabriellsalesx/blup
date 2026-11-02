---
title: Custom domain
description: The site at your own address, like mysite.com, with HTTPS.
weight: 3
---
Instead of the free address (`site.pages.dev` or `yourname.github.io`), the site can answer at an address of your own, like `mysite.com` or `blog.mysite.com`. You need a domain you bought (Namecheap, GoDaddy, Cloudflare...).

> [!info] Where it already works
> With **Cloudflare Pages** and **GitHub Pages**, a custom domain already works. For sites published with **Blup Cloud**, it comes with the Max and Ultra plans.

## With Blup Studio

The **Custom domain** block shows up on the **Publish** screen after the site is online with Cloudflare or GitHub.

1. Type just the address, like `mysite.com`, and click **Connect domain**.
2. The Studio connects the domain where the site is published.
3. If a step is missing, it shows the exact **DNS record** (Type, Name and Value, with **Copy**). Create that record on the site where you bought the domain.
4. Click **Check**. It can take from a few minutes to a few hours.
5. When it says **Working**, publish again so the site's links use the new address.

**Change** switches to another domain. **Back to the free address** turns off the domain for this site (it is still yours).

## With the command line

Run `blup customize` and pick **Custom domain**. Blup asks for the domain, connects it where the site is published, shows the DNS record if one is missing and turns on HTTPS. To check, go back to **Custom domain** and pick **Check if it is working**; every `blup publish` checks too.

> [!tip] Root domain on Cloudflare Pages
> A root domain (`mysite.com`, with no `www` or `blog` in front) on Cloudflare Pages has to be in your Cloudflare account, which is free: add the domain at dash.cloudflare.com and, where you bought it, swap the name servers for the ones Cloudflare shows. If you'd rather not change that, use `www.mysite.com`.

> [!note] A podcast needs a domain
> The blog podcast feed only works in podcast apps with the site's full address. See [Blog podcast](../AI%20features/Blog%20podcast.md).
