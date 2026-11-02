package update

import (
	"crypto/ed25519"
	"encoding/base64"
)

// publicKeys are the Ed25519 keys whose signature on checksums.txt an
// update must carry. Only the public half lives here; the private one is
// kept by the owner outside the repository (see scripts/sign-release).
// To rotate, add the new key, release with it, then drop the old one.
var publicKeys = []string{
	"lWau179UvGPPoNKqfHgPoy0AqkZKuWUiUqjNNykvSRI=", // 2026-11, ~/.config/blup-release/release.pub
}

// PublicKeys returns the release keys built into this binary.
func PublicKeys() []ed25519.PublicKey {
	out := make([]ed25519.PublicKey, 0, len(publicKeys))
	for _, k := range publicKeys {
		b, err := base64.StdEncoding.DecodeString(k)
		if err == nil && len(b) == ed25519.PublicKeySize {
			out = append(out, ed25519.PublicKey(b))
		}
	}
	return out
}
