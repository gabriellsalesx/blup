# Instructions for AI agents in this vault

This folder is a Blup site with the Mochi theme: a personal blog with
cards and covers. Blup turns these Markdown notes into a website with
Hugo. The person writes and publishes in Blup Studio.

Write notes in {{LANGUAGE}}.

## Golden rules

- Only create or change notes when the person asks.
- Do not delete, rename or move folders or notes unless asked.
- Never run `hugo`, `blup` or any build, preview or publish command. The
  Studio preview updates by itself when a note is saved. To publish, the
  person uses Publish in Blup Studio.
- Never use em dashes or en dashes in the text you write.

## How the folder becomes the site

- Every `.md` note becomes a page.
- In Mochi, every page is a post. The home page lists all visible notes,
  newest first, with one featured post on top.
- Every folder becomes a section, with its own page that lists its posts.
- A folder's `_index.md` note is the section page: its title, its
  description and the text shown at the top of the list.
- With no `menu` set (see below), the top menu shows Home plus the
  top-level folders, ordered by their `_index.md` `weight`.
- Tags get pages under `/tags/`, authors under `/authors/`.
- Search covers every page except hidden ones.

Left out of the site:

- Files and folders whose name starts with a dot (`.blup`, `.trash`...).
- The authors folder (the one whose `_index.md` has `blup: authors`).
  It holds profiles, not pages.
- The note templates folder, when the notes folder sets one.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md` and `INSTRUCTIONS.md`.
- Notes with `publish: false` or `private: true`, and their translations.
- Drafts (`draft: true`): they show only in the Studio preview.
- Text between `%%` and `%%` (comments).
- Any file that is not a note or a supported attachment.

## Create a post

Create a `.md` note anywhere in the site folder. A post needs nothing
else, but good posts have properties at the top:

```yaml
---
title: My first trip to Lisbon
date: 2026-10-29
description: One sentence that sums up the post.
tags: [Travel, Portugal]
authors: [Ana]
image: "[[lisbon.jpg]]"
---
Text of the post.
```

- Without `title`, the page uses the note's first `# Heading`, else the
  file name. A number prefix like `01 - ` is removed from the file name.
- The first `# Heading` of the text is not printed twice: the layout
  already shows the title.
- Without `description`, the start of the text is used (about 160
  characters).

## Create a page that is not a post

Put the note at the top of the site folder and mark it `hidden: true`.
It stays out of the post lists but can be linked and put in the menu.
The starter's "About" note works this way.

## Create a section

Create a folder and an `_index.md` inside it:

```yaml
---
title: Travel
description: Notes from the road.
weight: 2
---
Short introduction shown above the list of posts.
```

## Front matter keys Mochi reads

| Key | What it does |
| --- | --- |
| `title` | Page title. |
| `linkTitle` | Short name in menus. |
| `description` | Summary in cards, Google and link previews. |
| `date` | Post date (`2026-10-29`). Sorts posts, newest first. |
| `tags` | List of topics. Each tag gets a page. Related posts use them. |
| `authors` | List of author names. Empty: the site's author. |
| `image` | Cover image, like `"[[photo.jpg]]"`. `cover` also works. Without both, the first image in the text is used. |
| `featured` | `true` puts this post at the top of the home page. Without it, the newest post is featured. |
| `weight` | Order of sections in the automatic menu (lower first). |
| `draft` | `true`: only in the preview, never published. |
| `hidden` | `true`: published, but out of lists, search and Google (`noindex`). Reachable by its link. |
| `noindex` | `true`: asks Google not to index the page. |
| `publish` | `false`: never published at all. |
| `private` | `true`: never published at all. |
| `password` | Asks for a password to read. See below. |

If no date is set, Blup tries `publishDate`, then `created`, then a date
at the start of the file name, then the file's change date. `lastmod`,
`updated` or `modified` set the update date.

Keys the Studio writes for you. Do not set or edit them by hand:
`audio`, `audio_hash`, `audio_seconds` (post narration),
`translationKey`, `translation_hash`, `ai_translated` (translations),
`created`, `updated`.

## The home note and site settings

The `_index.md` at the top of the site folder holds the settings of the
whole site. Change it only when the person asks.

- `title`: the big heading of the home page. The note's text is the
  introduction under it.
- `site_name`, `site_description`: name and one sentence about the site.
- `logo`, `favicon`: images, like `"[[logo.png]]"`.
- `header_name`: `false` hides the site name next to the logo.
- Where the logo leads (the home page, or an address like the company's
  main site) is set in the Studio (Appearance), not in this note.
