package handlebars

// This file encodes additional known-answer vectors taken directly from the
// upstream Handlebars.js test suite (handlebars-lang/handlebars.js, spec/*.js).
// It focuses on behaviours that were gaps in the Go port before this audit:
// "/" path separators, bracketed literal `this`, whitespace-control on long
// comments and the {{~{ ... }~}} raw form, empty-object #if/#with semantics, and
// a broader sweep of helper, subexpression and partial cases. Run with
// `go test -run Parity`.

import "testing"

// TestParityPathSeparators mirrors spec/basic.js "nested paths", "paths with
// hyphens", "literal paths", "this keyword in paths" and "zeros": Handlebars
// accepts "/" as a path separator interchangeably with ".", and a bracketed
// [this] is a literal property name rather than the current-context keyword.
func TestParityPathSeparators(t *testing.T) {
	parity(t, "slash-nested", `Goodbye {{alan/expression}} world!`,
		map[string]any{"alan": map[string]any{"expression": "beautiful"}}, "Goodbye beautiful world!")
	parity(t, "slash-empty-string", `Goodbye {{alan/expression}} world!`,
		map[string]any{"alan": map[string]any{"expression": ""}}, "Goodbye  world!")
	parity(t, "slash-zero", `num: {{num1/num2}}`,
		map[string]any{"num1": map[string]any{"num2": 0}}, "num: 0")
	parity(t, "slash-hyphen", `{{foo/foo-bar}}`,
		map[string]any{"foo": map[string]any{"foo-bar": "baz"}}, "baz")
	parity(t, "literal-path-at", `Goodbye {{[@alan]/expression}} world!`,
		map[string]any{"@alan": map[string]any{"expression": "beautiful"}}, "Goodbye beautiful world!")
	parity(t, "literal-path-space", `Goodbye {{[foo bar]/expression}} world!`,
		map[string]any{"foo bar": map[string]any{"expression": "beautiful"}}, "Goodbye beautiful world!")
	parity(t, "this-slash-complex", `{{#hellos}}{{this/text}}{{/hellos}}`,
		map[string]any{"hellos": []any{
			map[string]any{"text": "hello"}, map[string]any{"text": "Hello"}, map[string]any{"text": "HELLO"}}},
		"helloHelloHELLO")
	// A bracketed [this] is a literal key, not the current context.
	parity(t, "bracket-this", `{{[this]}}`, map[string]any{"this": "bar"}, "bar")
	parity(t, "bracket-this-nested", `{{text/[this]}}`,
		map[string]any{"text": map[string]any{"this": "bar"}}, "bar")
}

// TestParityCommentsTilde mirrors spec/basic.js "comments": both short {{! }}
// and long {{!-- --}} comments honour the ~ whitespace-control markers on either
// delimiter.
func TestParityCommentsTilde(t *testing.T) {
	parity(t, "short-both-tilde", `    {{~! comment ~}}      blah`, nil, "blah")
	parity(t, "long-both-tilde", `    {{~!-- long-comment --~}}      blah`, nil, "blah")
	parity(t, "short-right-tilde", `    {{! comment ~}}      blah`, nil, "    blah")
	parity(t, "long-right-tilde", `    {{!-- long-comment --~}}      blah`, nil, "    blah")
	parity(t, "short-left-tilde", `    {{~! comment}}      blah`, nil, "      blah")
	parity(t, "long-left-tilde", `    {{~!-- long-comment --}}      blah`, nil, "      blah")
}

