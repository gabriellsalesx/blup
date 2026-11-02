package blupcloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
)

// Preparing a version: above the plan's storage, a version that swaps
// blocks (an edited note) only goes up when it doesn't grow the use. The
// cloud is told the whole version first (every block, the manifest), works
// out what it frees, and gives a ticket under which the blocks it lacks go
// up; then the version is saved with the ticket. The contract is the
// cloud's (cloud/worker, agreed with its team):
//
//	POST   /api/sync/vaults/:id/prepare  multipart parent, blocks (JSON
//	       [{id, size}]: every block of the version, sealed sizes; 0 for a
//	       block the cloud has, whose size it knows), manifest (sealed)
//	       -> {ticket, expiresAt, missing}
//	PUT    /api/sync/blocks/:id  with x-blup-upload-ticket: <ticket>
//	POST   /api/sync/vaults/:id/commit {ticket} -> {n}
//	DELETE /api/sync/preparations/:ticket  (cancels; safe to repeat)
//
// Errors: 413 QUOTA (the version grows the use above the limit), 409 MOVED
// (another computer saved first), 409 SYNC_BUSY, 410 TICKET_EXPIRED.

// BlockSize is a block of a prepared version and its sealed size.
type BlockSize struct {
	ID   string `json:"id"`
	Size int64  `json:"size"`
}

// Prepared is a version the cloud took in: its ticket and the blocks to
// send under it.
type Prepared struct {
	Ticket    string   `json:"ticket"`
	ExpiresAt string   `json:"expiresAt"`
	Missing   []string `json:"missing"`
}

type ticketKey struct{}

// validTicket: what goes in a path and a header, nothing else.
func validTicket(t string) bool {
	if t == "" || len(t) > 200 {
		return false
	}
	for _, c := range t {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

// Prepare announces version parent+1: blocks are all of its blocks, each
// once.
func (c *Client) Prepare(ctx context.Context, vault string, parent int64, blocks []BlockSize, sealedManifest []byte) (Prepared, error) {
	listed := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		if !validBlockID(b.ID) || listed[b.ID] || b.Size < 0 {
			return Prepared{}, fmt.Errorf("blupcloud: bad block %q", b.ID)
		}
		listed[b.ID] = true
	}
	list, err := json.Marshal(blocks)
	if err != nil {
		return Prepared{}, err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("parent", strconv.FormatInt(parent, 10))
	w.WriteField("blocks", string(list))
	f, err := w.CreateFormFile("manifest", "manifest")
	if err != nil {
		return Prepared{}, err
	}
	f.Write(sealedManifest)
	if err := w.Close(); err != nil {
		return Prepared{}, err
	}
	var p Prepared
	if err := c.do(ctx, http.MethodPost, "/sync/vaults/"+url.PathEscape(vault)+"/prepare", &buf, w.FormDataContentType(), &p); err != nil {
		return Prepared{}, err
	}
	if !validTicket(p.Ticket) {
		return Prepared{}, fmt.Errorf("%w: ticket", ErrBadAnswer)
	}
	for _, id := range p.Missing {
		if !listed[id] {
			return Prepared{}, fmt.Errorf("%w: missing block not in the version", ErrBadAnswer)
		}
	}
	return p, nil
}

// PutPreparedBlock sends a block the prepared version lacks.
func (c *Client) PutPreparedBlock(ctx context.Context, ticket, id string, sealed []byte) error {
	if !validTicket(ticket) {
		return ErrTicketExpired
	}
	return c.PutBlock(context.WithValue(ctx, ticketKey{}, ticket), id, sealed)
}

// CommitPrepared saves the prepared version; ErrMoved when another
// computer saved first.
func (c *Client) CommitPrepared(ctx context.Context, vault, ticket string, parent int64) (int64, error) {
	if !validTicket(ticket) {
		return 0, ErrTicketExpired
	}
	var r struct {
		N int64 `json:"n"`
	}
	if err := c.do(ctx, http.MethodPost, "/sync/vaults/"+url.PathEscape(vault)+"/commit", map[string]string{"ticket": ticket}, "", &r); err != nil {
		return 0, err
	}
	if r.N != parent+1 {
		return 0, fmt.Errorf("%w: saved as version %d after %d", ErrBadAnswer, r.N, parent)
	}
	return r.N, nil
}

// CancelPrepared lets go of a prepared version (nothing saved stays
// touched).
func (c *Client) CancelPrepared(ctx context.Context, ticket string) error {
	if !validTicket(ticket) {
		return nil
	}
	return c.do(ctx, http.MethodDelete, "/sync/preparations/"+url.PathEscape(ticket), nil, "", nil)
}
