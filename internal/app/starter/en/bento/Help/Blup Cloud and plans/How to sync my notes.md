---
title: How do I sync my notes between computers?
description: With Blup Cloud or a private GitHub repository.
weight: 3
---
1. In the Studio, click **Sync**, at the top.
2. Choose where to:
   - **Blup Cloud**: sign in to your account and create your notes password.
   - **GitHub**: the notes go to a private repository of yours. You don't need to know git.
3. On the other computer, open the Studio and sign in to the same account. With Blup Cloud, after you type your notes password, the Studio shows what is stored in the cloud and brings over what you choose. With GitHub, connect the same account and the Studio finds the sites stored there.

On the Free plan, syncing happens when you click the button. From Pro up, it happens in real time.

## On the command line

Run `blup sync` to sync with GitHub.

> [!info] Edited the same note in two places?
> Blup keeps both versions. See [Why is there a note with "conflict" in its name?](../Common%20problems/A%20note%20with%20conflict%20in%20its%20name.md)
