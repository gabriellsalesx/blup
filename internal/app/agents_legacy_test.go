package app

// The AGENTS.md an older Blup wrote (English), word for word.
const legacyAgentsEN = `# This is a Blup site

This folder is a site. Blup turns the notes into pages with Hugo, and Blup
Studio shows a live preview. Always write in English.

## How notes become the site

- Each .md note becomes a page; each folder becomes a section.
- A folder's _index.md note is the section's page.
- The top _index.md holds the site's settings (site_name, site_description,
  logo, favicon, menu, social, button_text, button_link). Only change it
  when asked.

## Properties (at the top of the note, in YAML)

- title, description: title and summary.
- date: a date like 2026-10-29.
- tags: [One, Another]; authors: [Name].
- image: "[[cover.jpg]]" (cover).
- draft: true (preview only); hidden: true (out of lists and Google).
- password: yes (the site's general password) or its own password.
- weight: order in menus (lower first).

## Bento theme sections

Each section's _index.md has the property blup: docs, roadmap, changelog,
help or blog.
- Roadmap: one note per item, with status (planned, in progress or done)
  and category.
- Changelog: one note per version, with date and tags.

## Writing

- Links between notes in standard Markdown, with a relative path:
  [text](../Folder/Note.md). Links to headings: Note.md#heading.
- Images live in the attachments folder (Attachments); use
  ![description](../Attachments/photo.jpg).
- YouTube video: ![title](https://www.youtube.com/watch?v=...).
- Obsidian callouts (> [!tip] Title), math ($x^2$), footnotes and tasks
  work. %%Comments%% never reach the site.
- No em or en dashes (— or –) in the text.

## Don't

- Don't delete or rename folders unless asked.
- Don't touch the .obsidian folder or these files (AGENTS.md, CLAUDE.md,
  GEMINI.md).
- Don't run Hugo: the Studio's preview updates by itself. To publish, the
  person uses the Studio's Publish button.
`
