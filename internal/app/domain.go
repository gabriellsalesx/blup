package app

import "golang.org/x/net/publicsuffix"

// A site's own domain is linked only on Blup Cloud (domain_blup.go): the
// cloud manages it end to end. Sites on Cloudflare Workers or GitHub Pages
// keep their free address.

// isApex reports whether d is a root domain (meusite.com.br), not a
// subdomain (blog.meusite.com.br).
func isApex(d string) bool {
	root, err := publicsuffix.EffectiveTLDPlusOne(d)
	return err == nil && root == d
}
