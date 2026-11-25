// Package web embeds templates, static files (CSS/JS/fonts) and the shared
// Schatzogram brand copies into the binary.
//
// brand/ and static/brand/ are copies of ../shared/schatzogram-brand; run
// `make sync-brand` after changing the shared files (a test enforces it).
package web

import "embed"

// FS holds templates/, static/ and brand/.
//
//go:embed templates static brand
var FS embed.FS
