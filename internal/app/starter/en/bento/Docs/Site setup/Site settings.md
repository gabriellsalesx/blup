---
title: Site settings
description: Name, description, author, logo and favicon, on the Site screen or in the _index note.
weight: 1
---
The site settings are saved as **properties of the _index note** in the site folder. There are three ways to change them, and all of them save to the same place.

## In Blup Studio: the Site screen

Click **Site** in the side bar. The screen shows how the site presents itself, in blocks:

1. **Name and description**
   - **Site name**: shows at the top, in the browser tab and on Google.
   - **Description**: one sentence. Shows at the top, on Google and when the site is shared.
   - **Author**: who signs the posts that do not name an author.
2. **Logo and favicon**
   - **Logo**: at the top of the site. Without a logo, the name shows.
   - **Show the name next to the logo**: when off, the top of the site shows the logo alone.
   - **Favicon**: the icon of the tab and of the app. Square, ideally 512 pixels or SVG.
3. **Top menu** and **Top button**. See [Menu, submenus and button](Menu,%20submenus%20and%20button.md).
4. **Social links**. See [Social links](Social%20links.md).

Each change shows up in the preview next to it right away, without reloading.

![Blup setting things up](../../Attachments/blup-configurar.jpg)

## In Obsidian: the _index note

| Property | What it does |
| --- | --- |
| `site_name` | Site name: browser tab, footer and Google. |
| `site_description` | One sentence about the site, for Google and social networks. |
| `logo` | Logo image at the top, like `"[[logo.svg]]"`. |
| `header_name` | `false` shows only the logo at the top, without the name. |
| `favicon` | Browser tab icon, like `"[[favicon.png]]"`. |
| `menu` | Items of the top menu. |
| `button_text` and `button_link` | The button at the top right. |
| `social` | Links to your profiles, with icons in the footer. |

```yaml
---
title: Acme
description: The simple way to do X.
site_name: Acme
site_description: The simple way to do X.
logo: "[[acme-logo.svg]]"
favicon: "[[acme-icone.png]]"
social:
  - https://github.com/acme
  - mailto:hi@acme.com
---
```

## From the command line

`blup customize` asks the same questions (name, description, author, logo, favicon, menu, social links) and saves them to the same properties.

## Logo and favicon

- In **SVG**, they stay sharp on any screen.
- In **PNG**, Blup makes the sizes that browsers and phones ask for.
- Without a logo, the top shows the name. Without a favicon, the site uses the mascot.

## The other site screens in the Studio

| Screen | What it sets up |
| --- | --- |
| Sections | The parts of Bento. See [Turn sections on and off](../Bento%20sections/Turn%20sections%20on%20and%20off.md) |
| Authors | The profiles. See [Authors](Authors.md) |
| Languages | The site in other languages. See [Multilingual sites](Multilingual%20sites.md) |
| Comments | Votes, reactions and comments. See [Comments and reactions](../Bento%20sections/Comments%20and%20reactions.md) |
| Cookies and statistics | Tools and banner. See [Cookies and statistics](Cookies%20and%20statistics.md) |
| Podcast | Blogs only. See [Blog podcast](../AI%20features/Blog%20podcast.md) |
| Publish | Where and how the site goes live. See [Publish from the Studio](../Publishing/Publish%20from%20the%20Studio.md) |

> [!note] What about blup.toml?
> In the site folder on your computer, **blup.toml** keeps what is not content: where to publish, the custom domain, the languages, the site password and the sync repository. The Studio and `blup customize` change all of it without you opening the file.
