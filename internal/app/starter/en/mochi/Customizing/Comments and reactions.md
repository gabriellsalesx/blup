---
title: Comments and reactions
date: {{date-9}}
description: Readers react to posts and comment on them without leaving the blog. Everything lives on your GitHub.
tags: [Customizing, CLI]
---
Visitors can react to posts (👍 ❤️ 🎉 🚀 and others) and leave comments. No server and no subscription: everything lives in issues of a public repository on your GitHub.

## Turn on

Run `blup customize` and choose **Comments and reactions on posts**. The new site wizard also asks at the end.

1. Blup creates the public repository named after the site plus `-feedback`. A blog called "My Blog" gets `my-blog-feedback`.
2. It opens the page of the **Blup Comments** app on GitHub. Choose **Only select repositories**, pick the `-feedback` repository and click **Install**. This app is what lets people react and comment without leaving the blog.
3. Run `blup publish`. On every publish Blup opens an issue for each new post and keeps titles up to date.

> [!info] Without the app
> If you skip the install, everything still works, but reacting and commenting take people to the issue on GitHub. Install it later the same way, from `blup customize`.

## How it shows

At the end of each post come the reactions, the comments and a box to comment, which takes Markdown. Click a reaction to add yours, or the smiley button to pick another one; clicking again takes it back.

To answer someone, click **Reply** on their comment: the box comes with a quote and a mention of that person, and GitHub lets them know (a notification and, if they want, an email). As the repository owner, you hear about every new comment.

## Who can take part

Reading is open to everyone. To react or comment, people click **Sign in with GitHub** and authorize the app once. They stay on the same post; the connection lives only in their browser, and **Sign out** disconnects.

Everyone can delete their own comments with the **Delete** button. As the repository owner, you see that button on every comment when you sign in on the blog.

Asking for a GitHub account keeps spam away. For more moderation, use GitHub's own tools: block a user or lock the conversation.

## Turn off

Run `blup customize`, choose **Comments and reactions on posts** and answer **No**. On the next publish the blocks leave the blog. The repository and its issues stay on your GitHub.
