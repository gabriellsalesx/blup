package blupcloud

import "testing"

func TestPageURLs(t *testing.T) {
	t.Setenv("BLUP_CLOUD_URL", "https://cloud.audit.invalid/")
	for page, want := range map[Page]string{PageAccount: "https://cloud.audit.invalid/account", PagePlan: "https://cloud.audit.invalid/account/plan"} {
		if !page.Valid() || page.URL() != want {
			t.Fatalf("%q: valid=%v url=%q", page, page.Valid(), page.URL())
		}
	}
	for _, bad := range []Page{"", "/conta", "/account/../staff", "https://evil.invalid"} {
		if bad.Valid() {
			t.Fatalf("%q should not be valid", bad)
		}
	}
}
