package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// ErrNotLive: the plan syncs by button only (no live room).
var ErrNotLive = errors.New("blupcloud: the plan has no live sync")

// Moved says a vault reached version N on another computer.
type Moved struct {
	Vault string `json:"vault"`
	N     int64  `json:"n"`
}

// The live room's timings: the handshake gives up after liveDial, a ping
// goes every livePing (the room answers "pong" itself), and a connection
// that hears nothing for liveIdle is taken as dead (a network that dropped
// without closing it), so the caller reconnects. Tests shorten them.
var (
	liveDial = 30 * time.Second
	livePing = 30 * time.Second
	liveIdle = 3 * livePing
)

// Listen joins the account's live room and calls moved for every new
// version saved by another computer (or by this one: compare N with the
// version synced here). It returns when ctx ends or the connection drops
// (or goes quiet for too long); call it again to reconnect.
func (c *Client) Listen(ctx context.Context, moved func(Moved)) error {
	// Ends the ping below as soon as Listen returns.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	u := c.BaseURL + "/api/sync/live"
	u = "ws" + strings.TrimPrefix(u, "http") // http→ws, https→wss
	conn, res, err := websocket.Dial(ctx, u, &websocket.DialOptions{
		// Bounds the handshake only (the library drops it afterwards).
		HTTPClient: &http.Client{Timeout: liveDial},
		HTTPHeader: http.Header{"Authorization": {"Bearer " + c.Token}, "User-Agent": {UserAgent()}},
	})
	if err != nil {
		if res != nil {
			switch res.StatusCode {
			case http.StatusUnauthorized:
				return ErrSignedOut
			case http.StatusForbidden:
				return ErrNotLive
			}
		}
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(1 << 16)

	// The room answers "ping" itself, without waking up; this keeps
	// proxies from closing a quiet connection. It ends before Listen
	// returns.
	var pinging sync.WaitGroup
	defer func() { cancel(); pinging.Wait() }()
	ping, idle := livePing, liveIdle
	pinging.Go(func() {
		t := time.NewTicker(ping)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if conn.Write(ctx, websocket.MessageText, []byte("ping")) != nil {
					return
				}
			}
		}
	})
	for {
		rctx, rcancel := context.WithTimeout(ctx, idle)
		_, data, err := conn.Read(rctx)
		rcancel()
		if err != nil {
			if ctx.Err() == nil && rctx.Err() != nil {
				return fmt.Errorf("blupcloud: live room quiet for %s: %w", idle, err)
			}
			return err
		}
		var m Moved
		// Only what a vault's change looks like: an id the cloud makes
		// and a version number.
		if json.Unmarshal(data, &m) == nil && ValidVaultID(m.Vault) && m.N > 0 {
			moved(m)
		}
	}
}
