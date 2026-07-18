package handlebars

import (
	"reflect"
	"testing"
)

func TestRegisterHelpers(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelpers(map[string]Helper{
		"upper": func(o *Options) interface{} { return "U" },
		"lower": func(o *Options) interface{} { return "L" },
	})
	if !tmpl.HasHelper("upper") || !tmpl.HasHelper("lower") {
		t.Fatalf("bulk registration failed: %v", tmpl.HelperNames())
	}
	// A nil value unregisters.
	tmpl.RegisterHelpers(map[string]Helper{"upper": nil})
	if tmpl.HasHelper("upper") {
		t.Fatalf("nil helper should unregister")
	}
	if err := tmpl.ParseString("{{lower}}"); err != nil {
		t.Fatal(err)
	}
	if out := tmpl.MustRender(nil); out != "L" {
		t.Fatalf("render = %q, want L", out)
	}
}

func TestRegisterPartials(t *testing.T) {
	tmpl := New()
	if err := tmpl.RegisterPartials(map[string]string{
		"a": "A{{name}}",
		"b": "B{{name}}",
	}); err != nil {
		t.Fatal(err)
	}
	if !tmpl.HasPartial("a") || !tmpl.HasPartial("b") {
		t.Fatalf("partials missing: %v", tmpl.PartialNames())
	}
	if err := tmpl.ParseString("{{> a}}-{{> b}}"); err != nil {
		t.Fatal(err)
	}
	if out := tmpl.MustRender(map[string]any{"name": "X"}); out != "AX-BX" {
		t.Fatalf("render = %q, want AX-BX", out)
	}
	// Parse errors are reported but other partials still register.
	err := tmpl.RegisterPartials(map[string]string{
		"good": "ok",
		"bad":  "{{#if}}",
	})
	if err == nil {
		t.Fatalf("expected error for malformed partial")
	}
	if !tmpl.HasPartial("good") {
		t.Fatalf("good partial should have registered despite sibling error")
	}
}

func TestUnregister(t *testing.T) {
	tmpl := New()
	tmpl.RegisterHelper("h", func(o *Options) interface{} { return "" })
	_ = tmpl.RegisterPartial("p", "x")
	tmpl.RegisterDecorator("d", func(o *DecoratorOptions) {})

	tmpl.UnregisterHelper("h")
	tmpl.UnregisterPartial("p")
	tmpl.UnregisterDecorator("d")

	if tmpl.HasHelper("h") {
		t.Fatalf("helper not unregistered")
	}
	if tmpl.HasPartial("p") {
		t.Fatalf("partial not unregistered")
	}
	if _, ok := tmpl.decorators["d"]; ok {
		t.Fatalf("decorator not unregistered")
	}
	// Unregistering an absent name is a no-op.
	tmpl.UnregisterHelper("nope")
}

func TestHelperAndPartialNames(t *testing.T) {
	tmpl := New()
	_ = tmpl.RegisterPartials(map[string]string{"z": "z", "a": "a", "m": "m"})
	got := tmpl.PartialNames()
	want := []string{"a", "m", "z"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PartialNames = %v, want %v (sorted)", got, want)
	}
	// Built-in helpers are present and the list is sorted.
	names := tmpl.HelperNames()
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("HelperNames not sorted: %v", names)
		}
	}
	if !tmpl.HasHelper("eq") || !tmpl.HasHelper("lookup") {
		t.Fatalf("built-in helpers missing from %v", names)
	}
}

func TestClone(t *testing.T) {
	base := New()
	base.RegisterHelper("shared", func(o *Options) interface{} { return "base" })
	_ = base.RegisterPartial("p", "base-partial")
	if err := base.ParseString("{{shared}}"); err != nil {
		t.Fatal(err)
	}

	clone := base.Clone()
	// Clone inherits the base's registrations and program.
	if !clone.HasHelper("shared") || !clone.HasPartial("p") {
		t.Fatalf("clone did not inherit registrations")
	}
	if out := clone.MustRender(nil); out != "base" {
		t.Fatalf("clone render = %q, want base", out)
	}

	// Mutating the clone must not affect the base.
	clone.RegisterHelper("shared", func(o *Options) interface{} { return "clone" })
	clone.RegisterHelper("cloneonly", func(o *Options) interface{} { return "" })
	clone.UnregisterPartial("p")

	if base.MustRender(nil) != "base" {
		t.Fatalf("base program affected by clone mutation")
	}
	if base.HasHelper("cloneonly") {
		t.Fatalf("base gained clone-only helper")
	}
	if !base.HasPartial("p") {
		t.Fatalf("base lost partial after clone unregistered it")
	}

	// And mutating the base must not affect the clone.
	base.RegisterHelper("baseonly", func(o *Options) interface{} { return "" })
	if clone.HasHelper("baseonly") {
		t.Fatalf("clone gained base-only helper")
	}
}
