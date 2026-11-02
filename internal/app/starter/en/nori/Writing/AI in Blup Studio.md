---
title: AI in Blup Studio
date: {{date-12}}
description: Post narration, note translation and a terminal with AI agents, all inside the Studio.
tags: [AI, Studio]
image: "[[blup-terminal.jpg]]"
---
Blup Studio brings AI close to what you write, always as a helper and never in your place. There are three features, all only in the Studio:

- **Narration**: a voice reads the post, and the site gets a player. See [Narrate a post as audio](Narrate%20a%20post%20as%20audio.md).
- **Translation**: the post becomes pages in other languages, with a language picker on the site. See [A blog in several languages](A%20blog%20in%20several%20languages.md).
- **Terminal with AI agents**: a coding AI works on your notes, while you watch and can undo.

And the narrations even become a podcast: see [Your blog becomes a podcast](../Customizing/Your%20blog%20becomes%20a%20podcast.md).

## Where the AI comes from

You choose, in the Studio's **Settings**, under **AI and voice**:

| Option | How it works |
| --- | --- |
| **Blup Cloud plan** | From Pro up, the plan includes some AI use each month. Settings show how much has been used. |
| **Your own OpenRouter key** | Connect your key and each use spends credits from your OpenRouter account. |
| **On the computer** | A voice and a translator that run on your computer, free and offline. They download once. |

On the computer, the notes never leave it. The local voice isn't available on every system yet; when it isn't, the Studio tells you.

## The terminal with AI agents

Click **Terminal**, at the bottom of the side menu. It opens in the notes folder. Under **Open an AI**, the Studio shows the agents you already have installed: Claude Code, Codex, OpenCode, Gemini CLI and Antigravity. The Studio doesn't install these programs; each one uses its own account.

Just ask in plain English: "review the post titles", "write a post about the trip from these notes", "add tags to the recipe posts".

> [!tip] You can undo it
> Before the AI starts, the Studio keeps a copy of the notes. In **See what the AI changed**, you see each new, changed or deleted note and, if you don't like it, click **Undo all**: the notes go back to how they were, and whatever the AI created goes to the Trash.

If the AI changes the note that is open in the editor, the Studio tells you and lets you choose between **Load the new one** and **Keep mine**.

> [!info] And the command line?
> Narration, translation and the terminal with agents are in the Studio. A narration or translation made in the Studio stays on the site even when you publish from the command line.
