package handlebars

import (
	"reflect"
	"testing"
)

func TestEscapeExpression(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil", nil, ""},
		{"plain", "hi there", "hi there"},
		{"html", `<a href="x">&'`, "&lt;a href&#x3D;&quot;x&quot;&gt;&amp;&#x27;"},
		{"safestring", SafeString("<b>raw</b>"), "<b>raw</b>"},
		{"int", 42, "42"},
		{"zero", 0, "0"},
		{"bool", false, "false"},
		{"float", 3.5, "3.5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := EscapeExpression(c.in); got != c.want {
				t.Fatalf("EscapeExpression(%v) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestStringify(t *testing.T) {
	cases := []struct {
		in   interface{}
		want string
	}{
		{nil, ""},
		{"abc", "abc"},
		{true, "true"},
		{7, "7"},
		{int64(-3), "-3"},
		{uint(9), "9"},
		{2.5, "2.5"},
		{[]byte("bytes"), "bytes"},
		{SafeString("<b>"), "<b>"},
	}
	for _, c := range cases {
		if got := Stringify(c.in); got != c.want {
			t.Fatalf("Stringify(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	var nilPtr *int
	x := 5
	cases := []struct {
		name string
		in   interface{}
		want bool
	}{
		{"nil", nil, true},
		{"false", false, true},
		{"true", true, false},
		{"empty string", "", true},
		{"string", "a", false},
		{"zero", 0, false},
		{"number", 3, false},
		{"empty slice", []int{}, true},
		{"slice", []int{1}, false},
		{"empty array", [0]int{}, true},
		{"empty map (not empty per handlebars)", map[string]int{}, false},
		{"map", map[string]int{"a": 1}, false},
		{"struct", struct{}{}, false},
		{"nil ptr", nilPtr, true},
		{"ptr", &x, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsEmpty(c.in); got != c.want {
				t.Fatalf("IsEmpty(%#v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestIsArray(t *testing.T) {
	cases := []struct {
		in   interface{}
		want bool
	}{
		{nil, false},
		{[]int{1, 2}, true},
		{[3]string{}, true},
		{map[string]int{}, false},
		{"str", false},
		{42, false},
	}
	for _, c := range cases {
		if got := IsArray(c.in); got != c.want {
			t.Fatalf("IsArray(%#v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExtend(t *testing.T) {
	a := map[string]interface{}{"x": 1, "y": 2}
	b := map[string]interface{}{"y": 3, "z": 4}
	got := Extend(a, b)
	want := map[string]interface{}{"x": 1, "y": 3, "z": 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Extend = %v, want %v", got, want)
	}
	// dst is mutated and returned.
	if !reflect.DeepEqual(a, want) {
		t.Fatalf("Extend did not mutate dst: %v", a)
	}
	// nil dst allocates a new map.
	n := Extend(nil, map[string]interface{}{"k": "v"})
	if n == nil || n["k"] != "v" {
		t.Fatalf("Extend(nil, ...) = %v", n)
	}
}

func TestCreateFrame(t *testing.T) {
	parent := map[string]interface{}{"root": "R"}
	frame := CreateFrame(parent)
	if frame["root"] != "R" {
		t.Fatalf("frame did not copy parent entries: %v", frame)
	}
	if !reflect.DeepEqual(frame["_parent"], parent) {
		t.Fatalf("frame._parent = %v, want %v", frame["_parent"], parent)
	}
	// Mutating the frame must not touch the parent (shallow copy of keys).
	frame["root"] = "changed"
	if parent["root"] != "R" {
		t.Fatalf("CreateFrame shared storage with parent")
	}
	// nil input yields a usable frame whose parent is a nil map.
	nf := CreateFrame(nil)
	if !reflect.DeepEqual(nf["_parent"], map[string]interface{}(nil)) {
		t.Fatalf("CreateFrame(nil)._parent = %#v, want nil map", nf["_parent"])
	}
}

func BenchmarkEscapeExpression(b *testing.B) {
	v := `Tom & Jerry <"quotes"> 'apostrophe' = sign`
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = EscapeExpression(v)
	}
}

func BenchmarkStringify(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Stringify(1234.5678)
	}
}