// TestParityWhitespaceMustache mirrors spec/whitespace-control.js "should strip
// whitespace around mustache calls", including the {{~{foo}~}} raw form.
func TestParityWhitespaceMustache(t *testing.T) {
	h := map[string]any{"foo": "bar<"}
	parity(t, "ws-both", ` {{~foo~}} `, h, "bar&lt;")
	parity(t, "ws-left", ` {{~foo}} `, h, "bar&lt; ")
	parity(t, "ws-right", ` {{foo~}} `, h, " bar&lt;")
	parity(t, "ws-amp-both", ` {{~&foo~}} `, h, "bar<")
	parity(t, "ws-raw-both", ` {{~{foo}~}} `, h, "bar<")
	parity(t, "ws-newlines", "1\n{{foo~}} \n\n 23\n{{bar}}4", map[string]any{}, "1\n23\n4")
	parity(t, "ws-once", ` {{~foo~}} {{foo}} {{foo}} `, map[string]any{"foo": "bar"}, "barbar bar ")
}

// TestParityWhitespaceBlocks mirrors spec/whitespace-control.js block sections,
// covering simple, inverse and complex block whitespace stripping.
func TestParityWhitespaceBlocks(t *testing.T) {
	h := map[string]any{"foo": "bar<"}
	parity(t, "blk-both", ` {{~#if foo~}} bar {{~/if~}} `, h, "bar")
	parity(t, "blk-right", ` {{#if foo~}} bar {{/if~}} `, h, " bar ")
	parity(t, "blk-left", ` {{~#if foo}} bar {{~/if}} `, h, " bar ")
	parity(t, "blk-none", ` {{#if foo}} bar {{/if}} `, h, "  bar  ")
	parity(t, "blk-multiline", " \n\n{{~#if foo~}} \n\nbar \n\n{{~/if~}}\n\n ", h, "bar")
	parity(t, "blk-multiline-a", " a\n\n{{~#if foo~}} \n\nbar \n\n{{~/if~}}\n\na ", h, " abara ")
	parity(t, "inv-both", ` {{~^if foo~}} bar {{~/if~}} `, map[string]any{}, "bar")
	parity(t, "inv-right", ` {{^if foo~}} bar {{/if~}} `, map[string]any{}, " bar ")
	parity(t, "cplx-1", `{{#if foo~}} bar {{~^~}} baz {{~/if}}`, h, "bar")
	parity(t, "cplx-2", `{{#if foo~}} bar {{^~}} baz {{/if}}`, h, "bar ")
	parity(t, "cplx-3", `{{#if foo}} bar {{~^~}} baz {{~/if}}`, h, " bar")
	parity(t, "cplx-4", `{{#if foo}} bar {{^~}} baz {{/if}}`, h, " bar ")
	parity(t, "cplx-else", `{{#if foo~}} bar {{~else~}} baz {{~/if}}`, h, "bar")
	parity(t, "cplx-inv-1", `{{#if foo~}} bar {{~^~}} baz {{~/if}}`, map[string]any{}, "baz")
	parity(t, "cplx-inv-2", `{{#if foo}} bar {{~^~}} baz {{/if}}`, map[string]any{}, "baz ")
	parity(t, "cplx-inv-3", `{{#if foo~}} bar {{~^}} baz {{~/if}}`, map[string]any{}, " baz")
	parity(t, "cplx-inv-4", `{{#if foo~}} bar {{~^}} baz {{/if}}`, map[string]any{}, " baz ")
}

// TestParityWhitespacePartials mirrors spec/whitespace-control.js
// "should strip whitespace around partials".
func TestParityWhitespacePartials(t *testing.T) {
	setup := func(tp *Template) { _ = tp.RegisterPartial("dude", "bar") }
	parityWith(t, "pw-both", `foo {{~> dude~}} `, nil, "foobar", setup)
	parityWith(t, "pw-right", `foo {{> dude~}} `, nil, "foo bar", setup)
	parityWith(t, "pw-none", `foo {{> dude}} `, nil, "foo bar ", setup)
	parityWith(t, "pw-indent", "foo\n {{~> dude}} ", nil, "foobar", setup)
	parityWith(t, "pw-indent-keep", "foo\n {{> dude}} ", nil, "foo\n bar", setup)
}

