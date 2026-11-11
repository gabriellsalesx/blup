//go:build !linux || android

package main

import (
	"context"
	"errors"
)

func portalAutostart(context.Context, bool, string) error {
	return errors.New("the desktop autostart portal is only available on Linux")
}

func portalAutostartEnabled() bool { return false }
