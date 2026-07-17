package handlebars

import (
	"fmt"
	"strconv"
	"strings"
)

// Parse compiles a template source string into a *Template.
func parse(src string) (*Program, error) {
	toks, err := newLexer(src).lex()
	if err != nil {
		return nil, err
	}
	p := &parser{toks: toks}
	prog, stop, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if stop != nil {
		return nil, fmt.Errorf("handlebars: unexpected {{/%s}} or {{else}}", stop.text)
	}
	return prog, nil
}

// parser turns the flat token stream into a tree of Nodes.
type parser struct {
	toks []token
	i    int
}

func (p *parser) more() bool  { return p.i < len(p.toks) }
func (p *parser) peek() token { return p.toks[p.i] }
func (p *parser) next() token { t := p.toks[p.i]; p.i++; return t }

// parseBody parses nodes until end of input, a block close, or a bare else.
// The stopping token (if any) is returned so callers can act on it.
func (p *parser) parseBody() (*Program, *token, error) {
	prog := &Program{}
	for p.more() {
		t := p.peek()
		if t.kind == tContent {
			p.next()
			if t.text != "" {
				prog.Body = append(prog.Body, &ContentNode{Value: t.text})
			}
			continue
		}
		switch t.mkind {
		case mkBlockClose:
			p.next() // consume the {{/name}}
			return prog, &t, nil
		case mkElse:
			p.next() // consume the {{else}} / {{else if ...}}
			return prog, &t, nil
		case mkInverse:
			if t.text == "" { // bare {{^}}
				p.next() // consume the {{^}}
				return prog, &t, nil
			}
			node, err := p.parseBlock(true)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, node)
		case mkBlockOpen:
			node, err := p.parseBlock(false)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, node)
		case mkPartialBlock:
			node, err := p.parsePartialBlock()
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, node)
		case mkDecoratorBlock:
			node, err := p.parseDecoratorBlock()
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, node)
		case mkDecorator:
			p.next()
			expr, err := parseMustacheExpr(t.text)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, &DecoratorNode{Expr: expr})
		case mkComment:
			p.next()
			prog.Body = append(prog.Body, &CommentNode{Value: t.text})
		case mkPartial:
			p.next()
			node, err := parsePartial(t)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, node)
		case mkUnescaped:
			p.next()
			expr, err := parseMustacheExpr(t.text)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, &MustacheNode{Expr: expr, Unescaped: true})
		default: // mkInterp
			p.next()
			if t.text == "" {
				continue
			}
			expr, err := parseMustacheExpr(t.text)
			if err != nil {
				return nil, nil, err
			}
			prog.Body = append(prog.Body, &MustacheNode{Expr: expr})
		}
	}
	return prog, nil, nil
}

// parseBlock parses a {{#name}}...{{/name}} (or {{^name}}...{{/name}}) block,
// including any {{else}} / {{else if}} chain.
func (p *parser) parseBlock(inverted bool) (Node, error) {
	open := p.next()
	return p.finishBlock(open.text, inverted)
}

// finishBlock parses a block body (headText is the opening expression, already
// consumed) followed by an optional inverse or {{else if}} chain and the closing
// tag. It is shared by {{#name}} openers and synthesised {{else if}} heads.
func (p *parser) finishBlock(headText string, inverted bool) (Node, error) {
	exprText, params := extractBlockParams(headText)
	expr, err := parseMustacheExpr(exprText)
	if err != nil {
		return nil, err
	}
	body, stop, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	block := &BlockNode{Expr: expr, Program: body, Inverted: inverted, Params: params}
	if stop == nil {
		return nil, fmt.Errorf("handlebars: unclosed block {{#%s}}", headText)
	}
	switch {
	case stop.mkind == mkElse && stop.text != "":
		// {{else if cond}} / {{else with x}}: a chained block that shares this
		// block's closing tag. It becomes the sole content of the inverse.
		chained, err := p.finishBlock(stop.text, false)
		if err != nil {
			return nil, err
		}
		block.Inverse = &Program{Body: []Node{chained}}
	case stop.mkind == mkElse || (stop.mkind == mkInverse && stop.text == ""):
		inv, stop2, err := p.parseBody()
		if err != nil {
			return nil, err
		}
		block.Inverse = inv
		if stop2 == nil || stop2.mkind != mkBlockClose {
			return nil, fmt.Errorf("handlebars: unclosed block {{#%s}}", headText)
		}
	case stop.mkind == mkBlockClose:
		// body fully closed
	default:
		return nil, fmt.Errorf("handlebars: unclosed block {{#%s}}", headText)
	}
	return block, nil
}