// TestParityBlockEmptyObject mirrors spec/builtins.js #with/#if semantics: an
// empty object (map) is truthy (Handlebars Utils.isEmpty treats only nil, false,
// the empty string and empty arrays as empty), so #with over {} renders its body
// and can still reach @root.
func TestParityBlockEmptyObject(t *testing.T) {
	parity(t, "with-empty-object-body", `{{#with x}}IN{{/with}}`,
		map[string]any{"x": map[string]any{}}, "IN")
	parity(t, "with-empty-object-root", `{{#with x}}{{@root.foo}}{{/with}}`,
		map[string]any{"foo": "hello", "x": map[string]any{}}, "hello")
	parity(t, "if-empty-object", `{{#if x}}yes{{else}}no{{/if}}`,
		map[string]any{"x": map[string]any{}}, "yes")
	parity(t, "if-empty-array", `{{#if x}}yes{{else}}no{{/if}}`,
		map[string]any{"x": []any{}}, "no")
	parity(t, "unless-empty-object", `{{#unless x}}yes{{else}}no{{/unless}}`,
		map[string]any{"x": map[string]any{}}, "no")
}

// TestParityEachAdvanced mirrors spec/builtins.js #each cases: nested @index,
// block params, @first / @last and object iteration (sorted keys in this port).
func TestParityEachAdvanced(t *testing.T) {
	goodbyes := map[string]any{"goodbyes": []any{
		map[string]any{"text": "goodbye"}, map[string]any{"text": "Goodbye"}, map[string]any{"text": "GOODBYE"}},
		"world": "world"}
	parity(t, "each-index", `{{#each goodbyes}}{{@index}}. {{text}}! {{/each}}cruel {{world}}!`, goodbyes,
		"0. goodbye! 1. Goodbye! 2. GOODBYE! cruel world!")
	parity(t, "each-nested-index",
		`{{#each goodbyes}}{{@index}}. {{text}}! {{#each ../goodbyes}}{{@index}} {{/each}}After {{@index}} {{/each}}{{@index}}cruel {{world}}!`,
		goodbyes,
		"0. goodbye! 0 1 2 After 0 1. Goodbye! 0 1 2 After 1 2. GOODBYE! 0 1 2 After 2 cruel world!")
	parity(t, "each-first", `{{#each goodbyes}}{{#if @first}}{{text}}! {{/if}}{{/each}}cruel {{world}}!`, goodbyes,
		"goodbye! cruel world!")
	parity(t, "each-last", `{{#each goodbyes}}{{#if @last}}{{text}}! {{/if}}{{/each}}cruel {{world}}!`, goodbyes,
		"GOODBYE! cruel world!")
	parity(t, "each-block-params",
		`{{#each goodbyes as |value index|}}{{index}}. {{value.text}}!{{/each}}`,
		map[string]any{"goodbyes": []any{map[string]any{"text": "goodbye"}, map[string]any{"text": "Goodbye"}}},
		"0. goodbye!1. Goodbye!")
	// Object iteration visits keys in sorted order in this port; @key is exposed.
	parity(t, "each-object-key", `{{#each m}}{{@key}}={{.}} {{/each}}`,
		map[string]any{"m": map[string]any{"b": 2, "a": 1, "c": 3}}, "a=1 b=2 c=3 ")
	parity(t, "each-empty-object-inverse", `{{#each m}}{{.}}{{else}}none{{/each}}`,
		map[string]any{"m": map[string]any{}}, "none")
}

