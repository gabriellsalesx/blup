# Instructions for AI agents in this vault

This folder is a Blup vault of notes: plain Markdown files the person
opens in Blup Studio (and, if they like, in Obsidian). It is not a
website. There is no theme, no preview and no publishing here. The
Studio gives it an editor, a calendar, a map of notes, canvases,
drawings, databases and sync.

Write notes in {{LANGUAGE}}.

## Golden rules

- Only create or change files when the person asks.
- Do not delete, rename or move notes or folders unless asked. When the
  Studio moves or renames something, it updates the links and keeps each
  database with its records; a move done by hand does not.
- Never run `hugo`, `blup` or any build, sync or publish command. Sync is
  turned on and run by the person in the Studio.
- Never use em dashes or en dashes in the text you write.

## What is in the folder

| Item | What it is |
| --- | --- |
| `Note.md` | A note: optional YAML properties at the top, then Markdown. |
| Folders | Plain folders to organize notes. Any depth. |
| Attachments folder | Images and files the notes use. |
| Calendar folder | One note per event (see Calendar). |
| Daily folder | One note per day, `2026-11-02.md`. |
| `Name.canvas` | A canvas (JSON Canvas). |
| `Name.draw` | A drawing (Blup's JSON). |
| `Name.table` + `Name/` | A database and the folder of its records. |

Some folder names depend on the language. The attachments folder follows
the vault's language. The calendar, daily and database folders follow
the Studio's language when the Studio creates them:

| Folder | en | pt | es |
| --- | --- | --- | --- |
| Attachments | `Attachments` | `Anexos` | `Adjuntos` |
| Calendar | `Calendar` | `Agenda` | `Agenda` |
| Daily notes | `Daily` | `Diário` | `Diario` |
| Databases | `Databases` | `Bancos de dados` | `Bases de datos` |

Use the folders that already exist in the vault.

The Studio's file list hides files and folders that start with a dot,
the attachments folder (it has its own screen) and the agent guides.

## Create a note

Create a `.md` file. The file name is the note's name. Avoid these
characters in names: `* " \ / < > : | ? # ^ [ ]`.

```yaml
---
created: 2026-11-02
updated: 2026-11-02
tags: [project, ideas]
---
# Title

Text.
```

- Properties are optional. The Studio writes `created` and `updated`.
- `tags` is a list. When most notes of a folder share a tag, the Studio
  gives new notes there the same tag. Do the same.

## Links

Both forms work in the editor, in "mentioned in" and in the map of notes:

```md
[Meeting notes](../Work/Meeting%20notes.md)
[[Meeting notes]]
[[Meeting notes|the meeting]]
```

- The Studio itself writes Markdown links with a path relative to the
  note, spaces as `%20` (and `(` `)` as `%28` `%29`). Prefer that form.
- `#heading` after the path links to a heading.
- Images use `![description](../Attachments/photo.png)`.

## Attachments

Pasted or dropped images go in the attachments folder at the top of the
notes folder, and the note gets a relative Markdown image link to it.
Images pasted on a canvas or a drawing go there too.

## Daily notes

The note of the day is `<Daily folder>/YYYY-MM-DD.md`. When the Studio
makes one it writes `created`, `updated` and the tag `daily` (en) or
`diario` (pt, es), and each time it is opened it adds a `## HH:MM`
heading at the end to write under.

## Calendar

Each event is a note directly inside the calendar folder at the top of
the notes folder (`Calendar` or `Agenda`; the Studio reads both). Notes
in subfolders are not read. The file name is the event's title.

```yaml
---
date: "2026-11-05"
time: "14:00"
event: exam
repeat: yearly
remind: 1d
---
What to study for the exam.
```

| Key | Values |
| --- | --- |
| `date` | Required. `YYYY-MM-DD`. Without a valid date the note is not an event. |
| `time` | `HH:MM` (24 h). None: the whole day. |
| `event` | `appointment`, `birthday`, `exam`, `interview`, `deadline`, `reminder`. Anything else counts as `reminder`. |
| `repeat` | `weekly`, `monthly`, `yearly`. None: once. |
| `remind` | `0` (at the time), `15m`, `1h`, `1d`, `1w`. None: no reminder. |

A whole day event reminds at 9:00. The note's text is free: what to
bring, questions, notes about it.

## Databases

A database is two things side by side in the same folder:

- `Clients.table`: JSON with `version`, `records` (the records' folder
  name), `tag`, `columns` and `views`.
- `Clients/`: one note per record. Each column is a property of the
  note, under the column's `key`. Every record has the database's `tag`.

Column types: `text`, `number`, `phone`, `email`, `url`, `date`,
`select`, `multi`, `check`, `link` (a note). Views are `table` or
`board` (a board groups by `groupBy`).

To add a record, create a note in the records folder with the tag and
the column keys as properties:

```yaml
---
created: 2026-11-02
updated: 2026-11-02
tags: [clients]
company: Acme
status: Hot
---
```

Never rename or move the `.table` file or its records folder by hand:
the Studio moves them together. Change the `.table` only when asked.

## Canvases (.canvas)

JSON Canvas 1.0, the format Obsidian uses. An empty one is
`{"nodes":[],"edges":[]}`.

- Node: `id`, `type` (`text`, `file`, `link`, `group`), `x`, `y`,
  `width`, `height`, optional `color` (`"1"` to `"6"` or `#hex`). Plus
  `text` (text card, Markdown), `file` (a path in the notes folder, like
  `Work/Plan.md`), `url` (link card) or `label` (group).
- Edge: `id`, `fromNode`, `toNode`, optional `fromSide`/`toSide` (`top`,
  `right`, `bottom`, `left`), `fromEnd`/`toEnd` (`none`, `arrow`),
  `color`, `label`.
- Ids must be unique. An edge to a missing node is dropped. Keys the
  Studio does not know are kept.
- Cards of other types (from Obsidian plugins) and the edges to them are
  not shown in the Studio, but are kept as they are when it saves. Do not
  remove or change them.

## Drawings (.draw)

JSON: `{"version":1,"elements":[...]}`. Each element has `id`, `type`
(`rect`, `ellipse`, `diamond`, `arrow`, `line`, `pen`, `text`, `image`),
a box `x`, `y`, `w`, `h`, `stroke` (`ink`, `red`, `orange`, `yellow`,
`green`, `blue`, `purple`, `gray` or `#hex`), `fill` (a color or
`none`), `width` and `dash` (`solid`, `dashed`, `dotted`). Lines, arrows
and pen strokes have `points`, relative to `x`, `y`. Text has `text` and
`size`; an image has `file`. Prefer editing drawings in the Studio.

## Writing that the editor shows well

Headings, lists, task lists, tables, quotes, code blocks, Obsidian
callouts (`> [!tip] Title`), `==highlight==` and `$math$`. Text between
`%%` and `%%` is a comment.

## Sync

Sync copies the notes to the person's other computers. Files and folders
whose name starts with a dot are not synced (except `.blup/site.toml`).

## Never touch

- `.obsidian/`, `.blup/` and any other folder that starts with a dot.
- `blup.toml` (it lives in Blup's own folder, not here).
- `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `INSTRUCTIONS.md`.
- The content between `%%` and `%%`: leave it as it is.
- `.table` files and records folders' names, unless asked.
- Any note you were not asked to change.
