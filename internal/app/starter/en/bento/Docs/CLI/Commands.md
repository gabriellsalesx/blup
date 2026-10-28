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
| `blup customize` | `blup personalizar` | Menu, sections, social links, logo, favicon, author, custom domain and general password |
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
