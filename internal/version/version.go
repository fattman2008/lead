package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// String returns the package version from VERSION.
func String() string {
	return strings.TrimSpace(raw)
}
