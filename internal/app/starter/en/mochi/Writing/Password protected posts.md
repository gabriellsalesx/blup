---
title: Password protected posts
date: {{date-5}}
description: Protect a post with a password, with no server and no one reading it through the page's code.
tags: [Writing, Features]
image: "[[blup-senha.jpg]]"
---
Any post can have a password. Whoever opens it sees the title and a field to type the password; the text only shows up with it.

See it working: [A locked post](A%20locked%20post.md). The password is `blup`.

## In the Studio

In the post's **Properties** panel, turn on **Password** and choose:

- **Its own password**: type the password for this post.
- **General password**: one password for many posts. It lives in the Studio's **Settings** and works for all your sites. If you want, a site can have its own password, which works only for that site.

## In the note's file

You can also write the **password** property by hand:

```yaml
password: my-password
```

To use the general password, write:

```yaml
password: yes
```

In the command line, the general password is set in `blup customize`.

## Is it safe?

The text is **encrypted** on your computer when the site is built (AES-256). Without the password, no one can read it, not even by looking at the page's code. There is no server keeping anything, and the password never goes to the site.

> [!warning] What isn't protected
> Images inside the post aren't encrypted. The title, the description and the cover stay visible, because they show up in lists. Password protected posts also don't get narration and aren't part of the podcast, because the audio would give the text away.

## Hide from lists

For a post that only people with the link should find, also turn on **Hidden** (`hidden: true`): it leaves the home page, the tags and Google, but stays online.
