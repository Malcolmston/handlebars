package handlebars

// This file encodes known-answer test vectors taken directly from the upstream
// Handlebars.js test suite (handlebars-lang/handlebars.js, spec/*.js) so the Go
// port is pinned to the original library's asserted behaviour. Each case mirrors
// an `expectTemplate(...).withInput(...).toCompileTo(...)` assertion from the
// referenced spec file. These are the durable parity artifact; run them with
// `go test -run Parity`.

import "testing"

// parity renders tmpl against data with a fresh Template (built-ins only) and
// asserts the output equals want.
func parity(t *testing.T, name, tmpl string, data interface{}, want string) {
	t.Helper()
	got, err := Render(tmpl, data)
	if err != nil {
		t.Errorf("%s: %q render error: %v", name, tmpl, err)
		return
	}
	if got != want {
		t.Errorf("%s: %q\n  got  %q\n  want %q", name, tmpl, got, want)
	}
}

// parityWith is like parity but lets the caller register helpers or partials on
// the template before rendering.
func parityWith(t *testing.T, name, tmpl string, data interface{}, want string, setup func(*Template)) {
	t.Helper()
	tp, err := Parse(tmpl)
	if err != nil {
		t.Errorf("%s: %q parse error: %v", name, tmpl, err)
		return
	}
	if setup != nil {
		setup(tp)
	}
	got, err := tp.Render(data)
	if err != nil {
		t.Errorf("%s: %q render error: %v", name, tmpl, err)
		return
	}
	if got != want {
		t.Errorf("%s: %q\n  got  %q\n  want %q", name, tmpl, got, want)
	}
}

