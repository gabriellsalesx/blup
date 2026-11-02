package app

import (
	"fmt"

	"github.com/gabriellsalesx/blup/internal/update"
)

// checkReleaseVersion tells whether v can be the Version of a production
// build: a real x.y.z (optional "v"), never the "dev" of go run.
func checkReleaseVersion(v string) error {
	if !update.Newer(v, "0.0.0") {
		return fmt.Errorf("blup: production build without a version (Version=%q); build with -ldflags \"-X github.com/gabriellsalesx/blup/internal/app.Version=x.y.z\"", v)
	}
	return nil
}
