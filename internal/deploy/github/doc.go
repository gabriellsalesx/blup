// Package github publishes an already-built static site (Hugo's public/
// folder) to GitHub Pages using only the GitHub REST API. No git binary and
// no git knowledge are needed: files are hashed locally the same way git does
// (sha1("blob <len>\x00" + content)), compared with the branch's current tree,
// and only new or changed files are uploaded through the Git Data API.
//
// # Authentication
//
// Two options:
//
//   - A fine-grained personal access token (recommended for advanced users).
//     Create it at https://github.com/settings/personal-access-tokens/new with
//     "Repository access: All repositories" (needed so the token can create
//     and then reach the new repository; "Only select repositories" works when
//     the repository already exists) and these repository permissions:
//
//     Administration: Read and write   POST /user/repos (create the repository)
//     Contents:       Read and write   git blobs/trees/commits/refs, contents PUT
//     Pages:          Read and write   GET/POST/PUT /repos/{o}/{r}/pages, builds
//     Metadata:       Read-only        GET /repos/{o}/{r} (always granted)
//
//     Source: https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens
//     GET /user needs no extra permission.
//
//   - The OAuth Device Flow (friendliest for laypeople): the CLI shows a short
//     code, the person opens https://github.com/login/device, types it and
//     clicks Authorize. See [DeviceFlow]. The OAuth App must have "Enable
//     Device Flow" ticked in its settings. Scope requested: "repo" (see
//     [DefaultDeviceScopes]); the Pages REST docs state OAuth app tokens need
//     the "repo" scope for the Pages endpoints.
//     Sources: https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps#device-flow
//     https://docs.github.com/en/rest/pages/pages
//     https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/scopes-for-oauth-apps
//
// # Limits handled
//
//   - Content-creation secondary limit (about 80 writes/minute, 500/hour):
//     writes are paced by [Client.MinWriteInterval]; text files are sent inline
//     in the tree request instead of one blob request per file; 403/429 with
//     Retry-After or x-ratelimit-reset are retried with backoff.
//     https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
//   - Recursive tree GET is capped at 100,000 entries / 7 MB; when the
//     response says truncated, subtrees are fetched one by one.
//     https://docs.github.com/en/rest/git/trees
//   - Large trees are created in chunks chained with base_tree (the first
//     chunk has no base so deleted files disappear).
//   - Files over 100 MiB are rejected (GitHub's per-file hard limit).
//
// All requests send X-GitHub-Api-Version: 2026-03-10 (current version, see
// https://docs.github.com/en/rest/about-the-rest-api/api-versions).
package github
