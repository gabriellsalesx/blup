---
title: Multilingual sites
description: The same site in Portuguese, English and Spanish, with a language switcher at the top.
weight: 5
---
A Blup site is written in one **main language** (Portuguese, English or Spanish) and can also be published in the other two. Each note gets a translation, and the site shows a **language switcher** at the top.

> [!info] Blup Studio only
> Translation happens in the Studio, by AI in the cloud or on your computer. See who translates and what it costs in [Translation](../AI%20features/Translation.md).

## Translate one note

1. Open the note in the Studio.
2. In the **Languages** block, in the properties, click **Translate** next to the language.
3. The Studio shows the progress ("Translating, part 1 of 3…") and tells you when the translation is ready.
4. **Open** takes you to the translation, so you can review it.

The AI translates the title, the description and the text. Code, links, images and the callout types do not change.

## Translate the whole site

Click **Languages** in the side bar. For each language, the screen shows how many notes are **translated**, **out of date** and **not translated**.

- **Translate what is missing** translates all notes without a translation in one go.
- **Update the translations** redoes the ones that fell behind.

## Where each translation lives

The translation goes in a folder named with the language code (`en`, `es` or `pt`), **next to the original note**, with a translated name:

```text
Blog/
├── How to publish.md       original, in English
├── es/
│   └── Cómo publicar.md    Spanish translation
└── pt/
    └── Como publicar.md    Portuguese translation
```

The original and its translations are linked by the `translationKey` property, which the Studio adds by itself. On the site, the translated page is the same page in the other language: the switcher takes you from one to the other. A link to a note with no translation yet goes to the original.

## What else is translated

- In the home note: the site description, the button text and the menu names.
- In any note: the short title (`linkTitle`), the category and the tags.
- Each section's _index note gives the section's translated name.
- The author profiles: the role and the bio.
- On the Feedback board: the texts of the example requests.

## Out of date translations

When you change the original, the translation shows as **Out of date: the original changed**. Click **Translate again**.

## The "Translated with AI" badge

A translation made by AI shows a **Translated with AI** badge next to the title, which opens a bubble with the original language and a link to it. Reviewed it? Click **Mark as reviewed** and the badge goes away.

## Comments and votes

All versions of a page share the same conversation: votes and reactions add up readers from every language. A comment shows which language it was written in, like "commented in Português".

> [!tip] Translating by hand
> You can translate by hand: create the `es` folder next to the note and write the translation there, with the same file name as the original (or the same `translationKey`). The published languages are listed under `translations` in blup.toml, which the Studio fills in when it translates.
