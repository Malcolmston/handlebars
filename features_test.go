package handlebars

import (
	"bytes"
	"log"
	"testing"
)

// ---- Block parameters -------------------------------------------------------

func TestBlockParamsEach(t *testing.T) {
	data := map[string]interface{}{"xs": []string{"a", "b"}}
	eq(t, render(t, "{{#each xs as |item index|}}{{index}}:{{item}} {{/each}}", data), "0:a 1:b ")
}

func TestBlockParamsEachSingle(t *testing.T) {
	data := map[string]interface{}{"xs": []int{5, 6}}
	eq(t, render(t, "{{#each xs as |n|}}[{{n}}]{{/each}}", data), "[5][6]")
}

func TestBlockParamsWith(t *testing.T) {
	data := map[string]interface{}{"o": map[string]interface{}{"n": "Z", "k": 3}}
	eq(t, render(t, "{{#with o as |v|}}{{v.n}}#{{v.k}}{{/with}}", data), "Z#3")
}

func TestBlockParamsMapKey(t *testing.T) {
	data := map[string]interface{}{"m": map[string]int{"a": 1, "b": 2}}
	eq(t, render(t, "{{#each m as |val key|}}{{key}}={{val}};{{/each}}", data), "a=1;b=2;")
}

func TestBlockParamsShadowContext(t *testing.T) {
	// The block param n shadows a context field also named n.
	data := map[string]interface{}{"xs": []map[string]interface{}{{"n": "inner"}}, "n": "outer"}
	eq(t, render(t, "{{#each xs as |row|}}{{row.n}}-{{n}}{{/each}}", data), "inner-inner")
}

func TestBlockParamsInSubexpression(t *testing.T) {
	data := map[string]interface{}{"xs": []int{1, 2, 3}}
	eq(t, render(t, "{{#each xs as |x|}}{{#if (eq x 2)}}[{{x}}]{{/if}}{{/each}}", data), "[2]")
}

func TestBlockParamsCustomHelper(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("repeat", func(o *Options) interface{} {
		out := ""
		for i := 0; i < 2; i++ {
			out += o.FnWithBlockParams(nil, i)
		}
		return out
	})
	_ = tmpl.ParseString("{{#repeat as |i|}}<{{i}}>{{/repeat}}")
	eq(t, tmpl.MustRender(nil), "<0><1>")
	// BlockParams exposes declared names.
	tmpl2 := New()
	var names []string
	tmpl2.RegisterHelper("cap", func(o *Options) interface{} { names = o.BlockParams(); return "" })
	_ = tmpl2.ParseString("{{#cap as |a b|}}x{{/cap}}")
	tmpl2.MustRender(nil)
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Fatalf("BlockParams got %v", names)
	}
}

// ---- else if chaining -------------------------------------------------------

func TestElseIfChain(t *testing.T) {
	src := "{{#if a}}A{{else if b}}B{{else if c}}C{{else}}D{{/if}}"
	eq(t, render(t, src, map[string]interface{}{"a": true}), "A")
	eq(t, render(t, src, map[string]interface{}{"b": true}), "B")
	eq(t, render(t, src, map[string]interface{}{"c": true}), "C")
	eq(t, render(t, src, map[string]interface{}{}), "D")
}

func TestElseWithChain(t *testing.T) {
	src := "{{#if a}}A{{else with o}}{{n}}{{/if}}"
	eq(t, render(t, src, map[string]interface{}{"o": map[string]interface{}{"n": "here"}}), "here")
	eq(t, render(t, src, map[string]interface{}{"a": true}), "A")
}

func TestElseUnlessChain(t *testing.T) {
	src := "{{#if a}}A{{else unless b}}NB{{/if}}"
	eq(t, render(t, src, map[string]interface{}{"a": false, "b": false}), "NB")
	eq(t, render(t, src, map[string]interface{}{"a": false, "b": true}), "")
}

// ---- data variables ---------------------------------------------------------

func TestDataFirstLastKey(t *testing.T) {
	data := map[string]interface{}{"xs": []string{"a", "b", "c"}}
	out := render(t, "{{#each xs}}{{@key}}:{{@first}}:{{@last}} {{/each}}", data)
	eq(t, out, "0:true:false 1:false:false 2:false:true ")
}

