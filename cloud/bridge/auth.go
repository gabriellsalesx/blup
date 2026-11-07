package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// Cada pedido entre o Blup Cloud e a Ponte vai assinado, nos dois sentidos:
// HMAC-SHA256 de "momento, nonce, método, caminho e o SHA-256 do corpo". Um
// pedido de mais de cinco minutos, ou repetido (mesmo nonce), é recusado.
const (
	headerTime      = "X-Bridge-Timestamp"
	headerSignature = "X-Bridge-Signature"
	headerNonce     = "X-Bridge-Nonce"
	skew            = 5 * time.Minute
)

var nonceFormat = regexp.MustCompile(`^[0-9a-f]{32}$`)

func signature(secret []byte, at int64, nonce, method, path string, body []byte) string {
	sum := sha256.Sum256(body)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(strconv.FormatInt(at, 10) + "\n" + nonce + "\n" + method + "\n" + path + "\n" + hex.EncodeToString(sum[:])))
	return hex.EncodeToString(mac.Sum(nil))
}

func sign(req *http.Request, secret []byte, body []byte, now time.Time) {
	at, raw := now.Unix(), make([]byte, 16)
	_, _ = rand.Read(raw)
	nonce := hex.EncodeToString(raw)
	req.Header.Set(headerTime, strconv.FormatInt(at, 10))
	req.Header.Set(headerNonce, nonce)
	req.Header.Set(headerSignature, signature(secret, at, nonce, req.Method, req.URL.RequestURI(), body))
}

// verifier lembra os nonces aceitos dentro da janela, para que um pedido
// capturado não possa ser reenviado.
type verifier struct {
	secret []byte
	now    func() time.Time
	mu     sync.Mutex
	seen   map[string]time.Time
}

func newVerifier(secret []byte, now func() time.Time) *verifier {
	return &verifier{secret: secret, now: now, seen: map[string]time.Time{}}
}

// plausible confere só os cabeçalhos (momento na janela, nonce e assinatura
// no formato), antes de ler o corpo: um pedido sem eles nem chega a gastar
// memória com até maxBody.
func (v *verifier) plausible(req *http.Request) bool {
	_, _, ok := v.headers(req, v.now())
	return ok
}

func (v *verifier) headers(req *http.Request, now time.Time) (at int64, got []byte, ok bool) {
	at, err := strconv.ParseInt(req.Header.Get(headerTime), 10, 64)
	if err != nil {
		return 0, nil, false
	}
	when := time.Unix(at, 0)
	if when.Before(now.Add(-skew)) || when.After(now.Add(skew)) {
		return 0, nil, false
	}
	got, err = hex.DecodeString(req.Header.Get(headerSignature))
	if err != nil || len(got) != sha256.Size || !nonceFormat.MatchString(req.Header.Get(headerNonce)) {
		return 0, nil, false
	}
	return at, got, true
}

func (v *verifier) check(req *http.Request, body []byte) bool {
	now := v.now()
	at, got, ok := v.headers(req, now)
	if !ok {
		return false
	}
	nonce := req.Header.Get(headerNonce)
	want, _ := hex.DecodeString(signature(v.secret, at, nonce, req.Method, req.URL.RequestURI(), body))
	if !hmac.Equal(got, want) {
		return false
	}
	key := nonce
	v.mu.Lock()
	defer v.mu.Unlock()
	for k, t := range v.seen {
		if t.Before(now.Add(-2 * skew)) {
			delete(v.seen, k)
		}
	}
	if _, again := v.seen[key]; again {
		return false
	}
	v.seen[key] = now
	return true
}
