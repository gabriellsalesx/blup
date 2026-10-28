package cfpages

import (
	"encoding/base64"
	"encoding/hex"
	"mime"
	"strings"

	"lukechampine.com/blake3"
)

// HashFile returns the asset hash exactly like wrangler's hashFile
// (@cloudflare/deploy-helpers, src/deploy/helpers/hash.ts):
//
//	blake3(base64(contents) + extname(path).substring(1)).hex().slice(0, 32)
//
// extname follows Node's path.extname (".htaccess" has no extension; case is
// preserved).
func HashFile(name string, contents []byte) string {
	in := base64.StdEncoding.EncodeToString(contents) + strings.TrimPrefix(nodeExtname(name), ".")
	sum := blake3.Sum256([]byte(in))
	return hex.EncodeToString(sum[:])[:32]
}

// nodeExtname is a port of Node's posix path.extname.
func nodeExtname(p string) string {
	startDot, startPart, end := -1, 0, -1
	matchedSlash := true
	preDotState := 0
	for i := len(p) - 1; i >= 0; i-- {
		c := p[i]
		if c == '/' {
			if !matchedSlash {
				startPart = i + 1
				break
			}
			continue
		}
		if end == -1 {
			matchedSlash = false
			end = i + 1
		}
		if c == '.' {
			if startDot == -1 {
				startDot = i
			} else if preDotState != 1 {
				preDotState = 1
			}
		} else if startDot != -1 {
			preDotState = -1
		}
	}
	if startDot == -1 || end == -1 || preDotState == 0 ||
		(preDotState == 1 && startDot == end-1 && startDot == startPart+1) {
		return ""
	}
	return p[startDot:end]
}

// webTypes matches the npm "mime" v3 package (used by wrangler) for common
// static-site extensions.
var webTypes = map[string]string{
	"html": "text/html", "htm": "text/html", "css": "text/css",
	"js": "application/javascript", "mjs": "application/javascript",
	"json": "application/json", "map": "application/json", "jsonld": "application/ld+json",
	"xml": "application/xml", "xsl": "application/xml", "rss": "application/rss+xml",
	"atom": "application/atom+xml", "webmanifest": "application/manifest+json",
	"txt": "text/plain", "md": "text/markdown", "csv": "text/csv",
	"yaml": "text/yaml", "yml": "text/yaml", "toml": "application/toml",
	"svg": "image/svg+xml", "png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg",
	"gif": "image/gif", "webp": "image/webp", "avif": "image/avif", "ico": "image/vnd.microsoft.icon",
	"bmp": "image/bmp", "tif": "image/tiff", "tiff": "image/tiff", "heic": "image/heic",
	"woff": "font/woff", "woff2": "font/woff2", "ttf": "font/ttf", "otf": "font/otf",
	"eot": "application/vnd.ms-fontobject",
	"pdf": "application/pdf", "zip": "application/zip", "gz": "application/gzip",
	"epub": "application/epub+zip", "wasm": "application/wasm",
	"mp4": "video/mp4", "webm": "video/webm", "mov": "video/quicktime",
	"mp3": "audio/mpeg", "ogg": "audio/ogg", "wav": "audio/wav", "m4a": "audio/mp4",
}

// contentTypeFor mirrors wrangler pages validate.ts:
// getType(name) || "application/octet-stream" (no charset suffix).
func contentTypeFor(name string) string {
	ext := strings.ToLower(strings.TrimPrefix(nodeExtname("/"+name), "."))
	if ext == "" {
		return "application/octet-stream"
	}
	if t, ok := webTypes[ext]; ok {
		return t
	}
	if t := mime.TypeByExtension("." + ext); t != "" {
		if mt, _, err := mime.ParseMediaType(t); err == nil {
			return mt
		}
	}
	return "application/octet-stream"
}
