---
title: Comments and reactions
date: {{date-9}}
description: Readers react to and comment on posts without leaving the blog. Everything lives on your GitHub.
tags: [Customizing, Studio]
---
People visiting the blog can react to posts (👍 ❤️ 🎉 🚀 and others) and leave comments. No server and no monthly fee: everything lives in issues of a public repository on your GitHub.

## Turn it on in the Studio

In the side menu, click **Comments** and turn on **Show comments and reactions**. The first time, there are two steps:

1. **Connect GitHub**: click **Open the form**, create the key (the permissions come already ticked), paste it into the Studio and click **Connect**.
2. **Create the comments space**: click **Create and turn on**. Blup creates a public repository on your GitHub just for the comments, named after the site plus `-feedback`.

After that, there is one more detail so people can comment **without leaving the blog**: installing the Blup app on the repository. Click **Install the app on GitHub** and, on the page that opens:

- choose **Only select repositories**;
- tick the `-feedback` repository;
- click **Install** and go back to the Studio. Click **I installed it**.

Finally, publish again so the comments show up on the site.

> [!info] With Obsidian
> Run `blup customize` and choose **Comments and reactions on posts**. Blup does the same steps in the terminal.

> [!info] Without the app
> If you skip the install, everything still works, but reacting and commenting take the person to the issue on GitHub. You can install it later, the same way.

## How it shows up

At the end of each post are the reactions, the comments and a box to comment in, which accepts Markdown. Click a reaction to add yours, or the smiley button to pick another one; clicking again undoes it.

To answer someone, click **Reply** on the comment: the box comes with a quoted bit and a mention of the person, and GitHub lets them know. As the repository owner, you are notified of every new comment.

On a blog in several languages, all versions of a post share the same conversation.

## Who can take part

Reading is open to everyone. To react or comment, the person clicks **Sign in with GitHub** and authorizes the app once. They stay on the same post; the connection lives only in their browser, and **Sign out** disconnects it.

Each person can delete their own comments with the **Delete** button. You, as the repository owner, see that button on every comment when you sign in to the blog.

Requiring a GitHub account keeps spam away. To moderate more, use GitHub's own tools: block a user or lock the conversation.

## Turn it off

On the **Comments** screen, turn off **Show comments and reactions** and publish again: the blocks leave the blog. The repository and the issues stay on your GitHub.
