---
title: Sync the notes
date: {{date-8}}
description: Your notes in a private GitHub repository, to use on another computer. No git needed.
tags: [Customizing, Studio]
image: "[[blup-sincronizar.jpg]]"
---
**Sync the notes** keeps a copy of the site's notes in a **private repository** on GitHub, where only you can see it. On another computer, install Blup Studio and sync: the notes arrive. Edit wherever you like and sync again.

You don't need to know git or install anything besides the Studio.

## The first time

1. Open **Publish** in the side menu and find **Sync the notes**.
2. Connect your GitHub account: click **Open in the browser**, create the key, paste it and click **Connect**. If you already publish on GitHub Pages, the account is already connected.
3. Click **Sync now**.

## Every time

Click **Check** to see what will happen before syncing: how many notes will be sent and how many will be received. Then click **Sync now**:

- what changed here is sent;
- what changed on another computer is received;
- what was deleted on one side is deleted on the other.

When there is nothing to do, the Studio shows **All up to date**.

## When both sides changed

If the same note was edited here and on another computer, nothing is lost: both versions are kept, and the Studio shows which note had the conflict. Compare them and delete the one you don't need.

> [!info] Nothing vanishes suddenly
> If the site folder suddenly looks empty (for example, if it was moved), Blup stops and deletes nothing on GitHub.
