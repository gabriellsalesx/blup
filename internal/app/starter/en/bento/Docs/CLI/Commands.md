---
title: Commands
description: One menu for everything, and one command per action, in English or Portuguese.
weight: 1
---
Running just `blup` opens the **menu**: preview, publish, customize, sync, switch site, create another and language. It's the easiest way.

![Blup at the terminal](../../Attachments/blup-terminal.jpg)

## Each action has a command

| English | Portuguese | What it does |
| --- | --- | --- |
| `blup new` | `blup novo` | Creates a new site |
| `blup preview` | `blup ver` | Opens the preview in the browser |
| `blup publish` | `blup publicar` | Puts the site online |
| `blup customize` | `blup personalizar` | Menu, sections, social links, logo, favicon, author, custom domain, general password and comments |
| `blup sync` | `blup sincronizar` | Keeps the notes in a private GitHub repository |
| `blup convert` | `blup converter` | Turns `[[wikilinks]]` into standard links |
| `blup build` | `blup gerar` | Builds the site into `public/`, with no questions |
| `blup language` | `blup idioma` | Changes Blup's language |
| `blup update` | `blup atualizar` | Updates Blup |
| `blup version` | `blup version` | Shows the version |

## More than one site

Blup remembers every site you created. In the menu, **Switch site** picks another. With commands, pass the site folder: `blup publish ~/Sites/my-site`.

## No questions

`blup build` asks nothing: good for scripts and automation. The site goes to the `public` folder inside the site folder.

## A site inside a subfolder

Want the docs at `mysite.pages.dev/bento/`, next to a landing page at the root? Build the site for that address and copy it into the landing page folder:

```sh
blup build --url https://mysite.pages.dev/bento/ --out ../my-landing/bento
```

- `--url` is the site's final address, and it can be a subfolder. Links, images and search work inside it.
- `--out` copies the finished site to another folder, replacing what Blup put there before. To be safe, Blup never wipes a folder it didn't fill.
- Then publish the landing page folder, for example with Cloudflare Pages connected to a GitHub repository: every `git push` updates the site.

> [!tip] In Portuguese too
> `blup gerar --url … --saida …` does the same.
