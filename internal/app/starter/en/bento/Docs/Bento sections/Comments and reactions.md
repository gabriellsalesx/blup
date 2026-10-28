---
title: Comments and reactions
description: Votes on the roadmap, reactions on the changelog and comments on posts, kept on GitHub.
weight: 7
---
Visitors can vote on **roadmap** items, react to **changelog** entries and comment on **blog** posts. No server and no subscription: everything lives in issues of a public repository on your GitHub.

## Turn on

Run `blup customize` and choose **Comments and reactions**. The new site wizard also asks at the end.

Blup creates the public repository named after the site plus `-feedback`. A site called "My Product" gets `my-product-feedback`.

Then run `blup publish`. On every publish Blup opens an issue for each new page and keeps titles up to date.

## How it shows

- **Roadmap**: each card shows its votes and comments. On the item page, the **Vote** button leads to the issue.
- **Changelog**: reactions show under each entry.
- **Blog and items**: the page shows reactions and comments, with a **Comment on GitHub** button.

Reading is open to everyone. Voting, reacting and commenting happen on GitHub, with a GitHub account. That keeps spam away, and you moderate with GitHub's own tools: delete a comment, block a user or lock the conversation.

> [!tip] Numbers update on their own
> The site reads GitHub when the page opens. A new vote shows up without publishing again.

## Turn off

Run `blup customize`, choose **Comments and reactions** and answer **No**. On the next publish the blocks leave the site. The repository and its issues stay on your GitHub, for when you want to turn it back on.
