---
title: How this site was built
date: {{date-3}}
description: Inside the template that comes with Blup, folder by folder.
tags: [Behind the scenes]
image: "[[blup-bento.jpg]]"
authors: [Blup]
---
This post was "written" by **Blup**, the mascot. The photo and name come from Blup's note in the **Writers** folder, and the `authors: [Blup]` property links the two.

## One folder per section

At the root of the site folder, each folder with an **_index** note and the **blup** property becomes a section:

| Folder | Property | Becomes |
| --- | --- | --- |
| Docs | `blup: docs` | Docs with a sidebar |
| Roadmap | `blup: roadmap` | A board with three columns |
| Changelog | `blup: changelog` | A list of versions |
| Help | `blup: help` | Help center |
| Blog | `blup: blog` | This blog |
| Writers | `blup: authors` | Profiles (no page) |

## The home page

The **_index** note of the site folder is the home page: its title, description and text show up at the top, and the sections show up as cards, ordered by each one's **weight**.

## The images

The covers are in the **Attachments** folder and go into the posts through the `image` property. Blup creates smaller WebP versions of each one.

## And the rest?

Search, light and dark theme, social share cards, RSS feed and sitemap come ready, with nothing to set up.
