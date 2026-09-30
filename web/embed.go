// Package webembed exposes the built frontend assets (web/dist produced by
// `npm run build`) for embedding into the single Go binary.
package webembed

import "embed"

// DistFS holds the production frontend. The directory always contains at
// least dist/.gitkeep so the pattern matches even before the first build.
//
//go:embed all:dist
var DistFS embed.FS
