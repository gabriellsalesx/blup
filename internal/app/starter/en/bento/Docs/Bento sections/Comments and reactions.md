---
title: Comments and reactions
description: Votes on the roadmap, reactions on the changelog and comments on posts, without leaving the site.
weight: 7
---
Visitors can vote on **roadmap** items, react to **changelog** entries and comment on **blog** posts. No server and no subscription: everything lives in issues of a public repository on your GitHub.

## Turn on

Run `blup customize` and choose **Comments and reactions**. The new site wizard also asks at the end.

1. Blup creates the public repository named after the site plus `-feedback`. A site called "My Product" gets `my-product-feedback`.
2. It opens the page of the **Blup Comments** app on GitHub. Choose **Only select repositories**, pick the `-feedback` repository and click **Install**. This app is what lets people vote and comment without leaving the site.
3. Run `blup publish`. On every publish Blup opens an issue for each new page and keeps titles up to date.

> [!info] Without the app
> If you skip the install, everything still works, but voting, reacting and commenting take people to the issue on GitHub. Install it later the same way, from `blup customize`.

## How it shows

- **Roadmap**: each card shows its votes and comments. On the item page, the **Vote** button adds the visitor's vote; clicking again takes it back.
- **Changelog**: reactions show under each entry. Click a reaction to add yours, or the smiley button to pick another one.
- **Blog and items**: the page shows reactions, comments and a box to comment, which takes Markdown.

## Who can take part

Reading is open to everyone. To vote, react or comment, people click **Sign in with GitHub** and authorize the app once. They stay on the same page and can take part as much as they like; the connection lives only in their browser, and **Sign out** disconnects.

To answer someone, click **Reply** on their comment: the box comes with a quote and a mention of that person, and GitHub lets them know (a notification and, if they want, an email). As the repository owner, you hear about every new comment.

Everyone can delete their own comments with the **Delete** button. As the repository owner, you see that button on every comment when you sign in on the site.

Asking for a GitHub account keeps spam away. For more moderation, use GitHub's own tools: block a user or lock the conversation.

> [!tip] Numbers update on their own
> The site reads GitHub when the page opens. A new vote shows up without publishing again.

## Turn off

Run `blup customize`, choose **Comments and reactions** and answer **No**. On the next publish the blocks leave the site. The repository and its issues stay on your GitHub, for when you want to turn it back on.