// TestParityBasic mirrors spec/basic.js.
func TestParityBasic(t *testing.T) {
	parity(t, "most-basic", `{{foo}}`, map[string]any{"foo": "foo"}, "foo")
	parity(t, "compile-basic", "Goodbye\n{{cruel}}\n{{world}}!",
		map[string]any{"cruel": "cruel", "world": "world"}, "Goodbye\ncruel\nworld!")

	// escaping (backslash-escaped mustaches)
	parity(t, "escape-1", `\{{foo}}`, map[string]any{"foo": "food"}, `{{foo}}`)
	parity(t, "escape-2", `content \{{foo}}`, map[string]any{"foo": "food"}, `content {{foo}}`)
	parity(t, "escape-3", `\\{{foo}}`, map[string]any{"foo": "food"}, `\food`)
	parity(t, "escape-4", `content \\{{foo}}`, map[string]any{"foo": "food"}, `content \food`)
	parity(t, "escape-5", `\\ {{foo}}`, map[string]any{"foo": "food"}, `\\ food`)

	// comments
	parity(t, "comment-ignored", "{{! Goodbye}}Goodbye\n{{cruel}}\n{{world}}!",
		map[string]any{"cruel": "cruel", "world": "world"}, "Goodbye\ncruel\nworld!")
	parity(t, "comment-trim-both", `    {{~! comment ~}}      blah`, nil, "blah")
	parity(t, "comment-long-trim-both", `    {{~!-- long-comment --~}}      blah`, nil, "blah")
	parity(t, "comment-trim-right", `    {{! comment ~}}      blah`, nil, "    blah")
	parity(t, "comment-long-trim-right", `    {{!-- long-comment --~}}      blah`, nil, "    blah")
	parity(t, "comment-trim-left", `    {{~! comment}}      blah`, nil, "      blah")
	parity(t, "comment-long-trim-left", `    {{~!-- long-comment --}}      blah`, nil, "      blah")

	// boolean
	parity(t, "boolean-true", `{{#goodbye}}GOODBYE {{/goodbye}}cruel {{world}}!`,
		map[string]any{"goodbye": true, "world": "world"}, "GOODBYE cruel world!")
	parity(t, "boolean-false", `{{#goodbye}}GOODBYE {{/goodbye}}cruel {{world}}!`,
		map[string]any{"goodbye": false, "world": "world"}, "cruel world!")

	// zeros
	parity(t, "zeros", `num1: {{num1}}, num2: {{num2}}`,
		map[string]any{"num1": 42, "num2": 0}, "num1: 42, num2: 0")
	parity(t, "zero-dot", `num: {{.}}`, 0, "num: 0")
	parity(t, "zero-nested", `num: {{num1/num2}}`,
		map[string]any{"num1": map[string]any{"num2": 0}}, "num: 0")

	// false values
	parity(t, "false-fields", `val1: {{val1}}, val2: {{val2}}`,
		map[string]any{"val1": false, "val2": false}, "val1: false, val2: false")
	parity(t, "false-dot", `val: {{.}}`, false, "val: false")

	// newlines
	parity(t, "newline-lf", "Alan's\nTest", nil, "Alan's\nTest")
	parity(t, "newline-cr", "Alan's\rTest", nil, "Alan's\rTest")

	// escaping expressions
	parity(t, "unescaped-triple", `{{{awesome}}}`, map[string]any{"awesome": "&'\\<>"}, "&'\\<>")
	parity(t, "unescaped-amp", `{{&awesome}}`, map[string]any{"awesome": "&'\\<>"}, "&'\\<>")
	parity(t, "escape-default", `{{awesome}}`,
		map[string]any{"awesome": "&\"'`\\<>"}, "&amp;&quot;&#x27;&#x60;\\&lt;&gt;")
	parity(t, "escape-amperstands", `{{awesome}}`,
		map[string]any{"awesome": "Escaped, <b> looks like: &lt;b&gt;"},
		"Escaped, &lt;b&gt; looks like: &amp;lt;b&amp;gt;")

	// paths with hyphens
	parity(t, "hyphen-1", `{{foo-bar}}`, map[string]any{"foo-bar": "baz"}, "baz")
	parity(t, "hyphen-2", `{{foo.foo-bar}}`,
		map[string]any{"foo": map[string]any{"foo-bar": "baz"}}, "baz")
	parity(t, "hyphen-3", `{{foo/foo-bar}}`,
		map[string]any{"foo": map[string]any{"foo-bar": "baz"}}, "baz")

	// nested paths
	parity(t, "nested", `Goodbye {{alan/expression}} world!`,
		map[string]any{"alan": map[string]any{"expression": "beautiful"}},
		"Goodbye beautiful world!")
	parity(t, "nested-empty", `Goodbye {{alan/expression}} world!`,
		map[string]any{"alan": map[string]any{"expression": ""}}, "Goodbye  world!")

	// literal paths (bracket form)
	parity(t, "literal-at", `Goodbye {{[@alan]/expression}} world!`,
		map[string]any{"@alan": map[string]any{"expression": "beautiful"}},
		"Goodbye beautiful world!")
	parity(t, "literal-space", `Goodbye {{[foo bar]/expression}} world!`,
		map[string]any{"foo bar": map[string]any{"expression": "beautiful"}},
		"Goodbye beautiful world!")

	// literal references (quoted-string form used as a path)
	parity(t, "litref-bracket", `Goodbye {{[foo bar]}} world!`,
		map[string]any{"foo bar": "beautiful"}, "Goodbye beautiful world!")
	parity(t, "litref-dquote", `Goodbye {{"foo bar"}} world!`,
		map[string]any{"foo bar": "beautiful"}, "Goodbye beautiful world!")
	parity(t, "litref-squote", `Goodbye {{'foo bar'}} world!`,
		map[string]any{"foo bar": "beautiful"}, "Goodbye beautiful world!")

	// pass string / number literals resolved as path
	parity(t, "pass-string-empty", `{{"foo"}}`, nil, "")
	parity(t, "pass-string", `{{"foo"}}`, map[string]any{"foo": "bar"}, "bar")
	parity(t, "pass-string-block", `{{#"foo"}}{{.}}{{/"foo"}}`,
		map[string]any{"foo": []string{"bar", "baz"}}, "barbaz")
	parity(t, "pass-number", `{{12}}`, map[string]any{"12": "bar"}, "bar")

	// this keyword in paths
	parity(t, "this-current", `{{#goodbyes}}{{this}}{{/goodbyes}}`,
		map[string]any{"goodbyes": []string{"goodbye", "Goodbye", "GOODBYE"}},
		"goodbyeGoodbyeGOODBYE")
	parity(t, "this-nested", `{{#hellos}}{{this/text}}{{/hellos}}`,
		map[string]any{"hellos": []map[string]any{{"text": "hello"}, {"text": "Hello"}, {"text": "HELLO"}}},
		"helloHelloHELLO")
	parity(t, "this-bracket", `{{[this]}}`, map[string]any{"this": "bar"}, "bar")
	parity(t, "this-nested-bracket", `{{text/[this]}}`,
		map[string]any{"text": map[string]any{"this": "bar"}}, "bar")

	// complex but empty paths
	parity(t, "empty-null", `{{person/name}}`,
		map[string]any{"person": map[string]any{"name": nil}}, "")
	parity(t, "empty-missing", `{{person/name}}`,
		map[string]any{"person": map[string]any{}}, "")

	// string virtual length
	parity(t, "string-length", `{{.}}{{length}}`, "bye", "bye3")
}

