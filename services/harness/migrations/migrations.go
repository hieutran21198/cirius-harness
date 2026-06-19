// Package migrations embeds the harness SQL migrations for goose.
package migrations

import "embed"

// FS holds the embedded goose migration files, ordered by version prefix.
//
//go:embed *.sql
var FS embed.FS
