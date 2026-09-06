// Package studioembed holds the exact-version AhdDataStudio sources shipped
// inside the AhdCode CLI. `ahdcode databases` materializes this tree; it
// does not look for a source checkout.
package studioembed

import "embed"

//go:embed all:AhdDataStudio
var Files embed.FS