// TestParityHelperLiterals mirrors spec/helpers.js literal-parameter cases:
// string, decimal, negative-number and boolean literals passed to helpers.
func TestParityHelperLiterals(t *testing.T) {
	parityWith(t, "decimal-literals", `Message: {{hello -1.2 1.2}}`, nil, "Message: Hello -1.2 1.2 times",
		func(tp *Template) {
			tp.RegisterHelper("hello", func(o *Options) any {
				return "Hello " + formatValue(o.Arg(0)) + " " + formatValue(o.Arg(1)) + " times"
			})
		})
	parityWith(t, "negative-literal", `Message: {{hello -12}}`, nil, "Message: Hello -12 times",
		func(tp *Template) {
			tp.RegisterHelper("hello", func(o *Options) any { return "Hello " + formatValue(o.Arg(0)) + " times" })
		})
	parityWith(t, "simple-literals", `Message: {{hello "world" 12 true false}}`, nil,
		"Message: Hello world 12 times: true false", func(tp *Template) {
			tp.RegisterHelper("hello", func(o *Options) any {
				return "Hello " + formatValue(o.Arg(0)) + " " + formatValue(o.Arg(1)) +
					" times: " + formatValue(o.Arg(2)) + " " + formatValue(o.Arg(3))
			})
		})
	parityWith(t, "string-escape", `Message: {{{hello "\"world\""}}}`, nil, `Message: Hello "world"`,
		func(tp *Template) {
			tp.RegisterHelper("hello", func(o *Options) any { return "Hello " + formatValue(o.Arg(0)) })
		})
	parityWith(t, "string-apostrophe", `Message: {{{hello "Alan's world"}}}`, nil, "Message: Hello Alan's world",
		func(tp *Template) {
			tp.RegisterHelper("hello", func(o *Options) any { return "Hello " + formatValue(o.Arg(0)) })
		})
	parityWith(t, "multi-params", `Message: {{goodbye cruel world}}`,
		map[string]any{"cruel": "cruel", "world": "world"}, "Message: Goodbye cruel world",
		func(tp *Template) {
			tp.RegisterHelper("goodbye", func(o *Options) any {
				return "Goodbye " + formatValue(o.Arg(0)) + " " + formatValue(o.Arg(1))
			})
		})
}

// TestParityHelperHash mirrors spec/helpers.js "hash" cases: helpers receive
// key=value hash arguments, including boolean values.
func TestParityHelperHash(t *testing.T) {
	parityWith(t, "optional-hash", `{{goodbye cruel="CRUEL" world="WORLD" times=12}}`, nil,
		"GOODBYE CRUEL WORLD 12 TIMES", func(tp *Template) {
			tp.RegisterHelper("goodbye", func(o *Options) any {
				return "GOODBYE " + o.HashStr("cruel", "") + " " + o.HashStr("world", "") + " " +
					o.HashStr("times", "") + " TIMES"
			})
		})
	goodbye := func(tp *Template) {
		tp.RegisterHelper("goodbye", func(o *Options) any {
			switch o.Hash["print"] {
			case true:
				return "GOODBYE " + o.HashStr("cruel", "") + " " + o.HashStr("world", "")
			case false:
				return "NOT PRINTING"
			default:
				return "THIS SHOULD NOT HAPPEN"
			}
		})
	}
	parityWith(t, "hash-bool-true", `{{goodbye cruel="CRUEL" world="WORLD" print=true}}`, nil,
		"GOODBYE CRUEL WORLD", goodbye)
	parityWith(t, "hash-bool-false", `{{goodbye cruel="CRUEL" world="WORLD" print=false}}`, nil,
		"NOT PRINTING", goodbye)
	parityWith(t, "block-hash", `{{#goodbye cruel="CRUEL" times=12}}world{{/goodbye}}`, nil,
		"GOODBYE CRUEL world 12 TIMES", func(tp *Template) {
			tp.RegisterHelper("goodbye", func(o *Options) any {
				return "GOODBYE " + o.HashStr("cruel", "") + " " + o.Fn(nil) + " " + o.HashStr("times", "") + " TIMES"
			})
		})
}

