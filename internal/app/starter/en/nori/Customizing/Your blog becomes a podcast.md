---
title: Your blog becomes a podcast
date: {{date-15}}
description: Post narrations become episodes, with a player on the blog and a feed for Spotify and Apple Podcasts.
tags: [Customizing, AI, Studio]
---
Narrated your posts? Then your blog already has a podcast ready. The **Podcast** section gathers the narrations as episodes, newest first, on a page with its own player, and publishes a feed for Spotify, Apple Podcasts and other apps.

## Turn on the section

In the Studio, click **Sections**, in the side menu, and add **Podcast**. The blog gets a **Podcast** folder, with a note you can edit, and the side menu gets the **Podcast** screen.

The episodes are the posts with a narration: see [Narrate a post as audio](../Writing/Narrate%20a%20post%20as%20audio.md). Each episode's cover is the post's cover. Hidden posts, drafts and password protected posts are left out.

## The podcast page

- A fixed player that moves on to the next episode by itself, even on a phone with the screen off.
- Back and forward 15 seconds, speed from 0.75 to 2 times, and the playlist.
- Each episode's notes (the post's text) open right there, without stopping the audio.
- Anyone who leaves and comes back finds the episode where they stopped.
- A button to copy the feed's address.

## The Podcast screen

The Studio's **Podcast** screen holds the show's details:

- **Podcast name** (empty, the site's name is used) and **Description**.
- **Author** and **Contact e-mail**. The e-mail is public in the feed: that is how Spotify and Apple confirm the podcast is yours.
- **Category**, from the Apple Podcasts list, and **Explicit content**.
- **Podcast cover**: a square from 1400 to 3000 pixels, as JPG or PNG.

The screen also shows how many episodes there are and the **Podcast feed**, with **Copy** and shortcuts to send it to Spotify and Apple Podcasts. Paste the address just once: new episodes arrive by themselves when you publish the site.

> [!warning] It needs a full address
> Spotify and Apple need the feed's full address. Once the site is published (through Blup Cloud, Cloudflare Workers or GitHub Pages), the full address shows on the Podcast screen.

> [!info] And the command line?
> The Podcast section is turned on in the Studio. Once it's on, the show's details live in the properties of the **_index** note in the Podcast folder, and the page and the feed keep being made when you publish from the command line.