- `button_text` and `button_link`: a button in the header (both needed).
- `social`: list of profile URLs. The icon comes from the address. An
  e-mail becomes a `mailto:` link.
- `menu`: one line per item, in order:
  - `Travel`: a folder or note of the site, by name.
  - `Trips | Travel`: shown name, then the folder or note.
  - `Shop | https://example.com`: an external link (also `mailto:` or
    `/path`).
  - `Home`: the home page.
  - `More |` then lines starting with `+ ` below it: a submenu.
  - Empty: Home plus the top-level folders.

## Authors

Authors live in the folder whose `_index.md` has `blup: authors`. A new
one is named `Writers` (en), `Autores` (pt) or `Escritores` (es), but find
it by that property, not by name. One note per author, named exactly like
the name used in `authors:`:

```yaml
---
photo: "[[ana.jpg]]"
role: Editor
social: [https://github.com/ana]
---
The note's text is the bio.
```

## Attachments

Images and files go in the attachments folder at the top of the site
folder. Its name depends on the site language: `Attachments` (en),
`Anexos` (pt), `Adjuntos` (es). Use the existing folder.

Supported files: png, jpg, jpeg, webp, avif, gif, svg, bmp, tif, tiff,
pdf, mp3, m4a, ogg, wav, flac, mp4, webm, mov, zip. Only the ones a
published page uses are published.

In the text, embed with a relative Markdown path:

```md
![A tram on a hill](../Attachments/tram.jpg)
![Small photo|300](../Attachments/tram.jpg)
![Episode](../Attachments/talk.mp3)
![Ana's talk](https://www.youtube.com/watch?v=VIDEOID)
```

The text in brackets is the image description (alt text). `|300` sets
the width. Audio and video files get a player, a PDF gets a link,
YouTube links get the video player.

In properties (`image`, `logo`, `favicon`, `photo`), use `"[[name.jpg]]"`:
the file is found by name.

## Links

Use standard Markdown links with a path relative to the note:

```md
[Packing list](../Guides/Packing%20list.md)
[The second day](Lisbon.md#the-second-day)
[Back to the top](#intro)
```

- The site does not turn `[[wikilinks]]` or `![[embeds]]` in the text
  into links. Do not write them in note text. Use Markdown links.
- A link to a draft, unpublished or missing note becomes plain text.
- Encode spaces as `%20` (and `(` `)` as `%28` `%29`).

## Writing that works

- Callouts: `> [!tip] Title`. Types include note, info, todo,
  success, question, warning, failure, danger, bug, example, quote,
  abstract. `> [!example]- Title` starts folded, `+` starts open.
- Math: `$x^2$` inline and `$$ ... $$` blocks.
- `==highlight==`, footnotes, task lists, tables and code blocks.
- A `{.text-center}` line under a paragraph centers it.
- `##` and `###` headings make the post's table of contents.
- A single line break is kept as a line break (unless the notes folder
  is set to strict line breaks).

## Password protected posts

`password: my-secret` gives a post its own password. `password: yes`
(also `true`, `sim`, `si`, `sí`) uses the site's general password, set by
the person in the Studio. The text is encrypted when the site is built.
The title, description and cover stay visible.

- Never add, change or remove `password` unless asked.
- Never copy text from a protected note into another note, a commit or
  your answer to a third party.
- Protected posts are not sent to cloud AI translation and get no
  narration.

## Languages and translations

The site's main language is the one you write in. When the person turned
on other languages in the Studio, a note's translation lives in a folder
named after the language code next to it: `Travel/en/`, `Travel/pt/` or
`Travel/es/`. The original and its translations share `translationKey`.
A section's `_index.md` keeps its name in the language folder. Do not
create language folders yourself: a folder named `en`, `pt` or `es` is a
normal section until the person turns that language on. Translations are
made in the Studio's Languages screen.

## Podcast

A folder whose `_index.md` has `blup: podcast` is the blog's podcast
page. The Studio creates and manages it. Its properties are
`podcast_title`, `podcast_author`, `podcast_email`, `podcast_category`,
`podcast_explicit`, `description` and `image`. Change them only if asked.

## Other special notes

- `blup: privacy` marks the privacy policy page. Edit it only when asked.
- `.canvas`, `.draw` and `.table` files are not pages of the site.

## Never touch

- `.blup/` and any other folder that starts with a dot.
- `blup.toml`, `hugo.toml`, `hugo.preview.toml` and the `themes/` folder,
  if you see them.
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `INSTRUCTIONS.md`.
- The content between `%%` and `%%`: leave it as it is.
- `password` values and protected notes (see above).
- The keys the Studio writes (narration and translation keys above).
