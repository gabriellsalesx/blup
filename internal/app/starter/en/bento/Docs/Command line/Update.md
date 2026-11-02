---
title: Update
description: Blup tells you when there's a new version and only updates if you want.
weight: 5
---
Once a day, when you open the menu, Blup checks for a new version. If there is one, it shows what changed and asks:

- **Update now**
- **Not now**
- **Don't remind me about this version**

To update at any time: `blup update`.

> [!note] What about Blup Studio?
> This command updates only the command line. The Studio looks for a new version in **Settings > About**. See [Installation](../Getting%20started/Installation.md).

## Go back

```sh
blup update --rollback
```

Goes back to the previous version, kept during the update.

## The themes

The themes come inside Blup: after updating, your sites get the new theme on the next preview or publish. Your notes don't change.

> [!info] No internet or in a hurry?
> The check gives up after 2 seconds and never stops Blup from opening.
