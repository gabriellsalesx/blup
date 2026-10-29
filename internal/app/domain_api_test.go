package app

import (
	"context"
	"testing"
)

func TestDomainUnpublished(t *testing.T) {
	site := settingsSite(t)
	if _, err := SetDomain(context.Background(), site, "nao é domínio"); err == nil {
		t.Fatal("accepted a bad domain")
	}
	st, err := SetDomain(context.Background(), site, "https://www.MeuSite.com.br/")
	if err != nil || st.Domain != "www.meusite.com.br" || st.State != "unpublished" || site.Domain != "www.meusite.com.br" {
		t.Fatalf("set: %+v %v", st, err)
	}
	if err := RemoveDomain(context.Background(), site); err != nil || site.Domain != "" {
		t.Fatalf("remove: %v %q", err, site.Domain)
	}
}
