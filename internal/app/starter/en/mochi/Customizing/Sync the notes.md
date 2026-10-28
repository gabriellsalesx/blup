---
title: Sync the notes
date: {{date-8}}
description: Your notes in a private GitHub repository, to use on another computer. No git needed.
tags: [Customizing, CLI]
image: "[[blup-sincronizar.jpg]]"
---
`blup sync` keeps a copy of the site's notes in a **private repository** on GitHub. On another computer, install Blup and sync: the notes arrive. Edit wherever you like and sync again.

You don't need to know git or install anything besides Blup.

## The first time

1. Run `blup sync` (or pick it in the menu, or answer "yes" at the end of the new site wizard).
2. Sign in with the same GitHub key you use to publish.
3. Choose the repository name. It is created **private**: only you can see it.

## Every time

Blup compares the notes here with the ones on GitHub and **shows what it will do** before doing it:

- what changed here is sent;
- what changed on another computer is received;
- what was deleted on one side is deleted on the other.

## When both sides changed

If the same note was edited here and on another computer, nothing is lost: your version stays, and the other one becomes a copy with "conflict" in its name. Compare them and delete the one you don't need.

> [!info] Nothing vanishes suddenly
> Notes deleted on another computer go to a trash folder outside the vault, and Blup tells you where it is. If the site folder suddenly looks empty (for example, if the vault moved), Blup stops and deletes nothing on GitHub.
