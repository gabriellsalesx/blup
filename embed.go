// Package blup embeds the Hugo theme shipped with the CLI.
package blup

import "embed"

// Theme is the "theme" folder, copied into every site as themes/blup.
//
//go:embed all:theme
var Theme embed.FS
