# Instructions for AI agents in this vault

This folder is a Blup site with the Bento theme: a product site with
docs, roadmap, changelog, help center, blog and feedback. Blup turns
these Markdown notes into a website with Hugo. The person writes and
publishes in Blup Studio (or edits the same files in Obsidian).

Write notes in {{LANGUAGE}}.

## Golden rules

- Only create or change notes when the person asks.
- Do not delete, rename or move folders or notes unless asked.
- Never run `hugo`, `blup` or any build, preview or publish command. The
  Studio preview updates by itself when a note is saved. To publish, the
  person uses Publish in Blup Studio.
- Never use em dashes or en dashes in the text you write.

## How the folder becomes the site

- Every `.md` note becomes a page. Every folder becomes a section.
- A folder's `_index.md` is the section's own page.
- A top-level folder is a special section when its `_index.md` has the
  `blup` property. Only the top-level folder counts: everything inside it
  follows its kind. The folder name does not matter.

| `blup:` | Section | Each note is |
| --- | --- | --- |
| `docs` | Docs with side menu, "On this page" and previous/next | A doc page |
| `roadmap` | Board with three columns | A card |
| `changelog` | Versions, newest first | A version |
| `help` | Help center with search and collections | An article |
| `blog` | Blog, like the Mochi theme | A post |
| `feedback` | Request board (added by the Studio) | Not used |
| `authors` | Author profiles, not pages | An author |

- Notes at the top of the site folder, and folders without a `blup`
  kind, become plain pages.
- With no `menu` set, the header shows the special sections, ordered by
  their `_index.md` `weight`. The home page shows one card per section.
  With a single section, or the section chosen as Home page in the
  Studio's Sections screen, the home shows that section itself and its
  links lead to `/`. Its pages keep their addresses.

Starter folder names (any name works): `Docs`, `Roadmap`, `Changelog`,
`Help`, `Blog`, `Feedback`, `Writers` (en); `Documentação`, `Ajuda`,
`Autores` (pt); `Documentación`, `Ayuda`, `Autores` (es). Find a
section by its `blup` property, not by its name.

Left out of the site:

- Files and folders whose name starts with a dot (`.obsidian`, `.blup`...).
- The authors folder, the Obsidian templates folder, and `AGENTS.md`,
  `CLAUDE.md`, `GEMINI.md`, `INSTRUCTIONS.md`.
- Notes with `publish: false` or `private: true`, and their translations.
- Drafts (`draft: true`): only in the Studio preview.
- Text between `%%` and `%%` (Obsidian comments).

## Common front matter keys

