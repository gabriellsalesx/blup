---
title: Cookies and statistics
date: {{date-11}}
description: Google Analytics, Meta Pixel, Umami, Plausible, YouTube videos and the cookie banner.
tags: [Customizing, Studio]
---
Blup sets no tracking cookies of its own. This part is for when you use tools to **count visits** or **measure ads**, and for **YouTube videos** in your notes.

## Ready-made tools

Open **Cookies and statistics** in the side menu. Click **Add tool**, pick the tool and paste only its ID:

| Tool | What to paste | Where to find it |
| --- | --- | --- |
| Google Analytics | Measurement ID, `G-...` | Admin › Data streams |
| Google Tag Manager | Container ID, `GTM-...` | At the top of the workspace, next to the container name |
| Meta Pixel | Pixel ID, only numbers | Events Manager › Data sources |
| Umami | Website ID | Settings › Websites › Edit |
| Plausible | Site domain | The domain registered in Plausible |

Running Umami or Plausible on your own server? Fill in **Own server (optional)** with the script address.

**Google Tag Manager** can hold statistics and ads tags. With the banner, it only loads after people accept **Statistics** or **Marketing**, and Blup tells it what they accepted through Google's Consent Mode: Google's tags follow each choice. Tags from other companies inside Tag Manager must be set up to respect consent.

## The banner

On the same screen, turn on **Cookie banner**. On:

- On the first visit a notice shows **Accept all**, **Reject** and **Choose**.
- In **Choose**, people turn each group on or off: **Statistics** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), with Google Tag Manager in both and **Videos and outside content** (YouTube). **Essential** (light or dark theme, sign in to comment) is always on.
- Nothing from third parties loads before they accept. The choice stays in the browser for a year.
- The **Cookies** link in the footer reopens the choices anytime.

Off, the tools and the videos load at once, without asking.

## Privacy policy

When you turn on the banner or a tool, choose where the **Privacy policy** is:

- **A page of the site (template to edit)**, the recommended one: Blup creates the **Privacy policy** note in the site folder, already written from the tools you added, the videos, the comments and the hosting. Click **Open the policy**, review it and edit it freely: it's yours and Blup never rewrites it. A comment at the top of the note lists what to check and doesn't show on the site.
- **An address of mine**, if the policy already lives somewhere else.

You can also add a **Contact e-mail**. The page is linked from the cookie notice and the footer, stays out of the post lists and gets no comments. If you change the tools later, update the note.

## YouTube videos

Paste the video link as an image:

```md
![How to publish](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

On the site it becomes the player, from `youtube-nocookie.com`. With the banner on, a **Load video** button shows until people accept videos.

To turn it all off, remove the tools and turn off the banner on the same screen.
