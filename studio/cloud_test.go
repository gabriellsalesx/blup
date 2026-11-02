package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
)

// Notificações desligadas sem rede: o cancelamento dos e-mails tenta de
// novo depois, até chegar ao Blup Cloud, e para se a pessoa religar.
func TestRetryCancel(t *testing.T) {
	offline := errors.New("dial tcp: no route to host")
	tries, waits := 0, 0
	retryCancel(func() bool { return true }, func(context.Context) error {
		tries++
		if tries < 3 {
			return offline
		}
		return nil
	}, time.Minute, func(d time.Duration) {
		if d != time.Minute {
			t.Fatalf("wait %v", d)
		}
		waits++
	})
	if tries != 3 || waits != 2 {
		t.Fatalf("tries %d waits %d", tries, waits)
	}

	// Turned on again meanwhile: it stops (the reminder list goes instead).
	tries = 0
	off := true
	retryCancel(func() bool { return off }, func(context.Context) error { tries++; return offline }, time.Minute, func(time.Duration) { off = false })
	if tries != 1 {
		t.Fatalf("kept trying after notifications came back: %d", tries)
	}

	// Signed out: nothing to cancel.
	tries = 0
	retryCancel(func() bool { return true }, func(context.Context) error { tries++; return blupcloud.ErrSignedOut }, time.Minute, func(time.Duration) { t.Fatal("waited") })
	if tries != 1 {
		t.Fatalf("tries %d", tries)
	}
}

// Quem traduz é escolha da pessoa: o antigo "cloud" (que deixava o Studio
// escolher o Blup Cloud sozinho) vale como sem escolha.
func TestTranslateProvider(t *testing.T) {
	for in, want := range map[string]string{"openrouter": "openrouter", "blup": "blup", "cloud": "", "local": "", "": "", "outro": ""} {
		if got := translateProvider(Prefs{TranslateAI: in}); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}
