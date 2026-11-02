// Package github publishes an already-built static site (Hugo's public/
// folder) to GitHub Pages using only the GitHub REST API. No git binary and
// no git knowledge are needed: files are hashed locally the same way git does
// (sha1("blob <len>\x00" + content)), compared with the branch's current tree,
// and only new or changed files are uploaded through the Git Data API.
//
// # Authentication
//
// A fine-grained personal access token limited to the repositories Blup
// uses. [TokenURL] opens https://github.com/settings/personal-access-tokens/new
// filled in with these repository permissions ([TokenPermissions]):
//
//	Contents: Read and write   git blobs/trees/commits/refs, contents PUT
//	Pages:    Read and write   GET/POST/PUT /repos/{o}/{r}/pages, builds
//	Issues:   Read and write   comments and reactions (feedback repository)
//	Metadata: Read-only        GET /repos/{o}/{r} (always granted)
//
// No Administration: the person picks "Only select repositories" and the
// repositories of the site, of the notes and of the comments. A link
// cannot choose them, and a key limited this way cannot create a
// repository, so the person creates it first through [NewRepoURL]
// (github.com/new filled in). [Client.EnsureRepoWith] still creates it
// when the key can (an older key), and otherwise returns a
// [RepoMissingError] with that link. Adding the "blup-site" topic needs
// Administration, so with these keys it is skipped.
// Source: https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens
//
// [DeviceFlow] (OAuth App) is kept but unused: OAuth Apps only have
// classic scopes and "repo" reaches every private repository.
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
// # Private repositories and note syncing
//
// [Client.EnsureRepoWith] can create a private (and optionally empty)
// repository; private repositories are free. internal/notesync builds a
// two-way note sync on the lower-level helpers [Client.BranchTip],
// [Client.ReadTree], [Client.CreateBlob], [Client.GetBlob],
// [Client.CreateTree], [Client.CreateCommit] and [Client.UpdateBranch]
// (never forced; a moved branch yields [ErrNotFastForward]). It needs only
// Contents read/write on that repository; Pages is not used.
//
// All requests send X-GitHub-Api-Version: 2026-03-10 (current version, see
// https://docs.github.com/en/rest/about-the-rest-api/api-versions).
package github
