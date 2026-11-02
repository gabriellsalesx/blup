---
title: Post narration
description: A voice reads the post and the site shows a player at the top. Free on the computer or through the cloud.
weight: 2
---
Narration turns a post's text into an MP3 audio file. On the site, the post gets a **player** at the top, for people who would rather listen.

> [!info] Blup Studio only
> Narration is made in the Studio. Once it is ready, it is a regular file in Attachments and can be published any way you like.

## Pick the voice (once)

1. Open **Settings > AI and voice > Narration**.
2. For each language (Português, English, Español), pick the voice. They come in groups:
   - **On the computer, free**: the Kokoro voices. The first time, download the voice in **Settings > AI and voice > AI**, with the **Download local voice** button.
   - **Blup Cloud, included in the plan**: needs a paid plan.
   - **OpenRouter, paid**: needs your key.
3. **Play sample** plays a bit of the voice.

The voice works for all your sites. See the ways to pay in [AI in Blup Studio](AI%20in%20Blup%20Studio.md).

## Narrate a post

1. Open the post in the Studio.
2. In the **Properties**, in the **Narration** block, click **Make narration**.
3. The Studio shows the progress: getting the text ready, making the voice part by part, saving the audio.
4. Done: the length and the size show up. Listen in the preview's player, on the side.

The audio is saved in **Attachments**, and the note gets the `audio`, `audio_hash` and `audio_seconds` properties.

## What the voice reads

The title, the description and the text. It skips what makes no sense to hear: code blocks, math, images, tables and `%%` comments. Headings and list items get pauses.

## When the text changes

The Studio compares the text with the narrated one:

- **Narration up to date with the text.**: all good.
- **The text changed after the narration.**: click **Make it again** so the audio follows.

## Posts in other languages

A post's translation (in the `en`, `es` or `pt` folder) uses the voice of that language and gets its own audio, like `Post (es).mp3`. See [Multilingual sites](../Site%20setup/Multilingual%20sites.md).

> [!warning] Posts with a password are not narrated
> A public audio file would give the protected text away. That is why the Studio refuses to narrate posts with a password.

> [!tip] Make it a podcast
> On Mochi and Nori blogs, the narrations can become podcast episodes. See [Blog podcast](Blog%20podcast.md).
