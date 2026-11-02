---
title: Create your first site
description: One step at a time, in the Studio or in the terminal. You can change everything later.
weight: 2
---
## In Blup Studio

On the home screen, click **Create a new site**. The Studio asks one question at a time, with **Continue** and **Back** at the bottom.

1. **What kind of site do you want to make?**
   - **Personal blog**: a blog, journal, newsletter or writing portfolio.
   - **Product site**: to present an app, SaaS, API or project, with docs and news.
2. **Which theme fits your site?**
   - Blog: **Mochi** (a featured post and a grid of posts) or **Nori** (a single column, with the text up front).
   - Product: **Bento**, this one.
3. **Which parts will the site have?** (Bento only): Docs, Roadmap, Changelog, Help center and Blog. Each part comes with example notes that teach you how to use it. Pick at least one.
4. **What's the site called?** The name, an optional description, your name (it signs the posts) and the **site language**.
5. **Where do the notes live?**
   - **In a new folder**: the simplest way. You can open it in Obsidian later.
   - **In my Obsidian vault**: the site uses a folder of the vault you already have, or the whole vault.
   - **In a folder I choose**: notes from VS Code, Foam or any other editor.
6. **All set?** Check the summary and click **Create the site**.

Done: click **Open the site**. The editor opens with the preview beside it. See [The Studio window](../Blup%20Studio/The%20Studio%20window.md).

> [!tip] Adjust Obsidian
> If the notes are in an Obsidian vault, the Studio offers **Adjust Obsidian so links work on the site**. This way new links between notes are saved in a format the web understands. Blup keeps a copy of your settings first.

> [!note] Where the site's files live
> In **Where to keep the site's files**, you pick the folder for the settings and the generated pages. It sits outside your notes and you never need to open it.

### Already have a Blup site?

On the home screen, **I already have a Blup site** opens a folder that is already a site, shows the sites **Opened lately** and, with your GitHub account connected, offers to **Bring** the sites kept there. With a Blup Cloud account, the Studio also lists your sites kept in the cloud.

## From the command line

Run `blup new` (or just `blup`, the first time). Each question shows up on its own and comes with a suggested answer. When in doubt, press Enter.

1. **Obsidian vault**: Blup lists the vaults it found.
2. **Folder**: the whole vault or just one folder, like "Site".
3. **Theme**: **Mochi** (blog) or **Bento** (product site).
4. **Sections** (Bento only): tick the ones you want.
5. **Site name**, **one sentence about it** and **your name**.
6. **Site language** and **where to save** the site's files on your computer.
7. **Summary** to confirm.
8. **Obsidian adjustment**: Markdown links with relative paths.
9. **Keep the notes on GitHub**: optional. See [Sync](../Command%20line/Sync.md).
10. **Open the preview**.

The command line also creates a **Templates** folder with note templates for Obsidian.

## What gets created

- **In your notes**: this example site (if the folder is empty), your author note and the folders of the sections you picked.
- **In a separate folder**: **blup.toml** and the files the site generator (Hugo) uses.

> [!tip] Already had notes in the folder?
> Nothing is deleted or overwritten. Blup only creates the **_index** note if it is missing, plus the folders of the sections you picked.
