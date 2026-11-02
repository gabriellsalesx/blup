---
title: AI in Blup Studio
description: What AI does in the Studio and the three ways to pay for it, including for free.
weight: 1
---
Blup Studio uses AI in four places. All of them are optional: Blup works fully without AI.

| Feature | What it does | Where to read |
| --- | --- | --- |
| Narration | A voice reads the post, and the site shows a player at the top | [Post narration](Post%20narration.md) |
| Translation | Translates notes into English, Spanish or Portuguese, and the site gets a language switcher | [Translation](Translation.md) |
| Podcast | On blogs, the narrations become episodes with a feed for Spotify and Apple Podcasts | [Blog podcast](Blog%20podcast.md) |
| Agents in the terminal | Claude Code, Codex, OpenCode, Gemini CLI or Antigravity working on your notes | [Terminal and AI agents](../Blup%20Studio/Terminal%20and%20AI%20agents.md) |

> [!info] Blup Studio only
> Narration, translation, podcast and the terminal with agents exist only in the Studio. A site that already has narrations and a podcast keeps publishing both from the command line.

## Three ways to pay

Narration and translation can run in three ways. You choose in **Settings > AI and voice**.

### 1. On your computer, for free

- **Voice on the computer (Kokoro)**: free and offline. The Studio downloads the engine, the model and the voices once, checks each file and gets everything ready. It is not available on every system yet: the Studio tells you when it is not.
- **Translation on the computer (Firefox Translations)**: the same translator as Firefox, by Mozilla. Free, offline, and your notes never leave the computer. It downloads about 100 MB once.

### 2. With your Blup Cloud plan

The paid plans (Pro, Max and Ultra) already include AI use for narration and translation, with no key from any service. Use is measured per month:

- In **Settings > Blup Cloud**, the **AI this month** bar shows how much was used and when it renews.
- Each week uses at most half of the month, so it does not all run out at once.
- Max has twice the use of Pro, and Ultra five times.
- Before narrating, the Studio shows how much of the month that text will use.

See [Account and plans](../Blup%20Cloud/Account%20and%20plans.md).

### 3. With your own OpenRouter key

OpenRouter is a service that gives access to many AI models, paid by use.

1. In **Settings > AI and voice > AI**, click **Make one on OpenRouter**.
2. In your OpenRouter account, click **Create Key**, give it a name (for example, Blup Studio) and copy the key.
3. Paste the key in the Studio. It checks that the key works and keeps it only on your computer.

Each use spends credits of your OpenRouter account: a narration costs about US$ 0.15 per post, and translation a few cents per note.

## Which one to pick?

| If you want to | Use |
| --- | --- |
| Spend nothing and keep the text on your computer | The computer |
| Not think about keys or credits | The Blup Cloud plan |
| Choose among many models and voices | OpenRouter |

> [!tip] The cloud sounds a little more natural
> Translation in the cloud usually reads a bit more naturally; on the computer it is free and private. Under **Who translates**, pick **On the computer** or **AI in the cloud**.
