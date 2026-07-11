package hades

import "strings"

// EscapeIdentifier returns identifier double-quoted for interpolation into
// SQL, with embedded quotes doubled.
func EscapeIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