| Key | What it does |
| --- | --- |
| `title` | Page title. Without it: the first `# Heading`, else the file name (a `01 - ` prefix is removed). |
| `linkTitle` | Short name in the menu and home cards (on a section's `_index.md`). |
| `description` | Line under the title, cards, Google. Without it, the start of the text. |
| `weight` | Order (lower first): docs pages, groups, roadmap cards, help articles and collections, sections. |
| `date` | Date like `2026-10-29`. Sorts changelog and blog. |
| `tags` | List of topics. Each tag gets a page. |
| `draft` | `true`: only in the preview. |
| `hidden` | `true`: published but out of lists, search and Google. |
| `noindex` | `true`: asks Google not to index the page. |
| `publish` / `private` | `publish: false` or `private: true`: never published. |
| `password` | Asks for a password to read (see below). |

In Bento, the date comes only from `date`, `publishDate` or `created`
(never from the file name or file time). Without a `weight`, docs follow
the file name.

## Docs (`blup: docs`)

- The docs folder's `_index.md` is the opening page.
- Each note is a page. Each subfolder is a group in the side menu, with
  the `title` of its `_index.md`.
- Order: `weight`, then the file name. Give groups a `weight` too.
- `##` and `###` headings make the "On this page" list.

```yaml
---
title: Install
description: Get the app running in two minutes.
weight: 1
---
```

## Roadmap (`blup: roadmap`)

One note per card:

```yaml
---
title: Dark mode
description: A dark theme for the whole app.
status: planned
category: Design
weight: 1
---
```

- `status` picks the column: planned (default), in progress, done. It
  also accepts `doing`, `started`, `shipped`, `completed`, and the
  Portuguese and Spanish words (`em andamento`, `concluído`, `en curso`,
  `listo`...).
- `category` is the card's label. Without it, the first tag.
- `weight` sorts the cards in a column.
- `sort_by_votes: false` on the roadmap's `_index.md` keeps the weight
  order when comments and votes are on.

## Changelog (`blup: changelog`)

One note per version, for example `v1.3.0.md`:

```yaml
---
title: Version 1.3.0
date: 2026-10-28
tags: [New, Fix]
type: new
---
- Faster search.
```

- `date` sorts the versions, newest first.
- `tags` become colored labels.
- `type` (`new`, `improvement`, `fix`, `security`, `breaking`) sets the
  entry's color on the changelog map. Without it, the tags are read.

## Help center (`blup: help`)

- Each subfolder is a collection. Its `_index.md` gives `title`,
  `description`, `weight` and `icon`.
- `icon` is any Lucide icon name (lucide.dev/icons), like `rocket`. An
  unknown name falls back to `circle-help`.
- Each note in a collection is an article, ordered by `weight`.
- On the help `_index.md`, `title` is the big heading and `linkTitle` the
  short menu name.

## Blog (`blup: blog`)

Each note is a post with cover, tags and authors:

```yaml
---
title: How we built it
date: 2026-10-25
description: A look inside.
tags: [Behind the scenes]
authors: [Ana]
image: "[[cover.jpg]]"
---
```

- `image` is the cover (`cover` also works). Without it, the first image
  in the text is used.
- `featured: true` puts a post at the top. Otherwise the newest is.
- `authors` empty: the site's author.

## Feedback (`blup: feedback`)

The Studio creates this section. Real requests are GitHub issues, not
notes. Its `_index.md` has `examples`, a list of sample requests (`title`,
`kind`, `status`, `votes`, `days`, `body`, `replies`). Edit only if asked.

## The home note and site settings

The `_index.md` at the top of the site folder holds the settings of the
whole site. Change it only when the person asks.

- `title`, `description` and the note's text: the home page heading.
- `site_name`, `site_description`, `logo`, `favicon` (`"[[logo.png]]"`).
- `header_name: false` hides the name next to the logo.
- `button_text` and `button_link`: a header button (both needed).
- `social`: list of profile URLs.
- `menu`: one line per item: `Docs` (a folder or note by name),
  `Guides | Docs` (name, then target), `Shop | https://example.com`,
  `Home`, and `Resources |` followed by `+ ` lines for a submenu.

## Authors

In the folder whose `_index.md` has `blup: authors`. One note per author,
named like the name in `authors:`, with `photo`, `role`, `social` and the
bio as the text.

## Attachments

Images and files go in the attachments folder at the top of the site
folder: `Attachments` (en), `Anexos` (pt) or `Adjuntos` (es), by site
language. Supported: png, jpg, jpeg, webp, avif, gif, svg, bmp, tif, tiff,
pdf, mp3, m4a, ogg, wav, flac, mp4, webm, mov, zip.

```md
![Settings screen](../Attachments/settings.png)
![Small|300](../Attachments/settings.png)
![Demo](https://www.youtube.com/watch?v=VIDEOID)
```

In properties (`image`, `logo`, `favicon`, `photo`), write `"[[name.png]]"`.

## Links

Use Markdown links with a path relative to the note, spaces as `%20`:

```md
[Install](../Docs/Getting%20started/Install.md)
[Limits](Plans.md#limits)
```

- The site does not turn `[[wikilinks]]` or `![[embeds]]` in the text
  into links. Do not write them in note text.
- A link to a draft, unpublished or missing note becomes plain text.
- Callouts (`> [!tip] Title`), math (`$x^2$`), `==highlight==`,
  footnotes, tasks and tables work.

## Password protected pages

`password: my-secret` sets the page's own password; `password: yes`
uses the site's general password. The text is encrypted at build time.
Never add, change or remove `password`, and never copy protected text
elsewhere, unless the person asks.

## Languages and translations

When the person turned on other languages in the Studio, a translation
lives in a folder named after the language code next to the note
(`Docs/en/`, `Docs/pt/`, `Docs/es/`) and shares `translationKey` with the
original. Do not create these folders yourself: until the language is on,
such a folder is a normal folder. Do not edit `translationKey`,
`translation_hash` or `ai_translated`.

## Never touch

- `.obsidian/`, `.blup/` and any other dot folder.
- `blup.toml`, `hugo.toml`, `hugo.preview.toml`, `themes/`, if present.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `INSTRUCTIONS.md`.
- Content between `%%` and `%%`.
- `password` values and protected notes.
- Keys the Studio writes: `audio`, `audio_hash`, `audio_seconds`,
  `translationKey`, `translation_hash`, `ai_translated`, `created`,
  `updated`.
