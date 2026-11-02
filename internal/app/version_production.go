//go:build production

package app

// Builds tagged production (the Studio's Taskfiles and .goreleaser.yaml)
// must carry the version from -ldflags: a "dev" there would turn off
// update checks for every user. Failing at start makes the mistake show up
// on the first run of the build, before it ships.
func init() {
	if err := checkReleaseVersion(Version); err != nil {
		panic(err)
	}
}
