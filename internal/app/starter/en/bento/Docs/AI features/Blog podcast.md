---
title: Blog podcast
description: On the Mochi and Nori themes, post narrations become episodes with a feed for Spotify and Apple Podcasts.
weight: 4
---
The **Podcast** is a section of blogs (the **Mochi** and **Nori** themes). It gathers the post narrations as episodes on a page with its own player and publishes a **feed** for Spotify, Apple Podcasts and other apps.

> [!info] Blog themes only, in the Studio
> Bento (this theme) does not have the Podcast section. On blogs, Blup Studio adds it. A blog that already has the section keeps building the page and the feed from the command line too.

## 1. Add the section

1. Open the blog in the Studio.
2. Click **Sections** and, next to **Podcast**, click **Add**.
3. The Studio creates the **Podcast** folder, and the **Podcast** screen shows up in the side bar.

## 2. Make the episodes

Each **post with a narration** is an episode. Narrate the posts you want (see [Post narration](Post%20narration.md)) and publish the site. The Podcast screen shows how many episodes there are.

Left out: drafts, hidden posts and posts with a password.

## 3. Fill in the show details

On the **Podcast** screen:

| Field | If left empty |
| --- | --- |
| Podcast name | The site name |
| Description | The text of the section's note |
| Author | The site author |
| Contact e-mail | No e-mail. Spotify and Apple use it to confirm the podcast is yours, and it is public in the feed |
| Category | Technology (the list is Apple Podcasts' one) |
| Explicit content | No |
| Podcast cover | The newest episode's cover, then the site logo |

The best cover is a square of 1400 to 3000 pixels, in JPG or PNG. The Studio tells you when the image will not work.

## 4. Send it to Spotify and Apple

The feed needs the site's full address, so **publish the site** first, with Blup Cloud, Cloudflare Pages or GitHub Pages (see [Where to publish](../Publishing/Where%20to%20publish.md)). With a custom domain, the feed uses your domain. Then the screen shows the **Podcast feed** of each language, with **Copy**.

- **Submit to Spotify**: in Spotify for Creators, choose "I already have a podcast" and paste the address. The confirmation code arrives at the contact e-mail.
- **Submit to Apple Podcasts**: in Apple Podcasts Connect, sign in with your Apple ID, create a show from the feed and wait for approval.

You only do this once: new episodes reach the apps by themselves when you publish the site.

## The podcast page on the site

- The list of episodes, newest first, with each post's cover.
- A fixed player that moves on to the next episode by itself, even on a phone with the screen off.
- Skip back and forward 15 seconds, speed (0.75 to 2), the play queue and the episode notes (the post's text), without leaving the page.
- The spot where the person stopped is saved on their device.
- A button to copy the feed.

> [!tip] In several languages
> On a blog in several languages, each language has its own feed, like `/podcast/feed.xml` and `/en/podcast/feed.xml`.
