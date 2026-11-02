package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/config"
)

// O que sobe para o Blup Cloud nos lembretes por e-mail: só horário e um id
// opaco. O id não pode ser conferido contra palpites (pasta, nota, dia) por
// quem lê o servidor, e uma lista vazia cancela tudo.
func TestRemindersSendOnlyOpaqueIDAndTime(t *testing.T) {
	cfg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", cfg)
	t.Setenv("HOME", cfg)
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/reminders" || r.Method != http.MethodPut {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		w.Write([]byte(`{"limit":50,"used":0}`))
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	if err := (config.Credentials{BlupToken: "tok"}).Save(); err != nil {
		t.Fatal(err)
	}

	key := "/home/maria/Notas|Agenda/Consulta no cardiologista.md|2026-11-05|1d"
	id := ReminderID(key)
	plain := sha256.Sum256([]byte(key))
	if len(id) != 32 || id == hex.EncodeToString(plain[:16]) {
		t.Fatalf("id %q is a plain hash of the event", id)
	}
	if ReminderID(key) != id || ReminderID(key+"x") == id {
		t.Fatal("ids must be stable and distinct")
	}
	st, err := os.Stat(filepath.Join(cfg, "blup", "reminder-key"))
	if err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("reminder key file: %v %v", st, err)
	}

	ctx := context.Background()
	if _, err := PushReminders(ctx, []blupcloud.Reminder{{ID: id, At: 1762329600000}}); err != nil {
		t.Fatal(err)
	}
	if _, err := PushReminders(ctx, nil); err != nil { // notifications off
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("requests: %v", bodies)
	}
	var first struct {
		Reminders []map[string]any `json:"reminders"`
	}
	json.Unmarshal([]byte(bodies[0]), &first)
	if len(first.Reminders) != 1 || len(first.Reminders[0]) != 2 || first.Reminders[0]["id"] != id {
		t.Fatalf("sent more than id and time: %s", bodies[0])
	}
	for _, leak := range []string{"Agenda", "cardiologista", "maria", "2026-11-05"} {
		if strings.Contains(bodies[0], leak) {
			t.Fatalf("%q leaked: %s", leak, bodies[0])
		}
	}
	if strings.TrimSpace(bodies[1]) != `{"reminders":[]}` {
		t.Fatalf("cancel sent %s", bodies[1])
	}
}
