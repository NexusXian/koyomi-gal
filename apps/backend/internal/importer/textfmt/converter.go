// Package textfmt normalizes free-text formats returned by external game
// sources into the site's canonical Markdown. The database only ever stores
// Markdown; each external source gets its own converter and conversion runs
// exactly once at import time.
package textfmt

// Converter transforms external source text into Markdown. Implementations
// must be safe for concurrent use.
type Converter interface {
	Convert(input string) string
}
