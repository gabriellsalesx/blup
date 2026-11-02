---
title: Sync the notes
date: {{date-8}}
description: The same notes on more than one computer, through Blup Cloud with end-to-end encryption or in a private GitHub repository.
tags: [Customizing, Studio]
image: "[[blup-sincronizar.jpg]]"
---
Syncing keeps a copy of the notes in the cloud. On another computer, open the Studio and sync: the notes arrive. Edit wherever you like and sync again.

In the Studio, click **Sync**, at the top. The first time, choose where to keep the notes:

- **Blup Cloud** (recommended): end-to-end encrypted. Not even Blup can read them.
- **GitHub**: in a private repository on your GitHub account. No need to know git.

## Through Blup Cloud

1. Click **Sign in with Blup**. Blup Cloud opens in the browser: check that the code is the same and approve.
2. Create the **notes password**. It scrambles the notes before they leave the computer.
3. Keep the **recovery key** that shows up. It opens the notes if you forget the password, and it won't show again.

On another computer, sign in with the same account and type the notes password (or the recovery key).

> [!warning] No one else has your password
> Not even Blup. If you lose both the password and the recovery key, the notes in the cloud can't be brought back. Keep the key somewhere safe, off the computer.

| Plan | Per month | Per year | Notes | How it syncs | Computers | Earlier versions |
| --- | --- | --- | --- | --- | --- | --- |
| Free | R$ 0 | R$ 0 | 50 MB | With the button | 1 | No |
| Pro | R$ 14.90 | R$ 149 | 1 GB | In real time | 2 | 1 month |
| Max | R$ 29.90 | R$ 299 | 10 GB | In real time | 3 | 12 months |
| Ultra | R$ 59.90 | R$ 599 | 50 GB | In real time | No limit | 12 months |

In real time, the notes sync by themselves while you write. On the paid plans, right clicking a note shows its **Earlier versions**: pick one and bring it back. The paid plans also publish sites through Blup Cloud and include AI to narrate and translate.

## Through GitHub

1. Connect your GitHub account, with the same key used to publish.
2. Blup creates a **private** repository: only you can see it.
3. Click **Check**. Blup compares the notes here with the ones on GitHub and **shows what it will do** before doing it: what changed here is sent, what changed on another computer is received, and what was deleted on one side is deleted on the other.
4. If everything looks right, click **Sync now**.

> [!info] In the command line
> Run `blup sync`. The command line syncs through GitHub, the same way.

## When both sides changed

If the same note was edited here and on another computer, nothing is lost: your version stays, and the other one becomes a copy with "conflict" in its name. Compare the two and delete the one you don't need.

> [!info] Nothing disappears all of a sudden
> Notes deleted on another computer go to a trash folder outside the notes folder. If the site folder suddenly shows up empty (for example, if it moved), Blup stops and deletes nothing in the cloud.
