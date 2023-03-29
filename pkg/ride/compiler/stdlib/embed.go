package stdlib

import "embed"

//go:embed funcs.json
//go:embed ride_objects.json
//go:embed vars.json
var embedFS embed.FS
