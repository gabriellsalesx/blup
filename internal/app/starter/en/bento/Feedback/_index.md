---
title: Feedback
linkTitle: Feedback
blup: feedback
weight: 6
description: Tell us what is missing, what broke and what could be better. Vote on other people's requests.
examples:
  - title: "Automatic dark mode"
    kind: feature
    status: planned
    votes: 42
    days: 2
    body: "Follow the system theme instead of switching by hand every night."
    replies:
      - author: marina
        days: 1
        text: "That would be great, a bright screen at night is tiring."
      - author: rafael
        days: 1
        text: "One more vote. Could it follow the system schedule too?"
  - title: "Search misses accented words"
    kind: bug
    status: progress
    votes: 17
    days: 9
    body: "Searching for \"café\" finds nothing, but \"cafe\" does."
    replies:
      - author: julia
        days: 6
        text: "Same here, with \"naïve\" and \"résumé\"."
      - author: team
        days: 2
        text: "We found the cause and the fix ships in the next version."
  - title: "Shortcut to create a new note"
    kind: improvement
    status: new
    votes: 11
    days: 15
    body: "A keyboard shortcut would help a lot when writing all day."
    replies:
      - author: pedro
        days: 10
        text: "Ctrl+N would feel the most natural."
  - title: "Export to PDF"
    kind: feature
    status: done
    votes: 29
    days: 22
    body: "Download a whole page as a PDF to send by email."
    replies:
      - author: carla
        days: 20
        text: "Used it today to send a report, it looks perfect. Thanks!"
  - title: "Windows XP version"
    kind: other
    status: declined
    votes: 3
    days: 40
    body: "I still use an old computer at the office."
    replies:
      - author: team
        days: 38
        text: "Windows XP no longer gets security updates, so we won't support it."
---
%%
This folder is the site's feedback board: visitors open requests (bug, feature, improvement or other), vote and comment, signing in with GitHub. The requests live in the issues of the site's comments repository, not here. You change each request's status (planned, in progress, done, declined) in Blup Studio, on the Feedback screen.

The requests in the examples list are examples, like the other sections' notes: they show on the board until the first real request arrives. Delete or change the list as you like.
%%