func TestDataParentIndex(t *testing.T) {
	data := map[string]interface{}{
		"xs": []map[string]interface{}{
			{"ys": []int{1, 2}},
			{"ys": []int{3}},
		},
	}
	out := render(t, "{{#each xs}}{{#each ys}}{{@../index}}-{{@index}} {{/each}}{{/each}}", data)
	eq(t, out, "0-0 0-1 1-0 ")
}

func TestDataLevel(t *testing.T) {
	data := map[string]interface{}{"xs": []map[string]interface{}{{"ys": []int{1}}}}
	out := render(t, "{{#each xs}}{{@level}}{{#each ys}}{{@level}}{{/each}}{{/each}}", data)
	eq(t, out, "12")
}

func TestDataRootDeep(t *testing.T) {
	data := map[string]interface{}{
		"site": "S",
		"xs":   []map[string]interface{}{{"ys": []int{1}}},
	}
	out := render(t, "{{#each xs}}{{#each ys}}{{@root.site}}{{/each}}{{/each}}", data)
	eq(t, out, "S")
}

func TestNoDataOption(t *testing.T) {
	tmpl, err := Compile("{{#each xs}}{{@index}}{{/each}}", NoData())
	if err != nil {
		t.Fatal(err)
	}
	// With data disabled @index resolves to empty.
	eq(t, tmpl.MustRender(map[string]interface{}{"xs": []int{1, 2}}), "")
}

// ---- helper missing hooks ---------------------------------------------------

func TestHelperMissingHook(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("helperMissing", func(o *Options) interface{} {
		return "missing:" + o.Name
	})
	_ = tmpl.ParseString("{{foo 1 2}}")
	eq(t, tmpl.MustRender(nil), "missing:foo")
}

func TestHelperMissingBareIsEmpty(t *testing.T) {
	// A bare {{foo}} with no args and no such helper is empty, not an error.
	eq(t, render(t, "[{{foo}}]", map[string]interface{}{}), "[]")
}

func TestBlockHelperMissingHook(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("blockHelperMissing", func(o *Options) interface{} {
		return "B[" + o.Fn(o.Arg(0)) + "]"
	})
	_ = tmpl.ParseString("{{#thing}}x{{/thing}}")
	eq(t, tmpl.MustRender(map[string]interface{}{"thing": map[string]interface{}{}}), "B[x]")
}

func TestBlockHelperMissingDefaultSection(t *testing.T) {
	// Without a hook, {{#obj}} still behaves as a Mustache section.
	data := map[string]interface{}{"obj": map[string]interface{}{"n": "V"}}
	eq(t, render(t, "{{#obj}}{{n}}{{/obj}}", data), "V")
}

// ---- inline partials, partial blocks, @partial-block ------------------------

func TestPartialBlock(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("layout", "<L>{{> @partial-block}}</L>")
	_ = tmpl.ParseString("{{#> layout}}body {{name}}{{/layout}}")
	eq(t, tmpl.MustRender(map[string]interface{}{"name": "X"}), "<L>body X</L>")
}

func TestPartialBlockFallback(t *testing.T) {
	// If the partial is undefined, the block body is the fallback.
	tmpl := New()
	_ = tmpl.ParseString("{{#> missing}}fallback{{/missing}}")
	eq(t, tmpl.MustRender(nil), "fallback")
}

func TestInlinePartialDecorator(t *testing.T) {
	out := render(t, `{{#*inline "myp"}}hi {{name}}{{/inline}}{{> myp}}`,
		map[string]interface{}{"name": "Q"})
	eq(t, out, "hi Q")
}

func TestInlinePartialInEach(t *testing.T) {
	src := `{{#*inline "row"}}[{{this}}]{{/inline}}{{#each xs}}{{> row}}{{/each}}`
	eq(t, render(t, src, map[string]interface{}{"xs": []int{1, 2}}), "[1][2]")
}

// ---- decorators -------------------------------------------------------------

