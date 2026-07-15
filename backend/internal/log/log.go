// Package log centralises process logging setup. Kept tiny on purpose: it
// only installs the default slog handler so the rest of the code uses the
// stdlib slog package directly.
package log

import (
	"log/slog"
	"os"
)

// Setup installs a JSON slog handler as the process default.
func Setup() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
}