// parsePartialBlock parses {{#> name}}...{{/name}}, capturing the block body as
// the partial's @partial-block.
func (p *parser) parsePartialBlock() (Node, error) {
	open := p.next()
	node, err := parsePartial(open)
	if err != nil {
		return nil, err
	}
	pn := node.(*PartialNode)
	body, stop, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if stop == nil || stop.mkind != mkBlockClose {
		return nil, fmt.Errorf("handlebars: unclosed partial block {{#>%s}}", open.text)
	}
	pn.Program = body
	return pn, nil
}

// parseDecoratorBlock parses {{#*name args}}...{{/name}} such as the inline
// partial form {{#*inline "row"}}...{{/inline}}.
func (p *parser) parseDecoratorBlock() (Node, error) {
	open := p.next()
	expr, err := parseMustacheExpr(open.text)
	if err != nil {
		return nil, err
	}
	body, stop, err := p.parseBody()
	if err != nil {
		return nil, err
	}
	if stop == nil || stop.mkind != mkBlockClose {
		return nil, fmt.Errorf("handlebars: unclosed decorator block {{#*%s}}", open.text)
	}
	return &DecoratorNode{Expr: expr, Program: body}, nil
}

// extractBlockParams splits a trailing block-parameter clause "as |a b|" from a
// block's opening expression, returning the bare expression text and the
// declared parameter names.
func extractBlockParams(head string) (string, []string) {
	open := strings.LastIndex(head, "|")
	bar := strings.Index(head, "|")
	if bar < 0 || open == bar {
		return head, nil
	}
	// Require an "as" keyword immediately before the opening bar.
	before := strings.TrimRight(head[:bar], " \t")
	if !strings.HasSuffix(before, "as") ||
		(len(before) > 2 && !isSpace(before[len(before)-3])) {
		return head, nil
	}
	inner := head[bar+1 : open]
	names := strings.Fields(inner)
	exprText := strings.TrimSpace(before[:len(before)-2])
	return exprText, names
}

