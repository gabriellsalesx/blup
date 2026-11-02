package main

import (
	"os"
	"strings"
	"testing"
)

func storeTemplate(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile("../../studio/build/windows/msix/AppxManifest.store.xml")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Placeholders made up for the test (not a real Partner Center identity).
var good = Identity{"12345Example.BlupStudio", "CN=00000000-0000-0000-0000-000000000000", "Exemplo & Cia", "1.2.3", "amd64"}

func TestManifest(t *testing.T) {
	b, err := render(storeTemplate(t), good)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{
		`Name="12345Example.BlupStudio"`, `Version="1.2.3.0"`, `ProcessorArchitecture="x64"`,
		`<PublisherDisplayName>Exemplo &amp; Cia</PublisherDisplayName>`,
		`Executable="blup-studio.exe"`, `<rescap:Capability Name="runFullTrust" />`,
		`Square44x44Logo="Assets\Square44x44Logo.png"`, `Wide310x150Logo="Assets\Wide310x150Logo.png"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Contains(s, "{{") {
		t.Error("a value was left unfilled")
	}
}

// What the Store refuses never reaches a package.
func TestManifestChecks(t *testing.T) {
	bad := func(f func(*Identity)) Identity { id := good; f(&id); return id }
	for name, id := range map[string]Identity{
		"no name":         bad(func(i *Identity) { i.Name = "" }),
		"name with space": bad(func(i *Identity) { i.Name = "Blup Studio" }),
		"publisher":       bad(func(i *Identity) { i.Publisher = "Gabriell" }),
		"quote":           bad(func(i *Identity) { i.Publisher = "CN=x\"\n" }),
		"display":         bad(func(i *Identity) { i.PublisherDisplayName = " " }),
		"version 0":       bad(func(i *Identity) { i.Version = "0.9.0" }),
		"version big":     bad(func(i *Identity) { i.Version = "1.70000.0" }),
		"version 4 parts": bad(func(i *Identity) { i.Version = "1.2.3.4" }),
		"arch":            bad(func(i *Identity) { i.Arch = "x86" }),
	} {
		if _, err := render(storeTemplate(t), id); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	b, err := render(storeTemplate(t), bad(func(i *Identity) { i.Version, i.Arch = "v2.0.10", "arm64" }))
	if err != nil || !strings.Contains(string(b), `Version="2.0.10.0"`) || !strings.Contains(string(b), `ProcessorArchitecture="arm64"`) {
		t.Fatalf("arm64: %v", err)
	}
}
