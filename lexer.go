package handlebars

import (
	"fmt"
	"strings"
)

// tokKind is the top level classification of a template token.
type tokKind int

const (
	tContent tokKind = iota
	tMustache
)

// mKind classifies a mustache token by its opening sigil.
type mKind int

const (
	mkInterp         mKind = iota // {{ expr }}
	mkUnescaped                   // {{{ expr }}} or {{& expr }}
	mkComment                     // {{! }} or {{!-- --}}
	mkBlockOpen                   // {{# expr }}
	mkInverse                     // {{^ expr }} (or bare {{^}})
	mkElse                        // {{else}} or {{else if cond}} chain separator
	mkBlockClose                  // {{/ name }}
	mkPartial                     // {{> name }}
	mkPartialBlock                // {{#> name }} ... {{/name}}
	mkDecorator                   // {{* name }}
	mkDecoratorBlock              // {{#*name }} ... {{/name}}
)

// token is a lexical unit produced by the outer lexer.
type token struct {
	kind       tokKind
	mkind      mKind
	text       string // content text, or the inner expression text of a mustache
	trimLeft   bool   // {{~
	trimRight  bool   // ~}}
	standalone bool   // occupies its own line (for whitespace control)
	indent     string // captured indentation for standalone partials
	line       int
}

// lexer scans a template source into a flat slice of tokens.
type lexer struct {
	src  string
	pos  int
	line int
}

func newLexer(src string) *lexer { return &lexer{src: src, line: 1} }

func (l *lexer) errorf(format string, args ...interface{}) error {
	return fmt.Errorf("handlebars: line %d: %s", l.line, fmt.Sprintf(format, args...))
}

// lex returns the full token stream for the source.
func (l *lexer) lex() ([]token, error) {
	var toks []token
	for l.pos < len(l.src) {
		open := strings.Index(l.src[l.pos:], "{{")
		if open < 0 {
			// Remaining source is plain content.
			text := l.src[l.pos:]
			toks = append(toks, token{kind: tContent, text: text, line: l.line})
			l.advance(len(text))
			break
		}
		before := l.src[l.pos : l.pos+open]
		// Count backslashes immediately preceding the "{{". An escaped mustache
		// (odd count) is emitted verbatim as content; a doubled backslash (even
		// count) yields one literal backslash and a live mustache. This mirrors
		// the Handlebars.js "\{{foo}}" / "\\{{foo}}" escaping rules.
		bs := 0
		for bs < len(before) && before[len(before)-1-bs] == '\\' {
			bs++
		}
		startLine := l.line
		if bs > 0 {
			content := before[:len(before)-bs] + strings.Repeat("\\", bs/2)
			if content != "" {
				toks = append(toks, token{kind: tContent, text: content, line: startLine})
			}
			l.advance(open)
			if bs%2 == 1 {
				raw, n := l.rawMustache()
				toks = append(toks, token{kind: tContent, text: raw, line: l.line})
				l.advance(n)
				continue
			}
			tok, err := l.lexMustache()
			if err != nil {
				return nil, err
			}
			toks = append(toks, tok)
			continue
		}
		if open > 0 {
			toks = append(toks, token{kind: tContent, text: before, line: startLine})
			l.advance(open)
		}
		tok, err := l.lexMustache()
		if err != nil {
			return nil, err
		}
		toks = append(toks, tok)
	}
	markStandalone(toks)
	applyWhitespaceControl(toks)
	return toks, nil
}

// applyWhitespaceControl mutates content tokens to honour explicit ~ trims and
// the standalone-line rule computed by markStandalone.
func applyWhitespaceControl(toks []token) {
	for i := range toks {
		t := &toks[i]
		if t.kind != tMustache {
			continue
		}
		if t.trimLeft {
			trimContentRight(toks, i)
		}
		if t.trimRight {
			trimContentLeft(toks, i)
		}
		if t.standalone {
			// An explicit ~ trim on a side supersedes the standalone rule for
			// that side; applying both would let stripStandaloneLeft mistake
			// already-trimmed content for indentation.
			if !t.trimLeft {
				t.indent = stripStandaloneLeft(toks, i)
			}
			if !t.trimRight {
				stripStandaloneRight(toks, i)
			}
		}
	}
}

// trimContentRight removes all trailing whitespace from the content token
// preceding index i.
func trimContentRight(toks []token, i int) {
	if i == 0 || toks[i-1].kind != tContent {
		return
	}
	toks[i-1].text = strings.TrimRight(toks[i-1].text, " \t\r\n")
}

// trimContentLeft removes all leading whitespace from the content token
// following index i.
func trimContentLeft(toks []token, i int) {
	if i == len(toks)-1 || toks[i+1].kind != tContent {
		return
	}
	toks[i+1].text = strings.TrimLeft(toks[i+1].text, " \t\r\n")
}

// stripStandaloneLeft removes the indentation on the standalone tag's line from
// the preceding content token and returns the removed indentation.
func stripStandaloneLeft(toks []token, i int) string {
	if i == 0 || toks[i-1].kind != tContent {
		return ""
	}
	text := toks[i-1].text
	idx := strings.LastIndexByte(text, '\n')
	indent := text[idx+1:]
	toks[i-1].text = text[:idx+1]
	return indent
}

