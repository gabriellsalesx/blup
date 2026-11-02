---
title: Passwords and hidden pages
description: Protect pages with a password, hide pages from the lists and keep drafts.
weight: 7
---
![Blup with a padlock](../../Attachments/blup-senha.jpg)

## Password

Any page or post can ask for a password. Whoever opens it sees the title and a field for the password. The content is **encrypted on your computer** when the site is built (AES-256-GCM): without the password, nobody can read it, not even in the page's code.

### In Blup Studio

In the note's **Properties**, under **Password**, pick:

- **General password**: the same password for many posts. It lives in **Settings > General password** and works for all your sites.
- **Its own password**: a password for this post only, in **This post's password**.

### In the note's file

```yaml
password: my-password
```

To use the general password, write:

```yaml
password: yes
```

On the command line, `blup customize` also sets a general password for that site only. General passwords stay on your computer only, never on the site.

> [!warning] What stays visible
> Title, description and cover show in the lists. Images inside the page are not encrypted. Posts with a password are also not narrated and do not go into the podcast, because the audio would give the text away.

## Hidden pages

In the Studio, turn on **Hidden** in the properties. In the note's file:

```yaml
hidden: true
```

The page is published, but it leaves the lists, the tags and the site search, and asks Google not to show it. Anyone with the link opens it as usual. It goes well with a password.

## Drafts

In the Studio, turn on **Draft**. In the note's file:

```yaml
draft: true
```

A draft shows only in the preview and is never published.

## Bits only you can see

Text between `%%` never goes to the site either. See [Math, code and comments](../Writing/Math,%20code%20and%20comments.md).
