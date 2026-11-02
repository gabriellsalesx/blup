package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gabriellsalesx/blup/internal/blupcloud"
	"github.com/gabriellsalesx/blup/internal/i18n"
)

// TestDeployBusyMessage: a publish already under way reads as a clear
// sentence in each language, also when wrapped.
func TestDeployBusyMessage(t *testing.T) {
	defer i18n.Set(i18n.Current())
	want := map[i18n.Lang]string{
		i18n.PT: "Já tem uma publicação deste site em andamento. Espere terminar e tente de novo.",
		i18n.EN: "A publish of this site is already under way. Wait for it to end and try again.",
		i18n.ES: "Ya hay una publicación de este sitio en curso. Espera a que termine y vuelve a intentarlo.",
	}
	for l, msg := range want {
		i18n.Set(l)
		if got := describe(fmt.Errorf("deploy: %w", blupcloud.ErrDeployBusy)); got != msg {
			t.Errorf("%v: %q", l, got)
		}
	}
}

// TestQuotaMessage: a sync refused for the storage says that deleting
// notes or the old history frees space, in each language, without dashes.
func TestQuotaMessage(t *testing.T) {
	defer i18n.Set(i18n.Current())
	want := map[i18n.Lang][]string{
		i18n.PT: {"Apagar notas, anexos ou o histórico antigo", "libera espaço", "mude de plano"},
		i18n.EN: {"Deleting notes, attachments or the old history", "frees space", "change plan"},
		i18n.ES: {"Borrar notas, adjuntos o el historial antiguo", "libera espacio", "cambia de plan"},
	}
	for l, parts := range want {
		i18n.Set(l)
		got := describe(fmt.Errorf("sync: %w", blupcloud.ErrQuota))
		for _, p := range parts {
			if !strings.Contains(got, p) {
				t.Errorf("%v: %q lacks %q", l, got, p)
			}
		}
		if strings.ContainsAny(got, "—–") {
			t.Errorf("%v: a dash in %q", l, got)
		}
	}
}
