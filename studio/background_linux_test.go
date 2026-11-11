//go:build linux && !android

package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
)

type fakeBackground struct {
	code    uint32
	results map[string]dbus.Variant
	wait    bool
	closed  bool
	signals chan *dbus.Signal
	options map[string]dbus.Variant
}

func (*fakeBackground) name() string { return ":1.42" }

func (f *fakeBackground) subscribe(_ context.Context, namespace dbus.ObjectPath, signals chan *dbus.Signal) (func(), error) {
	f.signals = signals
	return func() { f.signals = nil }, nil
}

func (f *fakeBackground) request(_ context.Context, options map[string]dbus.Variant) (dbus.ObjectPath, error) {
	f.options = options
	path := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/1_42/" + options["handle_token"].Value().(string))
	if !f.wait {
		// Emit before returning the handle: subscription must precede the
		// request. Also send another request's response, which must be ignored.
		f.signals <- &dbus.Signal{Path: path + "_other", Name: backgroundRequest + ".Response", Body: []any{uint32(0), map[string]dbus.Variant{"autostart": dbus.MakeVariant(!options["autostart"].Value().(bool))}}}
		f.signals <- &dbus.Signal{Path: path, Name: backgroundRequest + ".Response", Body: []any{f.code, f.results}}
	}
	return path, nil
}

func (f *fakeBackground) closeRequest(context.Context, dbus.ObjectPath) { f.closed = true }

func TestPortalAutostartRequiresDesktopApproval(t *testing.T) {
	for _, c := range []struct {
		name   string
		on     bool
		code   uint32
		grant  any
		wantOK bool
	}{
		{"enable", true, 0, true, true},
		{"disable", false, 0, false, true},
		{"denied", true, 0, false, false},
		{"cancelled", true, 1, true, false},
		{"invalid result", true, 0, "true", false},
		{"missing result", true, 0, nil, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			f := &fakeBackground{code: c.code, results: map[string]dbus.Variant{}}
			if c.grant != nil {
				f.results["autostart"] = dbus.MakeVariant(c.grant)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			err := applyPortalAutostart(ctx, f, c.on, "fixture")
			if (err == nil) != c.wantOK {
				t.Fatalf("portal result: %v", err)
			}
			if got := portalAutostartEnabled(); got != (c.on && c.wantOK) {
				t.Fatalf("unconfirmed autostart persisted: %v", got)
			}
			if !c.wantOK {
				file, _ := portalAutostartFile()
				if _, err := os.Stat(file); !os.IsNotExist(err) {
					t.Fatal("denied or cancelled request changed local state")
				}
			}
			if f.options["autostart"].Value() != c.on {
				t.Fatal("wrong portal option")
			}
			var argv []string
			if err := f.options["commandline"].Store(&argv); err != nil || len(argv) != 2 || argv[0] != "blup-studio" || argv[1] != hiddenArg {
				t.Fatalf("invalid autostart command: %v, %v", argv, err)
			}
		})
	}
}

func TestPortalCancellationClosesRequest(t *testing.T) {
	f := &fakeBackground{wait: true}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := requestPortalAutostart(ctx, f, true, "fixture"); err == nil {
		t.Fatal("a portal without a response was accepted")
	}
	if !f.closed {
		t.Fatal("the cancelled portal request was not closed")
	}
}
