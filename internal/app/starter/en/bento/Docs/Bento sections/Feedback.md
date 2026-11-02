---
title: Feedback
description: A board where visitors ask for features, report bugs and vote. Blup Studio only.
weight: 8
---
**Feedback** is a request board: visitors suggest features, report bugs, propose improvements and vote on other people's requests. You follow everything and change the status from the Studio.

> [!info] Blup Studio only
> The Feedback section is added on the Studio's **Sections** screen, and requests are managed on the **Feedback** screen. It needs [comments](Comments%20and%20reactions.md) turned on.

## Add it

1. Turn on comments, if you have not yet. See [Comments and reactions](Comments%20and%20reactions.md).
2. On the **Sections** screen, click **Add** next to **Feedback**.
3. The Studio creates the **Feedback** folder, with a few example requests.
4. Publish the site.

## The board on the site

| Control | What it does |
| --- | --- |
| Categories | All Feedback, Feature Requests, Bug Reports, Improvements and Other, with the count of each |
| Status | All, the active ones (New, Planned, In progress) or a single status |
| Most voted and Recent | The order of the list |
| Search | Looks in the title and the text |
| New request | Type, title and description. The person signs in with GitHub to send it |
| Load more requests | Shows ten more |

Click a request to see the text, the votes and the comments.

## The example requests

The **_index** note of the Feedback folder keeps a few example requests, in the `examples` property. They show how the board looks before the first real request arrives. When the first real request comes in, the examples disappear from the board.

You can edit or delete the examples as you like. On a site in several languages, their texts are translated too.

## Reply and change the status

Click **Feedback** in the Studio's side bar.

- The screen lists the real requests, from most voted to least, with type, author, comments, votes and language.
- Filter by category.
- Click the title to open it on GitHub, where you reply.
- Change the **Status**: New, Planned, In progress, Done or Declined. Done and Declined close the request; the others reopen it.
- **Refresh** reads it all again.

The site shows the new status within a minute, without publishing again.

> [!tip] Where the requests live
> Each request is an issue in the comments repository, on your GitHub. There is no note per request.
