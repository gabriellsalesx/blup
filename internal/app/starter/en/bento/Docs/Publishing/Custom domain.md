---
title: Custom domain
description: The site at your own address, like mysite.com, with HTTPS. With Blup Cloud, on the Max and Ultra plans.
weight: 3
---
Instead of `yoursite.getblup.site`, the site can answer at an address of your own, like `mysite.com` or `blog.mysite.com`. You need a domain you bought (Namecheap, GoDaddy, Cloudflare...).

> [!info] Only with Blup Cloud
> A custom domain is linked through **Blup Cloud**, on the **Max** and **Ultra** plans. Sites published with **Cloudflare Workers** or **GitHub Pages** stay at the service's free address (`name.youraccount.workers.dev` or `yourname.github.io`). To use a domain of your own, publish the site with Blup Cloud: see [Sites on Blup Cloud](../Blup%20Cloud/Sites%20on%20Blup%20Cloud.md).

## With Blup Studio

The **Own domain** block shows up on the **Publish** screen after the site is online with Blup Cloud.

1. Type just the domain, without `https://`, like `mysite.com`, and click **Link**.
2. The Studio shows the **DNS records** that are missing (Type, Name and Value, with **Copy**): a TXT record that proves the domain is yours and the CNAME that points it to Blup Cloud. Create those records on the site where you bought the domain.
3. Click **Check**. While the screen is open, the Studio checks again every minute. DNS can take from a few minutes to a few hours.
4. When it says **Active** and **HTTPS active**, the Studio publishes the site again on its own, so the links use the new address.

The `yoursite.getblup.site` address keeps working. **Remove** turns off the domain for this site, which stays only at `yoursite.getblup.site`.

> [!tip] Root domain
> A root domain (`mysite.com`, with no `www` or `blog` in front) needs a DNS that accepts a CNAME at the root (CNAME flattening or ALIAS). If yours doesn't, use `www.mysite.com`.

> [!warning] Changed plans?
> When you leave Max or Ultra, the domain keeps working for 14 days. Go back to one of those plans before then and nothing changes. After that, the site stays only at `yoursite.getblup.site`, and to use the domain again you have to link it again.

> [!note] A podcast needs the full address
> The blog podcast feed only works in podcast apps with the site's full address. With a custom domain, the feed uses your domain. See [Blog podcast](../AI%20features/Blog%20podcast.md).
