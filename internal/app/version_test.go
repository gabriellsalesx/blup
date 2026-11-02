package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCheckReleaseVersion(t *testing.T) {
	for _, v := range []string{"1.0.0", "v1.2.3", "1.0.0-rc1"} {
		if err := checkReleaseVersion(v); err != nil {
			t.Errorf("%q refused: %v", v, err)
		}
	}
	for _, v := range []string{"dev", "", "1.0", "0.0.0", "{{.VERSION}}"} {
		if checkReleaseVersion(v) == nil {
			t.Errorf("%q accepted", v)
		}
	}
}

// versionFlag is the -X flag that sets Version, from the real package path.
func versionFlag() string {
	return "-X " + reflect.TypeOf(knownError{}).PkgPath() + ".Version="
}

func readRepo(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

// The Studio's production builds and the CLI release write the version,
// and every number in the build assets agrees with config.yml.
func TestBuildsWriteTheVersion(t *testing.T) {
	cfg := regexp.MustCompile(`(?m)^\s+version: "([0-9.]+)" # The application version`).FindStringSubmatch(readRepo(t, "studio/build/config.yml"))
	if cfg == nil {
		t.Fatal("no info.version in studio/build/config.yml")
	}
	want := cfg[1]

	ldflags := regexp.MustCompile(`-ldflags="([^"]*)"`)
	for _, f := range []string{"studio/build/Taskfile.yml", "studio/build/windows/Taskfile.yml", "studio/build/darwin/Taskfile.yml", "studio/build/linux/Taskfile.yml"} {
		s := readRepo(t, f)
		found := ldflags.FindAllStringSubmatch(s, -1)
		if len(found) == 0 {
			t.Errorf("%s: no -ldflags", f)
		}
		for _, m := range found {
			if !strings.Contains(m[1], versionFlag()+"{{.VERSION}}") {
				t.Errorf("%s: -ldflags=%q does not set the version", f, m[1])
			}
		}
		for _, m := range regexp.MustCompile(`VERSION \| default "([^"]+)"`).FindAllStringSubmatch(s, -1) {
			if m[1] != want {
				t.Errorf("%s: VERSION default %s, config.yml says %s", f, m[1], want)
			}
		}
	}
	if s := readRepo(t, "studio/build/docker/Dockerfile.cross"); !strings.Contains(s, versionFlag()+"${VERSION:-"+want+"}") {
		t.Error("Dockerfile.cross does not set the version")
	}

	gr := readRepo(t, ".goreleaser.yaml")
	if !strings.Contains(gr, versionFlag()+"{{ .Version }}") || !strings.Contains(gr, "-tags=production") {
		t.Error(".goreleaser.yaml must set Version and build with -tags=production")
	}
	if !strings.Contains(gr, "./scripts/sign-release") || !strings.Contains(gr, "artifacts: checksum") {
		t.Error(".goreleaser.yaml must sign checksums.txt")
	}

	for f, needle := range map[string]string{
		"studio/build/darwin/Info.plist":             "<string>" + want + "</string>",
		"studio/build/ios/Info.plist":                "<string>" + want + "</string>",
		"studio/build/windows/info.json":             `"ProductVersion": "` + want + `"`,
		"studio/build/windows/wails.exe.manifest":    `version="` + want + `.0"`,
		"studio/build/windows/msix/app_manifest.xml": `Version="` + want + `.0"`,
		"studio/build/windows/msix/template.xml":     `Version="` + want + `.0"`,
		"studio/build/windows/nsis/wails_tools.nsh":  `INFO_PRODUCTVERSION "` + want + `"`,
		"studio/build/linux/nfpm/nfpm.yaml":          `version: "` + want + `"`,
	} {
		if !strings.Contains(readRepo(t, f), needle) {
			t.Errorf("%s: version is not %s", f, want)
		}
	}
}

// A production build without the version refuses to start; with it,
// it reports the version.
func TestProductionBuildNeedsVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "bin", "blup")
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	run := func(ldflags string) (string, error) {
		args := []string{"build", "-tags", "production", "-o", exe}
		if ldflags != "" {
			args = append(args, "-ldflags", ldflags)
		}
		build := exec.Command("go", append(args, "../../cmd/blup")...)
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("go build: %v\n%s", err, out)
		}
		cmd := exec.Command(exe, "version")
		cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_CONFIG_HOME="+dir, "APPDATA="+dir, "BLUP_NO_UPDATE_CHECK=1")
		out, err := cmd.CombinedOutput()
		return string(out), err
	}
	if out, err := run(""); err == nil || !strings.Contains(out, "production build without a version") {
		t.Fatalf("dev production build started: %v\n%s", err, out)
	}
	if out, err := run(versionFlag() + "1.0.0"); err != nil || !strings.Contains(out, "1.0.0") {
		t.Fatalf("versioned build: %v\n%s", err, out)
	}
}