// stripStandaloneRight removes leading whitespace and the terminating newline
// from the content token following a standalone tag.
func stripStandaloneRight(toks []token, i int) {
	if i == len(toks)-1 || toks[i+1].kind != tContent {
		return
	}
	text := toks[i+1].text
	j := 0
	for j < len(text) && (text[j] == ' ' || text[j] == '\t') {
		j++
	}
	if j < len(text) && text[j] == '\r' {
		j++
	}
	if j < len(text) && text[j] == '\n' {
		j++
		toks[i+1].text = text[j:]
		return
	}
	// A standalone tag whose line ends at the template's end (no terminating
	// newline) still consumes the trailing whitespace, matching Handlebars.
	if j == len(text) && i+1 == len(toks)-1 {
		toks[i+1].text = ""
	}
}

// advance moves the cursor forward n bytes, tracking line numbers.
func (l *lexer) advance(n int) {
	for i := 0; i < n; i++ {
		if l.src[l.pos] == '\n' {
			l.line++
		}
		l.pos++
	}
}

// rawMustache returns the verbatim text of the mustache beginning at the cursor
// (including its delimiters) and its byte length. It is used to emit an escaped
// "\{{...}}" mustache as literal content. A triple "{{{...}}}" is closed by
// "}}}"; every other form is closed by "}}".
func (l *lexer) rawMustache() (string, int) {
	s := l.src[l.pos:]
	closing := "}}"
	if strings.HasPrefix(s, "{{{") {
		closing = "}}}"
	}
	idx := strings.Index(s, closing)
	if idx < 0 {
		return s, len(s)
	}
	end := idx + len(closing)
	return s[:end], end
}

// lexMustache consumes a single {{ ... }} construct starting at the cursor.
func (l *lexer) lexMustache() (token, error) {
	startLine := l.line
	l.advance(2) // consume {{

	trimLeft := false
	if l.peek() == '~' {
		trimLeft = true
		l.advance(1)
	}

	// Unescaped triple-stache {{{ expr }}} and its whitespace-control variant
	// {{~{ expr }~}}. The inner braces select raw (non-HTML-escaped) output. This
	// is handled after the leading ~ so the tilde form is recognised too.
	if l.peek() == '{' {
		l.advance(1)
		inner, trimRight, err := l.readUnescaped()
		if err != nil {
			return token{}, err
		}
		return token{kind: tMustache, mkind: mkUnescaped, text: strings.TrimSpace(inner),
			trimLeft: trimLeft, trimRight: trimRight, line: startLine}, nil
	}

	// Comments have their own delimiters.
	if l.peek() == '!' {
		l.advance(1)
		if strings.HasPrefix(l.src[l.pos:], "--") {
			l.advance(2)
			inner, trimRight, err := l.readLongComment()
			if err != nil {
				return token{}, err
			}
			return token{kind: tMustache, mkind: mkComment, text: strings.TrimSpace(inner),
				trimLeft: trimLeft, trimRight: trimRight, line: startLine}, nil
		}
		inner, err := l.readCommentUntilClose()
		if err != nil {
			return token{}, err
		}
		trimRight, body := trailingTilde(inner)
		return token{kind: tMustache, mkind: mkComment, text: strings.TrimSpace(body),
			trimLeft: trimLeft, trimRight: trimRight, line: startLine}, nil
	}

	kind := mkInterp
	switch l.peek() {
	case '#':
		l.advance(1)
		switch l.peek() {
		case '*':
			kind = mkDecoratorBlock
			l.advance(1)
		case '>':
			kind = mkPartialBlock
			l.advance(1)
		default:
			kind = mkBlockOpen
		}
	case '/':
		kind = mkBlockClose
		l.advance(1)
	case '^':
		kind = mkInverse
		l.advance(1)
	case '>':
		kind = mkPartial
		l.advance(1)
	case '&':
		kind = mkUnescaped
		l.advance(1)
	case '*':
		kind = mkDecorator
		l.advance(1)
	}

	inner, err := l.readUntil("}}")
	if err != nil {
		return token{}, err
	}
	trimRight, body := trailingTilde(inner)
	body = strings.TrimSpace(body)

	// A bare {{else}} or {{else if cond}} chain separator is normalised to an
	// mkElse marker for the parser. The remainder (e.g. "if cond") is retained.
	if kind == mkInterp && (body == "else" || strings.HasPrefix(body, "else ")) {
		kind = mkElse
		body = strings.TrimSpace(strings.TrimPrefix(body, "else"))
	}

	return token{kind: tMustache, mkind: kind, text: body,
		trimLeft: trimLeft, trimRight: trimRight, line: startLine}, nil
}

func (l *lexer) peek() byte {
	if l.pos < len(l.src) {
		return l.src[l.pos]
	}
	return 0
}

