package session

import "regexp"

// secretPatterns match credential formats with distinctive prefixes, so
// ordinary text is not masked. The prefix is kept so a reader can tell what
// kind of value was removed.
var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\b(sk-(?:ant-|proj-)?)[A-Za-z0-9_\-]{20,}`),
	regexp.MustCompile(`\b(gh[pousr]_)[A-Za-z0-9]{30,}`),
	regexp.MustCompile(`\b(github_pat_)[A-Za-z0-9_]{30,}`),
	regexp.MustCompile(`\b(xox[abprs]-)[A-Za-z0-9\-]{10,}`),
	regexp.MustCompile(`\b(AKIA)[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\b(AIza)[0-9A-Za-z_\-]{35}`),
	regexp.MustCompile(`(?i)\b(Bearer\s+)[A-Za-z0-9._~+/\-]{20,}=*`),
}

var privateKeyBlock = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?(?:-----END [A-Z ]*PRIVATE KEY-----|$)`)

// RedactSecrets masks well-known credential formats in text meant for
// sharing. It is a safety net, not a guarantee: secrets without a
// recognisable shape pass through.
func RedactSecrets(text string) string {
	text = privateKeyBlock.ReplaceAllString(text, "[redacted private key]")
	for _, p := range secretPatterns {
		text = p.ReplaceAllString(text, "${1}[redacted]")
	}
	return text
}
