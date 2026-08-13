// Package migrations exposes the ordered PostgreSQL schema transitions owned by the API.
package migrations

import "embed"

// Files contains every migration embedded into the API binaries.
//
//go:embed *.sql
var Files embed.FS
