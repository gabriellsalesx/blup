---
title: Cookies and statistics
description: Google Analytics, Meta Pixel, Umami, Plausible and the cookie banner.
weight: 6
---
Blup does not add tracking cookies of its own. This part is for when you use tools to **count visits** or **measure ads**, and for **YouTube videos** in your notes.

## In Blup Studio

Click **Cookies and statistics** in the side bar.

1. Under **Tools**, click **Add tool** and pick one: Google Analytics, Google Tag Manager, Meta Pixel, Umami or Plausible.
2. Paste only the **ID** of each one. If the format does not look right, the Studio tells you.
3. Do you run Umami or Plausible on your own server? Fill in **Own server (optional)** with the script address.
4. Turn the **Cookie banner** on or off.
5. Under **Privacy policy**, pick **A page of the site (template to edit)** or **An address of mine**. You can add a **Contact e-mail**.
6. **Open the policy** opens the policy note in the editor.
7. Publish again for it to take effect on the site.

## From the command line

Run `blup customize` and pick **Cookies, visitor statistics and ads**. The questions are the same.

## Where to find each ID

| Tool | What to paste | Where to find it |
| --- | --- | --- |
| Google Analytics | Measurement ID, `G-...` | Admin › Data streams |
| Google Tag Manager | Container ID, `GTM-...` | At the top of the workspace, next to the container name |
| Meta Pixel | Pixel ID, numbers only | Events Manager › Data sources |
| Umami | Website ID | Settings › Websites › Edit |
| Plausible | Site domain | The domain added in Plausible |

**Google Tag Manager** can hold statistics tags and ad tags. With the banner on, it only loads after the person accepts **Statistics** or **Marketing**, and Blup reports what was accepted through Google's Consent Mode. Tags from other companies inside Tag Manager must be set up to respect consent.

## The banner

When it is on:

- On the first visit a notice shows up with **Accept all**, **Reject** and **Choose**.
- Under **Choose**, the person turns each group on or off: **Statistics** (Google Analytics, Umami, Plausible), **Marketing** (Meta Pixel), Google Tag Manager in both, and **Videos and outside content** (YouTube). The **Essential** ones (light or dark theme, signing in to comment) are always on.
- Nothing from third parties loads before the person accepts. The choice is kept in the browser for one year.
- The **Cookies** link in the footer opens the choices again at any time.

When it is off, the tools and the videos load right away, without asking.

## Privacy policy

- **A page of the site** (recommended): Blup creates the **Privacy policy** note, already written from the tools you picked, the videos, the comments and the hosting. Review and edit it as you like: it is yours and Blup never rewrites it. A comment at the top of the note reminds you what to check and does not show on the site.
- **An address of yours**, if the policy already lives somewhere else.

The page is linked from the cookie notice and the footer, stays out of the post lists and does not get comments. If you change the tools later, update the note.

## YouTube videos

Paste the video link as if it were an image:

```md
![How to publish](https://www.youtube.com/watch?v=XXXXXXXXXXX)
```

On the site it becomes the player, using the `youtube-nocookie.com` address. With the banner on, a **Load video** button shows until the person accepts videos.