// readUntil reads until the given closing delimiter, honouring quoted strings
// so a literal like "}}" inside an argument does not close the mustache early.
func (l *lexer) readUntil(closing string) (string, error) {
	start := l.pos
	var quote byte
	for l.pos < len(l.src) {
		c := l.src[l.pos]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			l.advance(1)
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			l.advance(1)
			continue
		}
		if strings.HasPrefix(l.src[l.pos:], closing) {
			inner := l.src[start:l.pos]
			l.advance(len(closing))
			return inner, nil
		}
		l.advance(1)
	}
	return "", l.errorf("unclosed %q", closing)
}

// readUnescaped reads the body of an unescaped {{{ ... }}} mustache after its
// inner "{" has been consumed. The body ends at the first "}" that is followed
// by "}}" (or "~}}" for the whitespace-control form). The returned bool reports
// whether that trailing "~" trim marker was present.
func (l *lexer) readUnescaped() (string, bool, error) {
	start := l.pos
	for l.pos < len(l.src) {
		if l.src[l.pos] == '}' {
			rest := l.src[l.pos+1:]
			if strings.HasPrefix(rest, "~}}") {
				inner := l.src[start:l.pos]
				l.advance(4)
				return inner, true, nil
			}
			if strings.HasPrefix(rest, "}}") {
				inner := l.src[start:l.pos]
				l.advance(3)
				return inner, false, nil
			}
		}
		l.advance(1)
	}
	return "", false, l.errorf("unclosed %q", "}}}")
}

// readLongComment reads the body of a {{!-- ... --}} comment after the leading
// "!--" has been consumed. It closes on "--}}" or, for the whitespace-control
// form, "--~}}"; the returned bool reports whether the "~" trim marker was
// present. Long comments may contain "}}" so a dedicated terminator is required.
func (l *lexer) readLongComment() (string, bool, error) {
	start := l.pos
	for l.pos < len(l.src) {
		if strings.HasPrefix(l.src[l.pos:], "--~}}") {
			inner := l.src[start:l.pos]
			l.advance(5)
			return inner, true, nil
		}
		if strings.HasPrefix(l.src[l.pos:], "--}}") {
			inner := l.src[start:l.pos]
			l.advance(4)
			return inner, false, nil
		}
		l.advance(1)
	}
	return "", false, l.errorf("unclosed %q", "--}}")
}

// readCommentUntilClose reads a short comment terminated by the first "}}".
func (l *lexer) readCommentUntilClose() (string, error) {
	start := l.pos
	for l.pos < len(l.src) {
		if strings.HasPrefix(l.src[l.pos:], "}}") {
			inner := l.src[start:l.pos]
			l.advance(2)
			return inner, nil
		}
		l.advance(1)
	}
	return "", l.errorf("unclosed comment")
}

// trailingTilde reports whether the inner text ended with a ~ trim marker and
// returns the text with that marker removed.
func trailingTilde(inner string) (bool, string) {
	trimmed := strings.TrimRight(inner, " \t")
	if strings.HasSuffix(trimmed, "~") {
		return true, trimmed[:len(trimmed)-1]
	}
	return false, inner
}

// markStandalone flags block, comment and partial mustaches that sit alone on
// their own line so the renderer can strip the surrounding whitespace, matching
// the Mustache/Handlebars standalone rule.
func markStandalone(toks []token) {
	for i, t := range toks {
		if t.kind != tMustache {
			continue
		}
		switch t.mkind {
		case mkBlockOpen, mkBlockClose, mkInverse, mkElse, mkComment,
			mkPartial, mkPartialBlock, mkDecorator, mkDecoratorBlock:
			// eligible
		default:
			continue
		}
		if standaloneLeft(toks, i) && standaloneRight(toks, i) {
			toks[i].standalone = true
		}
	}
}

// standaloneLeft reports whether only whitespace precedes the token back to the
// previous newline or the start of the template.
func standaloneLeft(toks []token, i int) bool {
	if i == 0 {
		return true
	}
	prev := toks[i-1]
	if prev.kind != tContent {
		return false
	}
	idx := strings.LastIndexByte(prev.text, '\n')
	if idx >= 0 {
		return strings.TrimLeft(prev.text[idx+1:], " \t") == ""
	}
	// No newline in the preceding content: it is a line start only if it is
	// entirely whitespace and there is no earlier token on the line.
	if strings.TrimLeft(prev.text, " \t") != "" {
		return false
	}
	return i-1 == 0
}

// standaloneRight reports whether only whitespace follows the token up to and
// including the next newline or the end of the template.
func standaloneRight(toks []token, i int) bool {
	if i == len(toks)-1 {
		return true
	}
	next := toks[i+1]
	if next.kind != tContent {
		return false
	}
	idx := strings.IndexByte(next.text, '\n')
	if idx >= 0 {
		return strings.TrimLeft(next.text[:idx], " \t") == ""
	}
	// No newline in the following content: standalone only if it is entirely
	// whitespace and is the final token.
	if strings.TrimLeft(next.text, " \t") != "" {
		return false
	}
	return i+1 == len(toks)-1
}
