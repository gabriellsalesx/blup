package blupcloud

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// saying serves body to every call.
func saying(t *testing.T, body string) *Client {
	return answering(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	})
}

const okSite = `{"id":"s_1-a","slug":"meu-blog","title":"Blog","url":"https://meu-blog.getblup.site/","live":true,"bytes":10,"files":2}`

// An answer without what the Studio uses is an error, never "no sites"
// (that reads as every site deleted) or a site with no id.
func TestSiteAnswersChecked(t *testing.T) {
	ctx := context.Background()
	for _, body := range []string{
		`{}`, `{"limit":1}`, `{"sites":[]}`, `{"limit":1,"sites":null}`, `{"limit":-2,"sites":[]}`,
		`{"limit":1,"sites":[{}]}`,
		`{"limit":1,"sites":[{"id":"../x","slug":"meu-blog","url":"https://a.b/"}]}`,
		`{"limit":1,"sites":[{"id":"s1","slug":"Meu Blog","url":"https://a.b/"}]}`,
		`{"limit":1,"sites":[{"id":"s1","slug":"meu-blog","url":"javascript:alert(1)"}]}`,
		`{"limit":1,"sites":[{"id":"s1","slug":"meu-blog","url":"https://a.b/","bytes":-1}]}`,
	} {
		if _, _, err := saying(t, body).Sites(ctx); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("Sites(%s) = %v", body, err)
		}
	}
	sites, limit, err := saying(t, `{"limit":-1,"downgrade":null,"sites":[`+okSite+`]}`).Sites(ctx)
	if err != nil || limit != -1 || len(sites) != 1 || sites[0].ID != "s_1-a" {
		t.Fatalf("a good list: %v %d %v", sites, limit, err)
	}
	if sites, _, err := saying(t, `{"limit":0,"sites":[]}`).Sites(ctx); err != nil || len(sites) != 0 {
		t.Fatalf("no sites: %v", err)
	}

	for _, body := range []string{`{}`, `{"id":"s1","slug":"outro","url":"https://outro.getblup.site/"}`, `{"id":"","slug":"meu-blog","url":"https://meu-blog.getblup.site/"}`} {
		if _, err := saying(t, body).CreateSite(ctx, "meu-blog", "Blog"); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("CreateSite(%s) = %v", body, err)
		}
	}
	// The title is optional.
	if s, err := saying(t, `{"id":"s1","slug":"meu-blog","url":"http://meu-blog.localhost:8788/"}`).CreateSite(ctx, "meu-blog", ""); err != nil || s.ID != "s1" {
		t.Fatalf("created: %+v %v", s, err)
	}

	for _, body := range []string{`{}`, `{"status":"maybe"}`, `{"status":null}`} {
		if _, err := saying(t, body).CheckSlug(ctx, "meu-blog"); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("CheckSlug(%s) = %v", body, err)
		}
	}
	r, err := saying(t, `{"status":"taken","suggestions":["meu-blog-2","<b>x</b>"]}`).CheckSlug(ctx, "meu-blog")
	if err != nil || r.Status != "taken" || len(r.Suggestions) != 1 || r.Suggestions[0] != "meu-blog-2" {
		t.Fatalf("taken: %+v %v", r, err)
	}
	if r, err := saying(t, `{"status":"free"}`).CheckSlug(ctx, "meu-blog"); err != nil || r.Suggestions == nil {
		t.Fatalf("free without suggestions: %+v %v", r, err)
	}
}

