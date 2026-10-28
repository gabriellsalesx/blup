---
title: Passwords and hidden pages
description: Protect pages with a password and publish pages only people with the link can find.
weight: 4
---
## Password

Any page or post can have a password:

```yaml
password: my-password
```

Visitors see the title and a field for the password. The content is **encrypted** when the site is built (AES-256-GCM): without the password, no one can read it, not even in the page source. There is no server.

To use the same password on many pages, set a **general password** in `blup customize` and write:

```yaml
password: yes
```

The general password stays only on your computer, in blup.toml.

> [!warning] What stays visible
> Title, description and cover show up in lists. Images inside the page are not encrypted.

## Hidden pages

```yaml
hidden: true
```

The page is published, but leaves the lists, the tags and the site search, and asks Google not to show it. Anyone with the link can open it. Pairs well with a password.

## Drafts

```yaml
draft: true
```

A draft shows up only in the preview (`blup preview`) and is never published.
