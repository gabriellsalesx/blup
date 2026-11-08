package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/i18n"
	"github.com/gabriellsalesx/blup/internal/notesync"
)

// A sync parada por segurança (bifurcação ou versão mais antiga) chega à
// janela como SyncStuckError: a mensagem do aviso e, no JSON que o Wails
// manda como cause, stuck: true. Os outros erros seguem como eram.
func TestSyncFailedMarksStuck(t *testing.T) {
	for _, base := range []error{notesync.ErrForked, notesync.ErrRolledBack} {
		err := syncFailed(fmt.Errorf("%w (version 3)", base))
		if !errors.Is(err, ErrSyncStuck) {
			t.Fatalf("%v: not stuck", base)
		}
		if err.Error() != i18n.T("blup.sync_stuck") {
			t.Fatalf("message = %q", err.Error())
		}
		// O que o Wails faz com o erro (defaultMarshalError).
		b, jerr := json.Marshal(&err)
		if jerr != nil || string(b) != `{"stuck":true}` {
			t.Fatalf("cause = %s, %v", b, jerr)
		}
	}
	err := syncFailed(blupcloud.ErrQuota)
	if errors.Is(err, ErrSyncStuck) || err.Error() != i18n.T("blup.err_quota") {
		t.Fatalf("other error = %v", err)
	}
}
