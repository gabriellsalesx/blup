---
title: Sync
description: The site's notes in a private GitHub repository, to use on another computer. No git needed.
weight: 4
---
`blup sync` keeps a copy of the notes in the site folder in a **private repository** on GitHub. On another computer, install Blup and sync: the notes arrive. Edit wherever you like and sync again.

![Two computers syncing](../../Attachments/blup-sincronizar.jpg)

## The first time

1. Run `blup sync`, pick it in the menu or answer "yes" at the end of the new site wizard.
2. Use the same GitHub key you use to publish.
3. Choose the repository name. It is created **private**. If the repository already exists and is public, Blup warns you first.

## Every time

Blup compares and **shows first** how many notes it will send, receive, delete on GitHub and move to the trash here. It only goes on if you confirm.

| Situation | What happens |
| --- | --- |
| Changed only here | It's sent |
| Changed only on another computer | It's received |
| Deleted on one side | It's deleted on the other |
| Deleted on one side and edited on the other | The edited version stays |
| Edited on both sides | Both stay: yours and a copy with "conflict" in its name |

## Safety

- Notes deleted on another computer go to a **trash folder outside the vault**; Blup tells you where.
- If the site folder looks empty but had notes before, Blup **stops** and deletes nothing on GitHub.
- Folders starting with a dot (like `.obsidian`) are not synced.
- Files over 100 MB are left out, with a warning.