// The other answers: the fields must be there; 0 and [] still go.
func TestMoreAnswersChecked(t *testing.T) {
	ctx := context.Background()
	a, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	for _, body := range []string{`{}`, `{"missing":null}`, `{"missing":["` + strings.Repeat("c", 64) + `"]}`, `{"missing":["` + a + `","` + a + `"]}`} {
		if _, err := saying(t, body).Missing(ctx, []string{a, b}); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("Missing(%s) = %v", body, err)
		}
		if _, err := saying(t, body).DeploySite(ctx, "s1", []SiteFile{{Path: "/a", Hash: a}, {Path: "/b", Hash: b}}); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("DeploySite(%s) = %v", body, err)
		}
	}
	if m, err := saying(t, `{"missing":[]}`).Missing(ctx, []string{a, b}); err != nil || len(m) != 0 {
		t.Fatalf("nothing missing: %v %v", m, err)
	}
	if m, err := saying(t, `{"missing":["`+b+`"]}`).Missing(ctx, []string{a, b}); err != nil || len(m) != 1 {
		t.Fatalf("one missing: %v %v", m, err)
	}

	for _, body := range []string{`{}`, `{"limit":5}`, `{"used":1}`, `{"limit":-1,"used":0}`, `{"limit":5,"used":-1}`} {
		if _, err := saying(t, body).ReminderEmails(ctx); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("ReminderEmails(%s) = %v", body, err)
		}
		if _, err := saying(t, body).SetReminders(ctx, nil); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("SetReminders(%s) = %v", body, err)
		}
	}
	// A plan that went down may have used more than it now allows.
	if u, err := saying(t, `{"limit":0,"used":3,"saved":0}`).ReminderEmails(ctx); err != nil || u.Used != 3 {
		t.Fatalf("over the limit: %+v %v", u, err)
	}

	for _, body := range []string{`{}`, `{"freedBytes":1}`, `{"storageBytes":1}`, `{"freedBytes":-1,"storageBytes":0}`} {
		if _, err := saying(t, body).PurgeHistory(ctx, vaultID); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("PurgeHistory(%s) = %v", body, err)
		}
	}
	if r, err := saying(t, `{"freedBytes":0,"storageBytes":0}`).PurgeHistory(ctx, vaultID); err != nil || r.FreedBytes != 0 {
		t.Fatalf("nothing freed: %v", err)
	}

	for _, body := range []string{`{}`, `{"vaults":null}`} {
		if _, err := saying(t, body).Vaults(ctx); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("Vaults(%s) = %v", body, err)
		}
	}
	if v, err := saying(t, `{"vaults":[]}`).Vaults(ctx); err != nil || len(v) != 0 {
		t.Fatalf("no vaults: %v", err)
	}
	for _, body := range []string{`{}`, `{"wrapped":{}}`, `{"wrapped":null}`} {
		if _, err := saying(t, body).Key(ctx); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("Key(%s) = %v", body, err)
		}
	}
}

// The account's id is shown and copied for help: only an id as the cloud
// makes them.
func TestAccountID(t *testing.T) {
	ctx := context.Background()
	me := func(id string) string {
		return `{"id":` + id + `,"plan":{"name":"free","limits":{"sites":0,"devices":-1}},"usage":{}}`
	}
	a, err := saying(t, me(`"0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e"`)).Me(ctx)
	if err != nil || a.ID != "0199f2a4-7c3e-7b1a-9d2e-4f5a6b7c8d9e" {
		t.Fatalf("a good id: %q %v", a.ID, err)
	}
	for _, id := range []string{`""`, `null`, `"<b>x</b>"`, `"a b"`, `"` + strings.Repeat("a", 129) + `"`} {
		if _, err := saying(t, me(id)).Me(ctx); !errors.Is(err, ErrBadAnswer) {
			t.Errorf("id %s: %v", id, err)
		}
	}
	if _, err := saying(t, me(`5`)).Me(ctx); err == nil {
		t.Error("a number was taken as the id")
	}
}

// Reminders tell when the account has no e-mail (passkey only); an older
// cloud that doesn't say means an e-mail is there.
func TestReminderEmailMissing(t *testing.T) {
	ctx := context.Background()
	if u, err := saying(t, `{"limit":5,"used":0,"emailMissing":true}`).ReminderEmails(ctx); err != nil || !u.EmailMissing {
		t.Fatalf("no e-mail: %+v %v", u, err)
	}
	if u, err := saying(t, `{"limit":5,"used":0}`).ReminderEmails(ctx); err != nil || u.EmailMissing {
		t.Fatalf("older cloud: %+v %v", u, err)
	}
	if _, err := saying(t, `{"limit":5,"used":0,"emailMissing":"yes"}`).ReminderEmails(ctx); err == nil {
		t.Fatal("a text was taken as emailMissing")
	}
}
