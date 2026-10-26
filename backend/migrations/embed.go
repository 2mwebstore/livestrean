// Package migrations holds the versioned SQL schema. Files are applied in
// filename order at startup; never edit a file that has already shipped,
// add a new one instead.
package migrations

import "embed"

//go:embed *.sql
var Files embed.FS
