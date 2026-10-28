---
title: Password protected posts
date: {{date-5}}
description: Protect a post with a password, with no server and no one reading it in the page source.
tags: [Obsidian, Features]
image: "[[blup-senha.jpg]]"
---
Any post can have a password. Visitors see the title and a field to type the password; the text only shows up with it.

## How to protect

Add the **password** property to the note:

```yaml
password: my-password
```

To avoid repeating the same password on many posts, set a **general password** in `blup customize` and use:

```yaml
password: yes
```

See it working: [A locked post](A%20locked%20post.md). The password is `blup`.

## Is it safe?

The text is **encrypted** when the site is built (AES-256). Without the password, no one can read it, not even by looking at the page source. There is no server storing anything.

> [!warning] What is not protected
> Images inside the post are not encrypted. The title, description and cover stay visible, because they appear in lists.

## Hide from lists

For a post that only people with the link should find, also use `hidden: true`: it leaves the home page, the tags and Google, but stays online.
