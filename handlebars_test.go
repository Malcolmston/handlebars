package handlebars

import (
	"strings"
	"testing"
)

// render is a test helper that parses and renders in one step, failing on error.
func render(t *testing.T, src string, data interface{}) string {
	t.Helper()
	out, err := Render(src, data)
	if err != nil {
		t.Fatalf("Render(%q) error: %v", src, err)
	}
	return out
}

func eq(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestPlainTextAndComments(t *testing.T) {
	eq(t, render(t, "hello world", nil), "hello world")
	eq(t, render(t, "a{{! short }}b{{!-- long -- comment --}}c", nil), "abc")
}

func TestInterpolationEscaping(t *testing.T) {
	data := map[string]interface{}{"h": `<a href="x">&'`}
	eq(t, render(t, "{{h}}", data), "&lt;a href&#x3D;&quot;x&quot;&gt;&amp;&#x27;")
	// Triple-stache and & are unescaped.
	eq(t, render(t, "{{{h}}}", data), `<a href="x">&'`)
	eq(t, render(t, "{{& h}}", data), `<a href="x">&'`)
}

func TestMissingValueIsEmpty(t *testing.T) {
	eq(t, render(t, "[{{nope}}]", map[string]interface{}{}), "[]")
	eq(t, render(t, "[{{a.b.c}}]", map[string]interface{}{"a": map[string]interface{}{}}), "[]")
}

func TestPaths(t *testing.T) {
	data := map[string]interface{}{
		"user": map[string]interface{}{
			"name":  "Ada",
			"roles": []string{"admin", "dev"},
		},
	}
	eq(t, render(t, "{{user.name}}", data), "Ada")
	eq(t, render(t, "{{user.roles.0}}-{{user.roles.1}}", data), "admin-dev")
	eq(t, render(t, "{{this.user.name}}", data), "Ada")
}

func TestParentPath(t *testing.T) {
	data := map[string]interface{}{
		"title": "Book",
		"chapters": []map[string]interface{}{
			{"name": "One"},
			{"name": "Two"},
		},
	}
	out := render(t, "{{#each chapters}}{{../title}}:{{name}} {{/each}}", data)
	eq(t, out, "Book:One Book:Two ")
}

func TestDeepParentPath(t *testing.T) {
	data := map[string]interface{}{
		"x": "root",
		"a": map[string]interface{}{
			"b": []string{"v"},
		},
	}
	out := render(t, "{{#with a}}{{#each b}}{{../../x}}{{/each}}{{/with}}", data)
	eq(t, out, "root")
}

func TestIfElseUnless(t *testing.T) {
	eq(t, render(t, "{{#if ok}}yes{{else}}no{{/if}}", map[string]interface{}{"ok": true}), "yes")
	eq(t, render(t, "{{#if ok}}yes{{else}}no{{/if}}", map[string]interface{}{"ok": false}), "no")
	eq(t, render(t, "{{#if list}}has{{else}}empty{{/if}}", map[string]interface{}{"list": []int{}}), "empty")
	eq(t, render(t, "{{#unless ok}}no{{/unless}}", map[string]interface{}{"ok": false}), "no")
	eq(t, render(t, "{{#unless ok}}no{{/unless}}", map[string]interface{}{"ok": true}), "")
}

func TestInverseSection(t *testing.T) {
	eq(t, render(t, "{{^flag}}off{{/flag}}", map[string]interface{}{"flag": false}), "off")
	eq(t, render(t, "{{^flag}}off{{/flag}}", map[string]interface{}{"flag": true}), "")
	eq(t, render(t, "{{^flag}}off{{else}}on{{/flag}}", map[string]interface{}{"flag": true}), "on")
}

func TestWith(t *testing.T) {
	data := map[string]interface{}{"acct": map[string]interface{}{"id": 7, "kind": "gold"}}
	eq(t, render(t, "{{#with acct}}{{kind}}#{{id}}{{/with}}", data), "gold#7")
	eq(t, render(t, "{{#with missing}}x{{else}}none{{/with}}", data), "none")
}

func TestEachSlice(t *testing.T) {
	data := map[string]interface{}{"xs": []string{"a", "b", "c"}}
	out := render(t, "{{#each xs}}[{{@index}}:{{this}}:{{@first}}:{{@last}}]{{/each}}", data)
	eq(t, out, "[0:a:true:false][1:b:false:false][2:c:false:true]")
}

func TestEachSliceEmpty(t *testing.T) {
	eq(t, render(t, "{{#each xs}}x{{else}}nada{{/each}}", map[string]interface{}{"xs": []int{}}), "nada")
}

func TestEachMapSortedKeys(t *testing.T) {
	data := map[string]interface{}{"m": map[string]int{"gamma": 3, "alpha": 1, "beta": 2}}
	// Keys must be visited in sorted order for determinism.
	out := render(t, "{{#each m}}{{@key}}={{this}}({{@index}});{{/each}}", data)
	eq(t, out, "alpha=1(0);beta=2(1);gamma=3(2);")
}

func TestNestedEach(t *testing.T) {
	data := map[string]interface{}{
		"groups": []map[string]interface{}{
			{"name": "A", "items": []string{"1", "2"}},
			{"name": "B", "items": []string{"3"}},
		},
	}
	out := render(t, "{{#each groups}}{{name}}:{{#each items}}{{this}}{{/each}} {{/each}}", data)
	eq(t, out, "A:12 B:3 ")
}

type profile struct {
	FullName string `json:"full_name"`
	Age      int
	secret   string //nolint:unused
}

func TestStructFieldAndJSONTag(t *testing.T) {
	p := profile{FullName: "Grace", Age: 40, secret: "x"}
	eq(t, render(t, "{{FullName}} {{Age}}", p), "Grace 40")
	eq(t, render(t, "{{full_name}}", p), "Grace")
	// case-insensitive fallback
	eq(t, render(t, "{{age}}", p), "40")
}

func TestStructPointerAndSlice(t *testing.T) {
	p := &profile{FullName: "Lin", Age: 5}
	eq(t, render(t, "{{FullName}}", p), "Lin")
	data := map[string]interface{}{"people": []profile{{FullName: "A"}, {FullName: "B"}}}
	eq(t, render(t, "{{#each people}}{{FullName}}{{/each}}", data), "AB")
}

func TestCustomHelperInline(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("upper", func(o *Options) interface{} {
		return strings.ToUpper(formatValue(o.Arg(0)))
	})
	if err := tmpl.ParseString("{{upper name}}"); err != nil {
		t.Fatal(err)
	}
	out, err := tmpl.Render(map[string]interface{}{"name": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, out, "HI")
}

func TestCustomHelperHashArgs(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("link", func(o *Options) interface{} {
		return SafeString("<a href=\"" + o.HashStr("href", "#") + "\">" + formatValue(o.Arg(0)) + "</a>")
	})
	_ = tmpl.ParseString(`{{link "Home" href="/index"}}`)
	out := tmpl.MustRender(nil)
	eq(t, out, `<a href="/index">Home</a>`)
}

func TestCustomBlockHelper(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("bold", func(o *Options) interface{} {
		return "<b>" + o.Fn(o.Arg(0)) + "</b>"
	})
	_ = tmpl.ParseString("{{#bold user}}{{name}}{{/bold}}")
	out := tmpl.MustRender(map[string]interface{}{"user": map[string]interface{}{"name": "Z"}})
	eq(t, out, "<b>Z</b>")
}

func TestBlockHelperInverse(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("iff", func(o *Options) interface{} {
		if isTruthy(o.Arg(0)) {
			return o.Fn(nil)
		}
		return o.Inverse(nil)
	})
	_ = tmpl.ParseString("{{#iff v}}Y{{else}}N{{/iff}}")
	eq(t, tmpl.MustRender(map[string]interface{}{"v": 0}), "N")
	eq(t, tmpl.MustRender(map[string]interface{}{"v": 1}), "Y")
}

func TestSubexpressions(t *testing.T) {
	eq(t, render(t, "{{#if (eq a b)}}same{{else}}diff{{/if}}", map[string]interface{}{"a": 2, "b": 2}), "same")
	eq(t, render(t, "{{#if (and x y)}}both{{/if}}", map[string]interface{}{"x": true, "y": true}), "both")
	eq(t, render(t, "{{#if (gt a b)}}g{{else}}le{{/if}}", map[string]interface{}{"a": 5, "b": 3}), "g")
	eq(t, render(t, "{{lookup m key}}", map[string]interface{}{"m": map[string]string{"k": "v"}, "key": "k"}), "v")
}

func TestBuiltinComparators(t *testing.T) {
	eq(t, render(t, "{{#if (ne a b)}}ne{{/if}}", map[string]interface{}{"a": 1, "b": 2}), "ne")
	eq(t, render(t, "{{#if (or x y)}}o{{/if}}", map[string]interface{}{"x": false, "y": true}), "o")
	eq(t, render(t, "{{#if (not x)}}n{{/if}}", map[string]interface{}{"x": false}), "n")
	eq(t, render(t, "{{#if (lte a b)}}le{{/if}}", map[string]interface{}{"a": 2, "b": 2}), "le")
	eq(t, render(t, "{{#if (gte a b)}}ge{{/if}}", map[string]interface{}{"a": 3, "b": 2}), "ge")
	eq(t, render(t, "{{#if (lt a b)}}lt{{/if}}", map[string]interface{}{"a": 1, "b": 2}), "lt")
}

func TestPartials(t *testing.T) {
	tmpl := New()
	if err := tmpl.RegisterPartial("greeting", "Hi {{name}}!"); err != nil {
		t.Fatal(err)
	}
	_ = tmpl.ParseString("{{> greeting}}")
	eq(t, tmpl.MustRender(map[string]interface{}{"name": "Sam"}), "Hi Sam!")
}

func TestPartialWithContextArg(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("card", "[{{title}}]")
	_ = tmpl.ParseString("{{> card featured}}")
	out := tmpl.MustRender(map[string]interface{}{"featured": map[string]interface{}{"title": "Hot"}})
	eq(t, out, "[Hot]")
}

func TestPartialWithHashArgs(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("row", "{{label}}={{value}}")
	_ = tmpl.ParseString("{{> row label=\"k\"}}")
	out := tmpl.MustRender(map[string]interface{}{"value": 9})
	eq(t, out, "k=9")
}

func TestDynamicPartial(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("en", "hello")
	_ = tmpl.RegisterPartial("fr", "bonjour")
	_ = tmpl.ParseString("{{> (lookup . 'which')}}")
	eq(t, tmpl.MustRender(map[string]interface{}{"which": "fr"}), "bonjour")
}

func TestStandaloneWhitespace(t *testing.T) {
	src := "start\n{{#each xs}}\n  {{this}}\n{{/each}}\nend\n"
	out := render(t, src, map[string]interface{}{"xs": []string{"a", "b"}})
	eq(t, out, "start\n  a\n  b\nend\n")
}

func TestTildeWhitespaceControl(t *testing.T) {
	eq(t, render(t, "a  {{~v~}}  b", map[string]interface{}{"v": "X"}), "aXb")
}

func TestStandalonePartialIndent(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartial("block", "line1\nline2\n")
	_ = tmpl.ParseString("A\n  {{> block}}\nB")
	eq(t, tmpl.MustRender(nil), "A\n  line1\n  line2\nB")
}

func TestRootData(t *testing.T) {
	data := map[string]interface{}{
		"site":  "MySite",
		"items": []string{"x"},
	}
	out := render(t, "{{#each items}}{{@root.site}}{{/each}}", data)
	eq(t, out, "MySite")
}

func TestFloatFormatting(t *testing.T) {
	eq(t, render(t, "{{n}}", map[string]interface{}{"n": 3.0}), "3")
	eq(t, render(t, "{{n}}", map[string]interface{}{"n": 3.5}), "3.5")
	eq(t, render(t, "{{n}}", map[string]interface{}{"n": 42}), "42")
}

func TestErrors(t *testing.T) {
	if _, err := Parse("{{#if x}}unclosed"); err == nil {
		t.Fatal("expected unclosed block error")
	}
	if _, err := Parse("{{/if}}"); err == nil {
		t.Fatal("expected unexpected close error")
	}
	if _, err := Render("{{> missing}}", nil); err == nil {
		t.Fatal("expected missing partial error")
	}
	if _, err := Render("{{nosuch x}}", nil); err == nil {
		t.Fatal("expected unknown helper error")
	}
}

func TestMustParsePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	MustParse("{{#each}}")
}

func TestEmptyTemplate(t *testing.T) {
	tmpl := New()
	out, err := tmpl.Render(nil)
	if err != nil || out != "" {
		t.Fatalf("empty template: out=%q err=%v", out, err)
	}
}

func TestEscapeHTMLFunc(t *testing.T) {
	eq(t, EscapeHTML("<>&\"'`="), "&lt;&gt;&amp;&quot;&#x27;&#x60;&#x3D;")
}

func TestBracketedSegment(t *testing.T) {
	data := map[string]interface{}{"m": map[string]interface{}{"a b": "spaced"}}
	eq(t, render(t, "{{m.[a b]}}", data), "spaced")
}

func TestChainedNestedWithParent(t *testing.T) {
	data := map[string]interface{}{
		"outer": "O",
		"list": []map[string]interface{}{
			{"inner": "I", "vals": []int{1, 2}},
		},
	}
	out := render(t, "{{#each list}}{{#each vals}}{{../inner}}{{this}}{{../../outer}} {{/each}}{{/each}}", data)
	eq(t, out, "I1O I2O ")
}

func TestIntKeyedMap(t *testing.T) {
	data := map[string]interface{}{"m": map[int]string{1: "one", 2: "two"}}
	eq(t, render(t, "{{m.1}}-{{m.2}}", data), "one-two")
	// iteration over an int-keyed map sorts keys deterministically
	eq(t, render(t, "{{#each m}}{{@key}}:{{this}};{{/each}}", data), "1:one;2:two;")
}

func TestSliceLength(t *testing.T) {
	eq(t, render(t, "{{xs.length}}", map[string]interface{}{"xs": []int{1, 2, 3}}), "3")
}

func TestTruthyKinds(t *testing.T) {
	eq(t, render(t, "{{#if n}}y{{else}}n{{/if}}", map[string]interface{}{"n": uint(0)}), "n")
	eq(t, render(t, "{{#if n}}y{{else}}n{{/if}}", map[string]interface{}{"n": uint(3)}), "y")
	eq(t, render(t, "{{#if f}}y{{else}}n{{/if}}", map[string]interface{}{"f": 0.0}), "n")
	eq(t, render(t, "{{#if s}}y{{else}}n{{/if}}", map[string]interface{}{"s": ""}), "n")
	var p *profile
	eq(t, render(t, "{{#if p}}y{{else}}n{{/if}}", map[string]interface{}{"p": p}), "n")
}

func TestFormatValueTypes(t *testing.T) {
	eq(t, render(t, "{{b}}", map[string]interface{}{"b": []byte("hi")}), "hi")
	eq(t, render(t, "{{e}}", map[string]interface{}{"e": errSample{}}), "boom")
	eq(t, render(t, "{{i}}", map[string]interface{}{"i": int64(-7)}), "-7")
	eq(t, render(t, "{{f}}", map[string]interface{}{"f": float32(1.5)}), "1.5")
	eq(t, render(t, "{{s}}", map[string]interface{}{"s": stringerSample{}}), "STR")
}

type errSample struct{}

func (errSample) Error() string { return "boom" }

type stringerSample struct{}

func (stringerSample) String() string { return "STR" }

func TestUnclosedCommentError(t *testing.T) {
	if _, err := Parse("{{! never ends"); err == nil {
		t.Fatal("expected unclosed comment error")
	}
	if _, err := Parse("{{{ unterminated"); err == nil {
		t.Fatal("expected unclosed triple error")
	}
	if _, err := Parse(`{{foo "unterminated}}`); err == nil {
		t.Fatal("expected unterminated string error")
	}
}

func TestNestedSubexpression(t *testing.T) {
	out := render(t, "{{#if (and (eq a 1) (gt b 2))}}ok{{/if}}",
		map[string]interface{}{"a": 1, "b": 5})
	eq(t, out, "ok")
}

func TestDefaultSectionObject(t *testing.T) {
	data := map[string]interface{}{"person": map[string]interface{}{"name": "Q"}}
	eq(t, render(t, "{{#person}}{{name}}{{/person}}", data), "Q")
	eq(t, render(t, "{{#missing}}x{{else}}none{{/missing}}", data), "none")
}
