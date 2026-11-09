package app

import (
	"context"
	"testing"

	"github.com/gabriellsalesx/blup/internal/i18n"
)

// Without a terminal (Blup Studio, go test) the clients never ask for a
// key: they say what to connect instead of failing with a TTY error.
func TestClientsNeverAskWithoutTerminal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	if _, _, err := githubClient(context.Background()); err == nil || err.Error() != i18n.T("pub.connect_github") {
		t.Fatalf("githubClient: %v", err)
	}
	if _, err := cloudflareClient(context.Background()); err == nil || err.Error() != i18n.T("pub.connect_cloudflare") {
		t.Fatalf("cloudflareClient: %v", err)
	}
}
