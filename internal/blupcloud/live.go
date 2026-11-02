package blupcloud

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

// Listen joins the account's live room and calls moved for every new
// version saved by another computer (or by this one: compare N with the
// version synced here). It returns when ctx ends or the connection drops;
// call it again to reconnect.
func (c *Client) Listen(ctx context.Context, moved func(Moved)) error {
	u := c.BaseURL + "/api/sync/live"
	u = "ws" + strings.TrimPrefix(u, "http") // http→ws, https→wss
	conn, res, err := websocket.Dial(ctx, u, &websocket.DialOptions{
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
	// proxies from closing a quiet connection.
	go func() {
		t := time.NewTicker(30 * time.Second)
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
	}()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
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
