// Command msix-manifest writes Blup Studio's AppxManifest.xml for the
// Microsoft Store from studio/build/windows/msix/AppxManifest.store.xml.
//
// The product identity comes from Partner Center (Product management >
// Product identity) and is never kept in this repository: the release
// workflow passes it from repository variables. Usage:
//
//	go run ./scripts/msix-manifest -version 1.2.0 -arch x64 -out AppxManifest.xml
//
// with MSIX_IDENTITY_NAME, MSIX_PUBLISHER and MSIX_PUBLISHER_DISPLAY_NAME
// set (or -name, -publisher and -display). The values are checked as the
// Store requires them: a version with a fourth section of 0 and a first
// one above 0, a publisher starting with CN=.
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

// Identity is what the manifest takes from Partner Center and the release.
type Identity struct {
	Name, Publisher, PublisherDisplayName, Version, Arch string
}

var (
	nameRe    = regexp.MustCompile(`^[A-Za-z0-9.-]{3,50}$`)
	versionRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
)

// storeVersion turns a release version (1.2.3) into the Store's (1.2.3.0):
// the fourth section is the Store's, the first can't be 0, none above 65535.
func storeVersion(v string) (string, error) {
	m := versionRe.FindStringSubmatch(strings.TrimPrefix(v, "v"))
	if m == nil {
		return "", fmt.Errorf("version %q is not X.Y.Z", v)
	}
	for i, s := range m[1:] {
		n, err := strconv.Atoi(s)
		if err != nil || n > 65535 || (i == 0 && n == 0) {
			return "", fmt.Errorf("version %q: the Store takes 1 to 65535 in the first section and 0 to 65535 in the others", v)
		}
	}
	return m[1] + "." + m[2] + "." + m[3] + ".0", nil
}

func storeArch(a string) (string, error) {
	switch a {
	case "x64", "amd64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	}
	return "", fmt.Errorf("architecture %q: x64 or arm64", a)
}

func clean(s string) bool {
	return !strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 || r == 0x7f })
}

// check makes id ready for the manifest, or tells what is wrong.
func (id Identity) check() (Identity, error) {
	var err error
	if !nameRe.MatchString(id.Name) {
		return id, errors.New("package name: 3 to 50 letters, digits, dots or dashes, as Partner Center shows it")
	}
	if !strings.HasPrefix(id.Publisher, "CN=") || len(id.Publisher) > 8192 || !clean(id.Publisher) {
		return id, errors.New("publisher: the CN=... value Partner Center shows")
	}
	if d := strings.TrimSpace(id.PublisherDisplayName); d == "" || len(d) > 256 || !clean(d) {
		return id, errors.New("publisher display name: the name Partner Center shows")
	}
	if id.Version, err = storeVersion(id.Version); err != nil {
		return id, err
	}
	if id.Arch, err = storeArch(id.Arch); err != nil {
		return id, err
	}
	return id, nil
}

func escape(s string) string {
	var b bytes.Buffer
	xml.EscapeText(&b, []byte(s))
	return b.String()
}

// render fills the template with id, escaped, and checks the result is a
// well formed manifest naming that identity.
func render(tmpl []byte, id Identity) ([]byte, error) {
	id, err := id.check()
	if err != nil {
		return nil, err
	}
	t, err := template.New("manifest").Option("missingkey=error").Parse(string(tmpl))
	if err != nil {
		return nil, err
	}
	esc := Identity{escape(id.Name), escape(id.Publisher), escape(id.PublisherDisplayName), id.Version, id.Arch}
	var out bytes.Buffer
	if err := t.Execute(&out, esc); err != nil {
		return nil, err
	}
	var pkg struct {
		XMLName  xml.Name `xml:"Package"`
		Identity struct {
			Name      string `xml:"Name,attr"`
			Publisher string `xml:"Publisher,attr"`
			Version   string `xml:"Version,attr"`
			Arch      string `xml:"ProcessorArchitecture,attr"`
		} `xml:"Identity"`
		PublisherDisplayName string `xml:"Properties>PublisherDisplayName"`
	}
	if err := xml.Unmarshal(out.Bytes(), &pkg); err != nil {
		return nil, fmt.Errorf("the manifest is not well formed: %w", err)
	}
	got := Identity{pkg.Identity.Name, pkg.Identity.Publisher, pkg.PublisherDisplayName, pkg.Identity.Version, pkg.Identity.Arch}
	if got != id {
		return nil, fmt.Errorf("the manifest names %+v, not %+v", got, id)
	}
	return out.Bytes(), nil
}

func main() {
	tmplPath := flag.String("template", "studio/build/windows/msix/AppxManifest.store.xml", "the manifest template")
	out := flag.String("out", "AppxManifest.xml", "where to write the manifest")
	id := Identity{}
	flag.StringVar(&id.Name, "name", os.Getenv("MSIX_IDENTITY_NAME"), "Package/Identity/Name (Partner Center)")
	flag.StringVar(&id.Publisher, "publisher", os.Getenv("MSIX_PUBLISHER"), "Package/Identity/Publisher, CN=... (Partner Center)")
	flag.StringVar(&id.PublisherDisplayName, "display", os.Getenv("MSIX_PUBLISHER_DISPLAY_NAME"), "Package/Properties/PublisherDisplayName (Partner Center)")
	flag.StringVar(&id.Version, "version", "", "the release version, X.Y.Z")
	flag.StringVar(&id.Arch, "arch", "x64", "x64 or arm64")
	flag.Parse()

	tmpl, err := os.ReadFile(*tmplPath)
	if err == nil {
		var b []byte
		if b, err = render(tmpl, id); err == nil {
			err = os.WriteFile(*out, b, 0o644)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "msix-manifest:", err)
		os.Exit(1)
	}
}
