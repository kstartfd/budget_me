package migrations

import "embed"

// FS holds the embedded SQL migration files, consumed by cmd/migrate (goose).
//
//go:embed *.sql
var FS embed.FS
