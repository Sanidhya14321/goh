package tui

import (
	"go/token"
	"os"
	"strings"
	"text/scanner"
)

// Highlighting is lexical. Text is always retained, including incomplete code
// arriving from a stream. Generated escapes are the only trusted styled text.
func highlightCode(code, language string) string {
	fields := strings.Fields(language)
	if len(fields) == 0 {
		return code
	}
	language = strings.ToLower(fields[0])
	switch language {
	case "go", "golang", "json", "jsonc", "js", "javascript", "ts", "typescript", "jsx", "tsx":
	default:
		return code
	}
	var scan scanner.Scanner
	scan.Init(strings.NewReader(code))
	scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats |
		scanner.ScanChars | scanner.ScanStrings | scanner.ScanRawStrings | scanner.ScanComments
	scan.Error = func(*scanner.Scanner, string) {}
	var out strings.Builder
	colored := os.Getenv("NO_COLOR") == ""
	end := 0
	for kind := scan.Scan(); kind != scanner.EOF; kind = scan.Scan() {
		text, start := scan.TokenText(), scan.Position.Offset
		out.WriteString(code[end:start])
		color := foreground
		switch kind {
		case scanner.String, scanner.RawString, scanner.Char:
			color = green
		case scanner.Int, scanner.Float:
			color = accent
		case scanner.Comment:
			color = muted
		case scanner.Ident:
			if (language == "go" || language == "golang") && token.Lookup(text).IsKeyword() {
				color = blue
			} else if strings.Contains(" true false null nil undefined ", " "+text+" ") {
				color = accent
			} else if language != "json" && language != "jsonc" &&
				strings.Contains(" const let var function return if else for while class interface type import export from async await new try catch throw switch case default break continue extends implements public private readonly static this ", " "+text+" ") {
				color = blue
			}
		}
		// Each physical line has independent styling so clipping a viewport
		// cannot inherit the previous row's terminal attributes.
		parts := strings.Split(text, "\n")
		for i, part := range parts {
			if i > 0 {
				out.WriteByte('\n')
			}
			if colored && color != foreground && part != "" {
				out.WriteString("\x1b[38;2;" + color + "m")
				out.WriteString(part)
				out.WriteString("\x1b[38;2;" + foreground + "m")
			} else {
				out.WriteString(part)
			}
		}
		end = start + len(text)
	}
	out.WriteString(code[end:])
	return out.String()
}
