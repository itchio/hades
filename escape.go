package hades

import "strings"

// EscapeIdentifier returns identifier double-quoted for safe interpolation
// into SQL, doubling any embedded quotes. Quoting unconditionally means
// keywords, punctuation, and identifiers taken from the database itself
// (e.g. index names in sqlite_master) are all handled.
func EscapeIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}
