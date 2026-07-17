package handlebars

import "strings"

// htmlReplacer mirrors the Handlebars.js HTML escaping table.
var htmlReplacer = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
	"'", "&#x27;",
	"`", "&#x60;",
	"=", "&#x3D;",
)

// EscapeHTML escapes a string using the same character set as Handlebars.js.
func EscapeHTML(s string) string {
	return htmlReplacer.Replace(s)
}