// TestParityBasicCompileOptions mirrors spec/basic.js cases that rely on the
// noEscape compile option, and the SafeString behaviour.
func TestParityBasicCompileOptions(t *testing.T) {
	tp, err := Compile(`{{a}}{{b}}`, NoEscape())
	if err != nil {
		t.Fatal(err)
	}
	if got := tp.MustRender(map[string]any{"a": 1, "b": 2}); got != "12" {
		t.Errorf("noEscape adjacent: got %q want %q", got, "12")
	}
	tp2, _ := Compile(`{{a}}{{b}}{{c}}`, NoEscape())
	if got := tp2.MustRender(map[string]any{"a": 1, "b": 2, "c": 3}); got != "123" {
		t.Errorf("noEscape 3 adjacent: got %q want %q", got, "123")
	}
	parity(t, "raw-adjacent", `{{{a}}}{{{b}}}`, map[string]any{"a": 1, "b": 2}, "12")

	// functions returning safestrings shouldn't be escaped
	parityWith(t, "safestring", `{{awesome}}`, nil, "&'\\<>", func(tp *Template) {
		tp.RegisterHelper("awesome", func(o *Options) any { return SafeString("&'\\<>") })
	})
}

// TestParityBlocks mirrors spec/blocks.js.
func TestParityBlocks(t *testing.T) {
	gb := []map[string]any{{"text": "goodbye"}, {"text": "Goodbye"}, {"text": "GOODBYE"}}

	parity(t, "array", `{{#goodbyes}}{{text}}! {{/goodbyes}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"},
		"goodbye! Goodbye! GOODBYE! cruel world!")
	parity(t, "array-empty", `{{#goodbyes}}{{text}}! {{/goodbyes}}cruel {{world}}!`,
		map[string]any{"goodbyes": []map[string]any{}, "world": "world"}, "cruel world!")
	parity(t, "array-index", `{{#goodbyes}}{{@index}}. {{text}}! {{/goodbyes}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"},
		"0. goodbye! 1. Goodbye! 2. GOODBYE! cruel world!")
	parity(t, "empty-block", `{{#goodbyes}}{{/goodbyes}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"}, "cruel world!")
	parity(t, "complex-lookup", `{{#goodbyes}}{{text}} cruel {{../name}}! {{/goodbyes}}`,
		map[string]any{"name": "Alan", "goodbyes": gb},
		"goodbye cruel Alan! Goodbye cruel Alan! GOODBYE cruel Alan! ")
	parity(t, "multiple-lookup", `{{#goodbyes}}{{../name}}{{../name}}{{/goodbyes}}`,
		map[string]any{"name": "Alan", "goodbyes": gb}, "AlanAlanAlanAlanAlanAlan")
	parity(t, "deep-nested",
		`{{#outer}}Goodbye {{#inner}}cruel {{../sibling}} {{../../omg}}{{/inner}}{{/outer}}`,
		map[string]any{"omg": "OMG!", "outer": []map[string]any{
			{"sibling": "sad", "inner": []map[string]any{{"text": "goodbye"}}}}},
		"Goodbye cruel sad OMG!")

	// inverted sections
	parity(t, "inverted-unset",
		`{{#goodbyes}}{{this}}{{/goodbyes}}{{^goodbyes}}Right On!{{/goodbyes}}`,
		map[string]any{}, "Right On!")
	parity(t, "inverted-false",
		`{{#goodbyes}}{{this}}{{/goodbyes}}{{^goodbyes}}Right On!{{/goodbyes}}`,
		map[string]any{"goodbyes": false}, "Right On!")
	parity(t, "inverted-empty",
		`{{#goodbyes}}{{this}}{{/goodbyes}}{{^goodbyes}}Right On!{{/goodbyes}}`,
		map[string]any{"goodbyes": []string{}}, "Right On!")
	parity(t, "block-inverted", `{{#people}}{{name}}{{^}}{{none}}{{/people}}`,
		map[string]any{"none": "No people"}, "No people")
	parity(t, "chained-else-if", `{{#people}}{{name}}{{else if none}}{{none}}{{/people}}`,
		map[string]any{"none": "No people"}, "No people")
}

// TestParityBuiltins mirrors spec/builtins.js (#if, #unless, #with, #each).
func TestParityBuiltins(t *testing.T) {
	ifTmpl := `{{#if goodbye}}GOODBYE {{/if}}cruel {{world}}!`
	parity(t, "if-true", ifTmpl, map[string]any{"goodbye": true, "world": "world"}, "GOODBYE cruel world!")
	parity(t, "if-string", ifTmpl, map[string]any{"goodbye": "dummy", "world": "world"}, "GOODBYE cruel world!")
	parity(t, "if-false", ifTmpl, map[string]any{"goodbye": false, "world": "world"}, "cruel world!")
	parity(t, "if-undefined", ifTmpl, map[string]any{"world": "world"}, "cruel world!")
	parity(t, "if-array", ifTmpl, map[string]any{"goodbye": []string{"foo"}, "world": "world"}, "GOODBYE cruel world!")
	parity(t, "if-empty-array", ifTmpl, map[string]any{"goodbye": []string{}, "world": "world"}, "cruel world!")
	parity(t, "if-zero", ifTmpl, map[string]any{"goodbye": 0, "world": "world"}, "cruel world!")
	parity(t, "if-include-zero",
		`{{#if goodbye includeZero=true}}GOODBYE {{/if}}cruel {{world}}!`,
		map[string]any{"goodbye": 0, "world": "world"}, "GOODBYE cruel world!")
	parity(t, "if-depth",
		`{{#with foo}}{{#if goodbye}}GOODBYE cruel {{../world}}!{{/if}}{{/with}}`,
		map[string]any{"foo": map[string]any{"goodbye": true}, "world": "world"},
		"GOODBYE cruel world!")

	// #with
	parity(t, "with", `{{#with person}}{{first}} {{last}}{{/with}}`,
		map[string]any{"person": map[string]any{"first": "Alan", "last": "Johnson"}}, "Alan Johnson")
	parity(t, "with-else",
		`{{#with person}}Person is present{{else}}Person is not present{{/with}}`,
		map[string]any{}, "Person is not present")
	parity(t, "with-block-param", `{{#with person as |foo|}}{{foo.first}} {{last}}{{/with}}`,
		map[string]any{"person": map[string]any{"first": "Alan", "last": "Johnson"}}, "Alan Johnson")

	// #each
	eachTmpl := `{{#each goodbyes}}{{text}}! {{/each}}cruel {{world}}!`
	gb := []map[string]any{{"text": "goodbye"}, {"text": "Goodbye"}, {"text": "GOODBYE"}}
	parity(t, "each", eachTmpl, map[string]any{"goodbyes": gb, "world": "world"},
		"goodbye! Goodbye! GOODBYE! cruel world!")
	parity(t, "each-empty", eachTmpl, map[string]any{"goodbyes": []map[string]any{}, "world": "world"}, "cruel world!")
	parity(t, "each-dot", `{{#each .}}{{.}}{{/each}}`, []string{"cruel", "world"}, "cruelworld")
	parity(t, "each-index", `{{#each goodbyes}}{{@index}}. {{text}}! {{/each}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"},
		"0. goodbye! 1. Goodbye! 2. GOODBYE! cruel world!")
	parity(t, "each-first", `{{#each goodbyes}}{{#if @first}}{{text}}! {{/if}}{{/each}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"}, "goodbye! cruel world!")
	parity(t, "each-last", `{{#each goodbyes}}{{#if @last}}{{text}}! {{/if}}{{/each}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"}, "GOODBYE! cruel world!")
	parity(t, "each-block-params",
		`{{#each goodbyes as |value index|}}{{index}}. {{value.text}}!{{/each}}`,
		map[string]any{"goodbyes": gb[:2]}, "0. goodbye!1. Goodbye!")
	parity(t, "each-nested-index",
		`{{#each goodbyes}}{{@index}}. {{text}}! {{#each ../goodbyes}}{{@index}} {{/each}}After {{@index}} {{/each}}{{@index}}cruel {{world}}!`,
		map[string]any{"goodbyes": gb, "world": "world"},
		"0. goodbye! 0 1 2 After 0 1. Goodbye! 0 1 2 After 1 2. GOODBYE! 0 1 2 After 2 cruel world!")

	// #each over an object with @key (single key for determinism)
	parity(t, "each-key", `{{#each goodbyes}}{{@key}}. {{this}}! {{/each}}`,
		map[string]any{"goodbyes": map[string]any{"foo": "bar"}}, "foo. bar! ")
}

// TestParityWhitespaceControl mirrors spec/whitespace-control.js.
func TestParityWhitespaceControl(t *testing.T) {
	hash := map[string]any{"foo": "bar<"}
	parity(t, "mustache-trim-both", ` {{~foo~}} `, hash, "bar&lt;")
	parity(t, "mustache-trim-left", ` {{~foo}} `, hash, "bar&lt; ")
	parity(t, "mustache-trim-right", ` {{foo~}} `, hash, " bar&lt;")
	parity(t, "mustache-raw-trim", ` {{~&foo~}} `, hash, "bar<")
	parity(t, "mustache-triple-trim", ` {{~{foo}~}} `, hash, "bar<")
	parity(t, "mustache-newlines", "1\n{{foo~}} \n\n 23\n{{bar}}4", nil, "1\n23\n4")

	// block trims
	parity(t, "block-trim-both", ` {{~#if foo~}} bar {{~/if~}} `, hash, "bar")
	parity(t, "block-trim-inner-right", ` {{#if foo~}} bar {{/if~}} `, hash, " bar ")
	parity(t, "block-trim-inner-left", ` {{~#if foo}} bar {{~/if}} `, hash, " bar ")
	parity(t, "block-no-trim", ` {{#if foo}} bar {{/if}} `, hash, "  bar  ")
	parity(t, "block-complex-1", `{{#if foo~}} bar {{~^~}} baz {{~/if}}`, hash, "bar")
	parity(t, "block-complex-2", `{{#if foo~}} bar {{^~}} baz {{/if}}`, hash, "bar ")
	parity(t, "block-complex-else", `{{#if foo~}} bar {{~else~}} baz {{~/if}}`, hash, "bar")
	parity(t, "block-complex-inv", `{{#if foo~}} bar {{~^~}} baz {{~/if}}`, nil, "baz")

	// partial trims
	parityWith(t, "partial-trim-both", `foo {{~> dude~}} `, nil, "foobar",
		func(tp *Template) { _ = tp.RegisterPartial("dude", "bar") })
	parityWith(t, "partial-trim-right", `foo {{> dude~}} `, nil, "foo bar",
		func(tp *Template) { _ = tp.RegisterPartial("dude", "bar") })
	parityWith(t, "partial-no-trim", `foo {{> dude}} `, nil, "foo bar ",
		func(tp *Template) { _ = tp.RegisterPartial("dude", "bar") })

	// strip whitespace only once
	parity(t, "strip-once", ` {{~foo~}} {{foo}} {{foo}} `, map[string]any{"foo": "bar"}, "barbar bar ")
}

// TestParitySubexpressions mirrors spec/subexpressions.js.
func TestParitySubexpressions(t *testing.T) {
	parityWith(t, "arg-less", `{{foo (bar)}}!`, nil, "LOLLOL!", func(tp *Template) {
		tp.RegisterHelper("foo", func(o *Options) any { return formatValue(o.Arg(0)) + formatValue(o.Arg(0)) })
		tp.RegisterHelper("bar", func(o *Options) any { return "LOL" })
	})
	parityWith(t, "helper-w-args", `{{blog (equal a b)}}`,
		map[string]any{"a": "x", "b": "x"}, "val is true", func(tp *Template) {
			tp.RegisterHelper("blog", func(o *Options) any { return "val is " + formatValue(o.Arg(0)) })
			tp.RegisterHelper("equal", func(o *Options) any { return o.Arg(0) == o.Arg(1) })
		})
	parityWith(t, "mixed-paths", `{{blog baz.bat (equal a b) baz.bar}}`,
		map[string]any{"baz": map[string]any{"bat": "foo!", "bar": "bar!"}, "a": 1, "b": 1},
		"val is foo!, true and bar!", func(tp *Template) {
			tp.RegisterHelper("blog", func(o *Options) any {
				return "val is " + formatValue(o.Arg(0)) + ", " + formatValue(o.Arg(1)) + " and " + formatValue(o.Arg(2))
			})
			tp.RegisterHelper("equal", func(o *Options) any { return o.Arg(0) == o.Arg(1) })
		})
	parityWith(t, "much-nesting", `{{blog (equal (equal true true) true)}}`, nil, "val is true",
		func(tp *Template) {
			tp.RegisterHelper("blog", func(o *Options) any { return "val is " + formatValue(o.Arg(0)) })
			tp.RegisterHelper("equal", func(o *Options) any { return o.Arg(0) == o.Arg(1) })
		})
	// GH-800: complex subexpressions
	dashConcat := func(tp *Template) {
		tp.RegisterHelper("dash", func(o *Options) any { return formatValue(o.Arg(0)) + "-" + formatValue(o.Arg(1)) })
		tp.RegisterHelper("concat", func(o *Options) any { return formatValue(o.Arg(0)) + formatValue(o.Arg(1)) })
	}
	ctx := map[string]any{"a": "a", "b": "b", "c": map[string]any{"c": "c"}, "d": "d", "e": map[string]any{"e": "e"}}
	parityWith(t, "gh800-strlit", `{{dash 'abc' (concat a b)}}`, ctx, "abc-ab", dashConcat)
	parityWith(t, "gh800-path", `{{dash d (concat a b)}}`, ctx, "d-ab", dashConcat)
	parityWith(t, "gh800-nested-path", `{{dash c.c (concat a b)}}`, ctx, "c-ab", dashConcat)
	parityWith(t, "gh800-reordered", `{{dash (concat a b) c.c}}`, ctx, "ab-c", dashConcat)
	parityWith(t, "gh800-both-nested", `{{dash (concat a e.e) c.c}}`, ctx, "ae-c", dashConcat)
}

// TestParityPartials mirrors spec/partials.js.
func TestParityPartials(t *testing.T) {
	hash := map[string]any{"dudes": []map[string]any{
		{"name": "Yehuda", "url": "http://yehuda"}, {"name": "Alan", "url": "http://alan"}}}
	parityWith(t, "basic-partials", `Dudes: {{#dudes}}{{> dude}}{{/dudes}}`, hash,
		"Dudes: Yehuda (http://yehuda) Alan (http://alan) ",
		func(tp *Template) { _ = tp.RegisterPartial("dude", "{{name}} ({{url}}) ") })
	parityWith(t, "partial-with-context", `Dudes: {{>dude dudes}}`,
		map[string]any{"dudes": map[string]any{"name": "Alan"}}, "Dudes: Alan",
		func(tp *Template) { _ = tp.RegisterPartial("dude", "{{name}}") })
	parityWith(t, "dynamic-partial", `Dudes: {{#dudes}}{{> (partial)}}{{/dudes}}`, hash,
		"Dudes: Yehuda (http://yehuda) Alan (http://alan) ", func(tp *Template) {
			_ = tp.RegisterPartial("dude", "{{name}} ({{url}}) ")
			tp.RegisterHelper("partial", func(o *Options) any { return "dude" })
		})
}

// TestParityData mirrors spec/data.js (@root and @-data access).
func TestParityData(t *testing.T) {
	parity(t, "root-lookup", `{{@root.foo}}`, map[string]any{"foo": "hello"}, "hello")
	parity(t, "root-in-block", `{{#with x}}{{@root.foo}}{{/with}}`,
		map[string]any{"foo": "hello", "x": map[string]any{"bar": "b"}}, "hello")
	parity(t, "index-data", `{{#each list}}{{@index}}{{/each}}`,
		map[string]any{"list": []string{"a", "b", "c"}}, "012")
}