// TestParityHelperMissing mirrors spec/helpers.js "helperMissing": a custom
// helperMissing hook receives options.name and the evaluated arguments.
func TestParityHelperMissing(t *testing.T) {
	parityWith(t, "missing-value", `{{hello}} {{link_to world}}`,
		map[string]any{"hello": "Hello", "world": "world"}, "Hello <a>world</a>",
		func(tp *Template) {
			tp.RegisterHelper("helperMissing", func(o *Options) any {
				if o.Name == "link_to" {
					return SafeString("<a>" + formatValue(o.Arg(0)) + "</a>")
				}
				return nil
			})
		})
	parityWith(t, "missing-no-args", `{{hello}} {{link_to}}`,
		map[string]any{"hello": "Hello", "world": "world"}, "Hello <a>winning</a>",
		func(tp *Template) {
			tp.RegisterHelper("helperMissing", func(o *Options) any {
				if o.Name == "link_to" {
					return SafeString("<a>winning</a>")
				}
				return nil
			})
		})
}

// TestParitySubexprHash mirrors spec/subexpressions.js "with hashes" and the
// "multiple subexpressions in a hash" input-building case.
func TestParitySubexprHash(t *testing.T) {
	parityWith(t, "subexpr-with-hash", `{{blog (equal (equal true true) true fun='yes')}}`, nil,
		"val is true", func(tp *Template) {
			tp.RegisterHelper("blog", func(o *Options) any { return "val is " + formatValue(o.Arg(0)) })
			tp.RegisterHelper("equal", func(o *Options) any { return o.Arg(0) == o.Arg(1) })
		})
	parityWith(t, "subexpr-input",
		`{{input aria-label=(t "Name") placeholder=(t "Example User")}}`, nil,
		`<input aria-label="Name" placeholder="Example User" />`, func(tp *Template) {
			tp.RegisterHelper("input", func(o *Options) any {
				return SafeString(`<input aria-label="` + EscapeExpression(o.Hash["aria-label"]) +
					`" placeholder="` + EscapeExpression(o.Hash["placeholder"]) + `" />`)
			})
			tp.RegisterHelper("t", func(o *Options) any { return SafeString(formatValue(o.Arg(0))) })
		})
}

// TestParityPartialsAdvanced mirrors spec/partials.js: nested partials, partial
// parameters via `others=..`, string context and undefined context.
func TestParityPartialsAdvanced(t *testing.T) {
	parityWith(t, "partial-in-partial", `Dudes: {{#dudes}}{{>dude}}{{/dudes}}`,
		map[string]any{"dudes": []any{
			map[string]any{"name": "Yehuda", "url": "http://yehuda"},
			map[string]any{"name": "Alan", "url": "http://alan"}}},
		`Dudes: Yehuda <a href="http://yehuda">http://yehuda</a> Alan <a href="http://alan">http://alan</a> `,
		func(tp *Template) {
			_ = tp.RegisterPartial("dude", `{{name}} {{> url}} `)
			_ = tp.RegisterPartial("url", `<a href="{{url}}">{{url}}</a>`)
		})
	parityWith(t, "partial-params-parent", `Dudes: {{#dudes}}{{> dude others=..}}{{/dudes}}`,
		map[string]any{"foo": "bar", "dudes": []any{
			map[string]any{"name": "Yehuda", "url": "http://yehuda"},
			map[string]any{"name": "Alan", "url": "http://alan"}}},
		"Dudes: barYehuda (http://yehuda) barAlan (http://alan) ",
		func(tp *Template) { _ = tp.RegisterPartial("dude", `{{others.foo}}{{name}} ({{url}}) `) })
	parityWith(t, "partial-string-context", `Dudes: {{>dude "dudes"}}`, nil, "Dudes: dudes",
		func(tp *Template) { _ = tp.RegisterPartial("dude", `{{.}}`) })
	parityWith(t, "partial-undefined-context", `Dudes: {{>dude dudes}}`, nil, "Dudes:  Empty",
		func(tp *Template) { _ = tp.RegisterPartial("dude", `{{foo}} Empty`) })
}
