---
title: Comments and reactions
description: Votes on the roadmap, reactions on the changelog and comments on posts, without leaving the site.
weight: 7
---
Visitors can vote on **roadmap** items, react to **changelog** versions and comment on **blog** posts. No server and no monthly fee: everything lives in issues of a public repository on your GitHub.

## Turn it on in Blup Studio

Click **Comments** in the side bar and follow **Turn on for the first time**:

1. **Connect GitHub**: click **Open the form** (the permissions come already ticked), create the key, paste it and click **Connect**.
2. **Create the comments space**: click **Create and turn on**. Blup creates a public repository on your GitHub just for the comments.
3. **Comment without leaving the site**: click **Install the app on GitHub**. On the page that opens, choose **Only select repositories**, tick the comments repository and click **Install**. Come back and click **I installed it**.
4. Publish again for the comments to show on the site.

The **Show comments and reactions** switch turns them on and off later.

## Turn it on from the command line

Run `blup customize` and pick **Comments and reactions (votes on the roadmap)** (the new site wizard also asks at the end).

1. Blup creates the public repository with the site name plus `-feedback`.
2. It opens the page of the **Blup Comments** app on GitHub. Choose **Only select repositories**, tick the repository and click **Install**.
3. Run `blup publish`. Each time you publish, Blup opens an issue for each new page and keeps the titles up to date.

> [!info] Without the app
> If you skip the install, everything still works, but voting, reacting and commenting take the person to the issue on GitHub.

## How it looks

- **Roadmap**: each card shows the votes and the comments. On the item's page, the **Vote** button adds the person's vote; clicking again takes it back.
- **Changelog**: the reactions show under each version. Click a reaction to add yours, or the smiley button to pick another one.
- **Blog and items**: the page shows the reactions, the comments and a box to comment, which accepts Markdown.

## Who can take part

Anyone can read. To vote, react or comment, the person clicks **Sign in with GitHub** and allows the app once. The connection stays only in their browser, and **Sign out** disconnects.

To answer someone, click **Reply** on the comment: the box comes with a quoted bit and a mention of the person, and GitHub lets them know. As the repository owner, you are told about every new comment.

Each person can delete their own comments with the **Delete** button. You, as the repository owner, see that button on every comment when you sign in on the site.

Asking for a GitHub account keeps spam away. To moderate more, use GitHub's own tools: block a user or lock the conversation.

> [!tip] The numbers update by themselves
> The site reads GitHub when the page opens. A new vote shows up without publishing again.

## In several languages

All translated versions of a page share the same conversation: votes and reactions add up readers from every language.

## Turn it off

In the Studio, turn off **Show comments and reactions**. On the command line, run `blup customize`, pick **Comments and reactions (votes on the roadmap)** and answer **No**. On the next publish the blocks leave the site. The repository and the issues stay on your GitHub.
