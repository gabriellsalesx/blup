package blupcloud

// Page is an address of the Blup Cloud panel that the CLI and the Studio
// open in the browser. Only these exist; the panel's own routes are English
// (its pages are translated) and must match cloud/shared/panel-paths.ts.
type Page string

const (
	PageAccount Page = "/account"
	PagePlan    Page = "/account/plan"
)

// Valid reports whether p is one of the pages above.
func (p Page) Valid() bool {
	switch p {
	case PageAccount, PagePlan:
		return true
	}
	return false
}

// URL is the page's full address on BaseURL.
func (p Page) URL() string { return BaseURL() + string(p) }
