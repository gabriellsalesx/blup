package app

import (
	"fmt"
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