func TestInlineDecoratorMechanism(t *testing.T) {
	tmpl := New()
	called := false
	tmpl.RegisterDecorator("mark", func(o *DecoratorOptions) { called = true })
	_ = tmpl.ParseString("a{{* mark}}b")
	eq(t, tmpl.MustRender(nil), "ab")
	if !called {
		t.Fatal("decorator not invoked")
	}
}

func TestUnknownDecoratorError(t *testing.T) {
	if _, err := Render("{{* nope}}", nil); err == nil {
		t.Fatal("expected unknown decorator error")
	}
}

// ---- compile options --------------------------------------------------------

func TestNoEscapeOption(t *testing.T) {
	tmpl, err := Compile("{{h}}", NoEscape())
	if err != nil {
		t.Fatal(err)
	}
	eq(t, tmpl.MustRender(map[string]interface{}{"h": "<b>&"}), "<b>&")
}

func TestStrictOption(t *testing.T) {
	tmpl, _ := Compile("{{a.b}}", Strict())
	if _, err := tmpl.Render(map[string]interface{}{}); err == nil {
		t.Fatal("expected strict-mode error for missing path")
	}
	// A present value renders fine.
	ok, _ := Compile("{{a}}", Strict())
	eq(t, ok.MustRender(map[string]interface{}{"a": "v"}), "v")
}

func TestKnownHelpersOnly(t *testing.T) {
	// Unknown helper with args errors under knownHelpersOnly.
	tmpl, _ := Compile("{{shout x}}", KnownHelpersOnly())
	tmpl.RegisterHelper("shout", func(o *Options) interface{} { return "S" })
	if _, err := tmpl.Render(map[string]interface{}{"x": 1}); err == nil {
		t.Fatal("expected error: shout not a known helper")
	}
	// Declaring it known lets it run.
	tmpl2, _ := Compile("{{shout x}}", KnownHelpersOnly(), KnownHelpers("shout"))
	tmpl2.RegisterHelper("shout", func(o *Options) interface{} { return "S" })
	eq(t, tmpl2.MustRender(map[string]interface{}{"x": 1}), "S")
}

// ---- log helper -------------------------------------------------------------

func TestLogHelper(t *testing.T) {
	var buf bytes.Buffer
	tmpl := New().SetLogger(log.New(&buf, "", 0))
	_ = tmpl.ParseString(`{{log "hello" name level="info"}}out`)
	eq(t, tmpl.MustRender(map[string]interface{}{"name": "world"}), "out")
	if buf.String() != "[info] hello world\n" {
		t.Fatalf("log output = %q", buf.String())
	}
}

// ---- whitespace control -----------------------------------------------------

func TestStandalonePartialBlockLines(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("layout", "top\n{{> @partial-block}}\nbot\n")
	_ = tmpl.ParseString("{{#> layout}}\nMIDDLE\n{{/layout}}\n")
	eq(t, tmpl.MustRender(nil), "top\nMIDDLE\nbot\n")
}

func TestStandaloneInlinePartial(t *testing.T) {
	// A standalone inline-partial declaration has its own lines stripped.
	src := "start\n{{#*inline \"x\"}}\nY\n{{/inline}}\n{{> x}}\nend\n"
	eq(t, render(t, src, nil), "start\nY\nend\n")
}

func TestTildeOnBlock(t *testing.T) {
	eq(t, render(t, "a\n  {{~#if v~}}  X  {{~/if~}}\n  b", map[string]interface{}{"v": true}), "aXb")
}

// ---- nested subexpressions --------------------------------------------------

func TestDeeplyNestedSubexpr(t *testing.T) {
	data := map[string]interface{}{"a": 1, "b": 2, "c": 3}
	out := render(t, "{{#if (or (eq a 9) (and (lt a b) (lt b c)))}}yes{{/if}}", data)
	eq(t, out, "yes")
}

func TestHashArgsOnBlockHelper(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("wrap", func(o *Options) interface{} {
		return o.HashStr("tag", "span") + ":" + o.Fn(o.Arg(0))
	})
	_ = tmpl.ParseString(`{{#wrap tag="b"}}hi{{/wrap}}`)
	eq(t, tmpl.MustRender(nil), "b:hi")
}
