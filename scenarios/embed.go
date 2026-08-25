package scenarios

import "embed"

// Bundled ships with the game binary.
//
//go:embed *.json
var Bundled embed.FS
