package blupcloud

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func resetDevice(t *testing.T) {
	t.Helper()
	reset := func() {
		thisDevice.Lock()
		thisDevice.set, thisDevice.mark, thisDevice.name = false, "", ""
		thisDevice.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

// The mark is the same for the same system id (so a reinstall is the same
// device), never the id itself, and empty without one.
func TestDeviceMark(t *testing.T) {
	resetDevice(t)
	SetDevice(" 9774d56d682e549c ", "Galaxy S23")
	a := DeviceMark()
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(a) || strings.Contains(a, "9774d56d682e549c") {
		t.Fatalf("mark %q", a)
	}
	SetDevice("9774d56d682e549c", "Galaxy S23")
	if DeviceMark() != a {
		t.Fatal("the same device must give the same mark")
	}
	SetDevice("another", "")
	if DeviceMark() == a {
		t.Fatal("another device must give another mark")
	}
	SetDevice("", "x")
	if DeviceMark() != "" {
		t.Fatal("no system id, no mark")
	}
}

func TestDeviceNameInUserAgent(t *testing.T) {
	resetDevice(t)
	if ua := UserAgent(); !strings.HasPrefix(ua, "Blup Studio (") {
		t.Fatalf("the Studio keeps its name: %s", ua)
	}
	SetDevice("id", "Galaxy (S23); x\n")
	if ua := UserAgent(); !strings.HasPrefix(ua, "Blup Notes (") || !strings.HasSuffix(ua, "; Galaxy S23 x)") {
		t.Fatalf("user agent %q", ua)
	}
	if n := cleanName(strings.Repeat("é", 80)); len([]rune(n)) != 64 {
		t.Fatalf("name not cut at 64 characters: %d", len([]rune(n)))
	}
}

// Only the request for a sign-in code carries the mark.
func TestDeviceHeaderOnlyOnCode(t *testing.T) {
	resetDevice(t)
	SetDevice("phone-id", "Galaxy S23")
	seen := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path] = r.Header.Get(DeviceHeader)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/auth/device/code" {
			w.Write([]byte(`{"device_code":"d","user_code":"ABCDEFGH","verification_uri":"https://x/device","verification_uri_complete":"https://x/device?user_code=ABCDEFGH","interval":5,"expires_in":900}`))
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	t.Setenv("BLUP_CLOUD_URL", srv.URL)
	if _, err := StartLogin(t.Context()); err != nil {
		t.Fatal(err)
	}
	c := &Client{BaseURL: srv.URL, Token: "t", HTTP: srv.Client()}
	var out map[string]any
	_ = c.do(t.Context(), http.MethodGet, "/me", nil, "", &out)
	if seen["/api/auth/device/code"] != DeviceMark() || seen["/api/me"] != "" {
		t.Fatalf("headers %v", seen)
	}
}
