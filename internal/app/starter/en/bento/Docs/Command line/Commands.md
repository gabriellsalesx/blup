---
title: Commands
description: One menu for everything, and one command for each action, in English or Portuguese.
weight: 1
---
The command line is the way to go for people who write in **Obsidian** and want to build and publish the site without Blup Studio. Install it by following [Installation](../Getting%20started/Installation.md).

Running just `blup` opens the **menu**: preview, publish, customize, sync, convert links, switch sites, create another one and language. It's the easiest way.

![Blup in the terminal](../../Attachments/blup-terminal.jpg)

## Each action has a command

| English | Portuguese | What it does |
| --- | --- | --- |
| `blup new` | `blup novo` | Creates a new site |
| `blup preview` | `blup ver` | Opens the preview in the browser |
| `blup publish` | `blup publicar` | Puts the site online |
| `blup customize` | `blup personalizar` | Menu, sections, social links, logo, favicon, author, custom domain, general password and comments |
| `blup sync` | `blup sincronizar` | Keeps the notes in a private GitHub repository |
| `blup convert` | `blup converter` | Swaps `[[wikilinks]]` for standard links |
| `blup build` | `blup gerar` | Builds the site into `public/`, with no questions |
| `blup language` | `blup idioma` | Changes Blup's language |
| `blup update` | `blup atualizar` | Updates Blup |
| `blup version` | `blup version` | Shows the version |
| `blup help` | `blup --help` | Shows the help |

## More than one site

Blup remembers every site you created. In the menu, **Switch to another site** picks another one. With commands, pass the site's folder: `blup publish ~/Sites/my-site`.

## No questions

`blup build` asks nothing: good for scripts and automations. The site ends up in the `public` folder inside the site's folder.

## Site inside a subfolder

Want the docs at `mysite.pages.dev/bento/`, next to a landing page at the root? Build the site for that address and copy it into the landing page's folder:

```sh
blup build --url https://mysite.pages.dev/bento/ --out ../my-landing/bento
```

- `--url` is the final address of the site, and it can be a subfolder. Links, images and search then work inside it.
- `--out` copies the finished site into another folder, replacing what Blup had put there before. To be safe, Blup never deletes a folder it didn't fill itself.
- Then publish the landing page's folder, for example with Cloudflare Pages connected to a GitHub repository: each `git push` updates the site.

> [!tip] In Portuguese too
> `blup gerar --url … --saida …` does the same.

## What only the Studio does

Some things have no command: note vaults, translation, narration, the podcast, the Feedback board, submenus, and publishing and syncing with Blup Cloud. A site that already uses these features keeps building and publishing normally from the terminal. See [Studio and command line](../Getting%20started/Studio%20and%20command%20line.md).
