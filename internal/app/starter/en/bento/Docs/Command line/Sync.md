---
title: Sync
description: The site's notes in a private GitHub repository, to use on another computer. No git needed.
weight: 4
---
`blup sync` keeps a copy of the notes in the site's folder in a **private repository** on GitHub. On another computer, install Blup and sync: the notes arrive. Edit wherever you want and sync again.

![Two computers syncing](../../Attachments/blup-sincronizar.jpg)

> [!info] What about Blup Cloud?
> Encrypted sync with Blup Cloud is done in Blup Studio. See [Sync your notes](../Blup%20Cloud/Sync%20your%20notes.md).

## The first time

1. Run `blup sync`, pick it in the menu or answer "yes" at the end of the new site wizard.
2. Use the same GitHub key you use for publishing.
3. Pick the repository name. It is created **private**. If the repository already exists and is public, Blup warns you first.

## Every time

Blup compares and **shows you first** how many notes it will send, receive, delete on GitHub and move to the trash here. It only goes on if you confirm.

| Situation | What happens |
| --- | --- |
| Changed only here | It is sent |
| Changed only on another computer | It is received |
| Deleted on one side | It is deleted on the other |
| Deleted on one side and edited on the other | The edited version stays |
| Edited on both sides | Both stay: yours and a copy with "conflict" in its name |

## Safety

- Notes deleted on another computer go to a **trash outside your notes**, in the site's folder; Blup shows you where.
- If the site's folder shows up empty but it had notes before, Blup **stops** and deletes nothing on GitHub.
- Folders that start with a dot (like `.git`) are not synced.
- Files over 100 MB stay out, with a warning.
