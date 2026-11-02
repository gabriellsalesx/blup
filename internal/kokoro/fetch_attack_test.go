package kokoro

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Um espelho/servidor hostil: conteúdo trocado é recusado pelo SHA-256, e
// uma resposta sem fim é cortada perto do tamanho fixado.
func TestFetchRefusesTamperedAndHuge(t *testing.T) {
	good := []byte("arquivo verdadeiro")
	sum := sha256.Sum256(good)
	body := good
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/huge" {
			chunk := make([]byte, 64<<10)
			for range 64 { // 4 MiB for a file pinned at 18 bytes
				if _, err := w.Write(chunk); err != nil {
					return
				}
			}
			return
		}
		w.Write(body)
	}))
	defer srv.Close()
	d := download{srv.URL + "/f", hex.EncodeToString(sum[:]), int64(len(good))}
	ctx := context.Background()
	if b, err := fetch(ctx, d, nil); err != nil || string(b) != string(good) {
		t.Fatalf("good file: %q %v", b, err)
	}
	body = []byte("arquivo trocado!!!")
	if _, err := fetch(ctx, d, nil); err == nil || !strings.Contains(err.Error(), "integrity") {
		t.Fatalf("tampered file: %v", err)
	}
	d.url = srv.URL + "/huge"
	var seen int64
	if _, err := fetch(ctx, d, func(n int64) { seen = n }); err == nil || !strings.Contains(err.Error(), "larger") {
		t.Fatalf("huge file: %v", err)
	}
	if seen > 2<<20 {
		t.Fatalf("read %d bytes before stopping", seen)
	}
}
