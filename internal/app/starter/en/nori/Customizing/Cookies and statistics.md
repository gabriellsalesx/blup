---
title: Cookies and statistics
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, YouTube videos and the cookie banner.
tags: [Customizing, Studio]
---
Blup doesn't set any tracking cookies of its own. This part is for when you use tools to **count visits** or **measure ads**, and for **YouTube videos** in your notes.

## Ready made tools

In the side menu, click **Cookies and statistics**. Under **Tools**, click **Add tool**, pick which one and paste just its ID:

| Tool | What to paste | Where to find it |
| --- | --- | --- |
| Google Analytics | Measurement ID, `G-...` | Admin › Data streams |
| Google Tag Manager | Container ID, `GTM-...` | At the top of the workspace, next to the container's name |
| Meta Pixel | Pixel ID, numbers only | Events Manager › Data sources |
| Umami | Website ID | Settings › Websites › Edit |
| Plausible | The site's domain | The domain registered in Plausible |

Running Umami or Plausible on your own server? Also fill in **Own server (optional)** with the script's address.

**Google Tag Manager** can hold statistics and ad tags. With the banner, it only loads after the person accepts **Statistics** or **Marketing**, and Blup passes on what was accepted through Google's Consent Mode: Google's tags follow each choice. Tags from other companies inside Tag Manager need to be set up to respect consent.

## The banner

On the same screen, turn on the **Cookie banner**. When it's on:

- On the first visit a notice shows up with **Accept all**, **Reject** and **Choose**.
- Under **Choose**, the person turns each group on or off: **Statistics** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager in both, and **Videos and outside content** (YouTube). The **Essential** ones (light or dark theme, signing in to comment) are always on.
- Nothing from third parties loads before they accept. The choice is kept in the browser for a year.
- The **Cookies** link in the footer reopens the choices at any time.

When it's off, the tools and the videos load right away, without asking.

## Privacy policy

Still on the same screen, choose where the **Privacy policy** lives:

- **A page of the site (template to edit)**: Blup creates the **Privacy policy** note in the site folder, already written from the tools, the videos, the comments and the hosting. Click **Open the policy**, review it and edit as you like: it's yours and Blup never rewrites it. A comment at the top of the note reminds you what to check and doesn't show on the site.
- **An address of mine**, if the policy already exists somewhere else.

Also fill in the **Contact e-mail (optional)**. The page is linked from the cookie notice and the footer, stays out of the post lists and doesn't take comments. If you change the tools later, update the note.

## YouTube videos

Paste the video's link as an image:

```md
![How to publish](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

On the site it becomes the player, through the `youtube-nocookie.com` address. With the banner on, a **Load video** button shows until the person accepts videos.

To turn it all off, remove the tools and turn off the banner on the same screen.

> [!info] With Obsidian
> Run `blup customize` and choose **Cookies, visitor statistics and ads**. The terminal asks the same things: the tools, the IDs, the banner and the privacy policy.
