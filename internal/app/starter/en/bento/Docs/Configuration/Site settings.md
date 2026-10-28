---
title: Site settings
description: Name, description, logo, favicon and social links live in the _index note of the site folder.
weight: 1
---
The site settings are **properties of the _index note** of the site folder. Change them in Obsidian and the site changes. In the terminal, `blup customize` asks the same questions and saves to the same properties.

| Property | What it does |
| --- | --- |
| `site_name` | Site name: browser tab, footer and Google. |
| `site_description` | One sentence about the site, for Google and social networks. |
| `logo` | Logo image at the top, like `"[[logo.svg]]"`. |
| `favicon` | Browser tab icon, like `"[[favicon.png]]"`. |
| `menu` | Top menu items. See [Menu and button](Menu%20and%20button.md). |
| `button_text` and `button_link` | The button at the top right. |
| `social` | Links to your profiles, with icons in the footer. |

## Example

```yaml
---
title: Acme
description: The simple way to do X.
site_name: Acme
site_description: The simple way to do X.
logo: "[[acme-logo.svg]]"
favicon: "[[acme-icon.png]]"
social:
  - https://github.com/acme
  - https://www.linkedin.com/company/acme
  - mailto:hi@acme.com
---
```

## Social links

Blup recognizes the network from the address and uses the right icon: Instagram, X, LinkedIn, YouTube, GitHub, Facebook, WhatsApp (`wa.me`) and e-mail (`mailto:`). Other addresses get a link icon.

## Logo and favicon

- As **SVG**, they stay sharp on any screen.
- As **PNG**, Blup creates the sizes browsers and phones ask for.
- Without a logo, the site shows the mascot. Without a favicon, too.

> [!note] What about blup.toml?
> In the site folder on your computer, **blup.toml** keeps what isn't content: where to publish, the general password and the sync repository. The name and description you gave in the wizard are stored there too, as a fallback, but the _index note is what counts.