func isSpace(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// parsePartial builds a PartialNode from a {{> ...}} token.
func parsePartial(t token) (Node, error) {
	atoms, err := lexExpr(t.text)
	if err != nil {
		return nil, err
	}
	ep := &exprParser{atoms: atoms}
	if !ep.more() {
		return nil, fmt.Errorf("handlebars: empty partial name")
	}
	name := ep.parsePrimary()
	node := &PartialNode{Name: name, Indent: t.indent}
	for ep.more() {
		a := ep.peek()
		if a.kind == aPath && ep.peekAt(1).kind == aEquals {
			key := ep.next().val
			ep.next() // '='
			node.Hash = append(node.Hash, HashPair{Key: key, Value: ep.parsePrimary()})
			continue
		}
		node.Context = ep.parsePrimary()
	}
	return node, nil
}

// parseMustacheExpr lexes and parses the inner expression of a mustache.
func parseMustacheExpr(text string) (*Expr, error) {
	atoms, err := lexExpr(text)
	if err != nil {
		return nil, err
	}
	if len(atoms) == 0 {
		return nil, fmt.Errorf("handlebars: empty expression")
	}
	ep := &exprParser{atoms: atoms}
	return ep.parseCall(), nil
}

// ---- expression atom lexer ----

type atomKind int

const (
	aPath atomKind = iota
	aString
	aNumber
	aBool
	aLParen
	aRParen
	aEquals
)

type atom struct {
	kind atomKind
	val  string
	num  float64
	b    bool
}

// lexExpr splits a mustache's inner text into expression atoms.
func lexExpr(s string) ([]atom, error) {
	var atoms []atom
	i := 0
	for i < len(s) {
		c := s[i]
		switch c {
		case ' ', '\t', '\n', '\r':
			i++
		case '(':
			atoms = append(atoms, atom{kind: aLParen})
			i++
		case ')':
			atoms = append(atoms, atom{kind: aRParen})
			i++
		case '=':
			atoms = append(atoms, atom{kind: aEquals})
			i++
		case '"', '\'':
			j := i + 1
			var sb strings.Builder
			for j < len(s) && s[j] != c {
				if s[j] == '\\' && j+1 < len(s) {
					j++
				}
				sb.WriteByte(s[j])
				j++
			}
			if j >= len(s) {
				return nil, fmt.Errorf("handlebars: unterminated string in expression %q", s)
			}
			atoms = append(atoms, atom{kind: aString, val: sb.String()})
			i = j + 1
		default:
			j := i
			for j < len(s) {
				ch := s[j]
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
					ch == '(' || ch == ')' || ch == '=' {
					break
				}
				if ch == '[' { // bracketed segment may contain spaces
					k := strings.IndexByte(s[j:], ']')
					if k < 0 {
						j = len(s)
						break
					}
					j += k + 1
					continue
				}
				j++
			}
			word := s[i:j]
			atoms = append(atoms, classifyWord(word))
			i = j
		}
	}
	return atoms, nil
}

// classifyWord decides whether a bare word is a boolean, number or path atom.
func classifyWord(w string) atom {
	switch w {
	case "true":
		return atom{kind: aBool, b: true}
	case "false":
		return atom{kind: aBool, b: false}
	case "null", "undefined":
		return atom{kind: aPath, val: w} // resolves to nil at render time
	}
	if !strings.ContainsAny(w, ".[/") {
		if n, err := strconv.ParseFloat(w, 64); err == nil {
			return atom{kind: aNumber, num: n}
		}
	}
	return atom{kind: aPath, val: w}
}

// ---- expression parser ----

type exprParser struct {
	atoms []atom
	i     int
}

func (e *exprParser) more() bool { return e.i < len(e.atoms) }
func (e *exprParser) peek() atom { return e.atoms[e.i] }
func (e *exprParser) next() atom { a := e.atoms[e.i]; e.i++; return a }

func (e *exprParser) peekAt(n int) atom {
	if e.i+n < len(e.atoms) {
		return e.atoms[e.i+n]
	}
	return atom{kind: aEquals + 100} // sentinel that matches nothing
}

// parseCall reads a callee followed by positional params and hash arguments,
// stopping at a closing parenthesis or the end of input.
func (e *exprParser) parseCall() *Expr {
	callee := e.parsePrimary()
	for e.more() && e.peek().kind != aRParen {
		if e.peek().kind == aPath && e.peekAt(1).kind == aEquals {
			key := e.next().val
			e.next() // '='
			callee.Hash = append(callee.Hash, HashPair{Key: key, Value: e.parsePrimary()})
			continue
		}
		callee.Params = append(callee.Params, e.parsePrimary())
	}
	return callee
}

// parsePrimary reads a single literal, path, or parenthesised subexpression.
func (e *exprParser) parsePrimary() *Expr {
	if !e.more() {
		return &Expr{Kind: exprPath, Path: parsePath("")}
	}
	a := e.peek()
	switch a.kind {
	case aLParen:
		e.next()
		sub := e.parseCall()
		if e.more() && e.peek().kind == aRParen {
			e.next()
		}
		sub.Kind = exprSubexpr
		return sub
	case aString:
		e.next()
		return &Expr{Kind: exprString, Str: a.val}
	case aNumber:
		e.next()
		return &Expr{Kind: exprNumber, Num: a.num}
	case aBool:
		e.next()
		return &Expr{Kind: exprBool, Bool: a.b}
	default: // aPath
		e.next()
		return &Expr{Kind: exprPath, Path: parsePath(a.val)}
	}
}

// parsePath parses a dotted path such as foo.bar.0, ../name, this or @index.
func parsePath(s string) *Path {
	p := &Path{Original: s}
	rest := s
	if strings.HasPrefix(rest, "@") {
		p.Data = true
		rest = rest[1:]
	}
	for strings.HasPrefix(rest, "../") {
		p.Depth++
		rest = rest[3:]
	}
	rest = strings.TrimPrefix(rest, "./")
	if rest == "" || rest == "this" || rest == "." {
		p.This = true
		return p
	}
	segs := splitPath(rest)
	if len(segs) > 0 && segs[0] == "this" {
		segs = segs[1:]
	}
	p.Segments = segs
	return p
}

// splitPath breaks a path body into dotted segments, honouring [bracket]
// segments that may contain otherwise-special characters.
func splitPath(s string) []string {
	var segs []string
	var cur strings.Builder
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '[' {
			k := strings.IndexByte(s[i:], ']')
			if k < 0 {
				cur.WriteString(s[i+1:])
				break
			}
			cur.WriteString(s[i+1 : i+k])
			i += k + 1
			continue
		}
		if c == '.' {
			segs = append(segs, cur.String())
			cur.Reset()
			i++
			continue
		}
		cur.WriteByte(c)
		i++
	}
	segs = append(segs, cur.String())
	out := segs[:0]
	for _, s := range segs {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
