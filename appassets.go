package appassets

import "embed"

// Dist bundles the built frontend so the Wails shell can serve it directly.
//
//go:embed all:frontend/dist
var Dist embed.FS

//go:embed wails.json
var WailsConfig []byte
