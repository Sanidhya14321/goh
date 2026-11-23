package session

import "github.com/Sanidhya14321/goh/internal/core"

// RedactSecrets masks well-known credential formats in text meant for
// sharing (transcripts, exports). It lives in core so model adapters can use
// it without depending on the session journal.
func RedactSecrets(text string) string { return core.RedactSecrets(text) }
