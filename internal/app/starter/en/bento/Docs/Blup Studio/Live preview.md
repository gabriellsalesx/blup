---
title: Live preview
description: The real site next to the editor, updated while you write.
weight: 3
---
When you open a site, the Studio turns on the preview next to the editor. It is your site, built on your computer, just like what goes live.

## How it works

- **Write it, see it**: every saved note updates the preview on its own, without flashing and keeping your place on the page.
- **The preview follows the note**: open another note in the editor and the preview goes to its page.
- **And the other way around**: click a link inside the preview and the note for that page opens in the editor.
- **Settings right away**: name, logo, menu and social links saved on the Site screen show in the preview without reloading.
- **Drafts show up** in the preview (and never on the published site).
- If the note has a narration, the player shows at the top of the post.

## Turn it on and off

- The **Preview** button at the top, or `Ctrl+Alt+P`, turns it on and off.
- In **Settings > Editor**, **Open with the preview on** decides whether the preview starts on.
- Drag the divider between the editor and the preview to change their widths. The Studio remembers.

> [!info] Only on your computer
> The preview runs only on your computer. Nobody else sees it, and nothing goes to the internet until you publish.

> [!note] No audio player on Linux?
> Some Linux systems are missing an audio component (GStreamer). In that case the preview hides the player, but the published site shows it as usual.

## In Obsidian

Without the Studio, run `blup preview` in the terminal: the preview opens in your browser and updates with every saved note. See [Preview](../Command%20line/Preview.md).
