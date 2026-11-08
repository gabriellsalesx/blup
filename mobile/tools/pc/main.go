// pc plays a second device for the Blup Notes sync tests, on a throwaway
// folder, against the Blup Cloud on this computer.
//
//	pc <dir> <cookie file> <notes password>
package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/cloudcrypt"
	"github.com/gabriellsalesx/blup/internal/config"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// client talks to the local Blup Cloud; a stuck request fails the test.
var client = &http.Client{Timeout: 30 * time.Second}

func approve(ctx context.Context, cookie, code string) {
	clean := strings.ReplaceAll(code, "-", "")
	claim, err := http.NewRequestWithContext(ctx, http.MethodGet, blupcloud.BaseURL()+"/api/auth/device?user_code="+clean, nil)
	if err != nil {
		log.Fatal(err)
	}
	claim.Header.Set("origin", blupcloud.BaseURL())
	claim.Header.Set("cookie", cookie)
	first, err := client.Do(claim)
	if err != nil {
		log.Fatal(err)
	}
	first.Body.Close()
	body := fmt.Sprintf(`{"userCode":%q}`, clean)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, blupcloud.BaseURL()+"/api/auth/device/approve", strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("origin", blupcloud.BaseURL())
	req.Header.Set("cookie", cookie)
	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Fatalf("approve: %s", res.Status)
	}
}

func main() {
	dir, cookieFile, password := os.Args[1], os.Args[2], os.Args[3]
	os.Setenv("BLUP_CLOUD_URL", "http://localhost:18820")
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	raw, err := os.ReadFile(cookieFile)
	if err != nil {
		log.Fatal(err)
	}
	creds, _ := config.LoadCredentials()
	if creds.BlupToken == "" {
		l, err := blupcloud.StartLogin(ctx)
		if err != nil {
			log.Fatal(err)
		}
		approve(ctx, strings.TrimSpace(string(raw)), l.Code)
		if creds.BlupToken, err = l.Wait(ctx); err != nil {
			log.Fatal(err)
		}
	}
	c := blupcloud.New(creds.BlupToken)
	w, err := c.Key(ctx)
	if err != nil {
		log.Fatal(err)
	}
	master, err := cloudcrypt.Unlock(w, password)
	if err != nil {
		log.Fatal("unlock: ", err)
	}
	creds.NotesKey = base64.StdEncoding.EncodeToString(master)
	if err := creds.Save(); err != nil {
		log.Fatal(err)
	}
	k, err := cloudcrypt.Derive(master)
	if err != nil {
		log.Fatal(err)
	}
	vaults, err := c.Vaults(ctx)
	if err != nil || len(vaults) == 0 {
		log.Fatal("vaults: ", err, len(vaults))
	}
	name, _, _ := k.OpenName(vaults[0].ID, vaults[0].Name)
	me, err := c.Me(ctx)
	if err != nil {
		log.Fatal(err)
	}
	notes := filepath.Join(dir, "notas")
	os.MkdirAll(notes, 0o700)
	sy := &notesync.Syncer{
		Remote: &blupcloud.Remote{Client: c, Keys: k, Vault: vaults[0].ID, MaxFile: int64(me.Plan.Limits.MaxFileMB) << 20, Cache: filepath.Join(dir, "cache")},
		Dir:    notes, StateFile: filepath.Join(dir, "state.json"),
	}
	r, err := sy.Sync(ctx, nil)
	if err != nil {
		log.Fatal("sync: ", err)
	}
	fmt.Printf("cofre=%q enviou=%d recebeu=%d conflitos=%v\n", name, len(r.Uploaded)+len(r.Deleted), len(r.Downloaded)+len(r.DeletedLocal), r.ConflictCopies)
}
