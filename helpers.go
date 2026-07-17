package handlebars

import (
	"fmt"
	"reflect"
	"strings"
)

// registerBuiltins installs the standard inline helpers. The block helpers
// if/unless/each/with are handled directly by the renderer so they cannot be
// shadowed, but these inline helpers are useful on their own and inside
// subexpressions such as {{#if (eq a b)}}.
func registerBuiltins(t *Template) {
	t.helpers["lookup"] = helperLookup
	t.helpers["log"] = helperLog
	t.helpers["eq"] = helperEq
	t.helpers["ne"] = helperNe
	t.helpers["not"] = helperNot
	t.helpers["and"] = helperAnd
	t.helpers["or"] = helperOr
	t.helpers["gt"] = helperGt
	t.helpers["lt"] = helperLt
	t.helpers["gte"] = helperGte
	t.helpers["lte"] = helperLte
}

// helperLookup implements {{lookup obj key}}, resolving a dynamic key against a
// map, struct or slice.
func helperLookup(o *Options) interface{} {
	if len(o.Args) < 2 {
		return nil
	}
	v, _ := lookup(o.Args[0], formatValue(o.Args[1]))
	return v
}

// helperLog implements {{log msg ...}}: it writes its arguments to the
// template's logger (stderr by default, override with SetLogger) and produces no
// output. A level="..." hash argument is prefixed to the message.
func helperLog(o *Options) interface{} {
	parts := make([]string, len(o.Args))
	for i, a := range o.Args {
		parts[i] = formatValue(a)
	}
	msg := strings.Join(parts, " ")
	if lvl, ok := o.Hash["level"]; ok {
		msg = fmt.Sprintf("[%s] %s", formatValue(lvl), msg)
	}
	if o.r != nil && o.r.tmpl.logger != nil {
		o.r.tmpl.logger.Println(msg)
	}
	return ""
}

func helperEq(o *Options) interface{} {
	if len(o.Args) < 2 {
		return false
	}
	return compareEqual(o.Args[0], o.Args[1])
}

func helperNe(o *Options) interface{} {
	if len(o.Args) < 2 {
		return true
	}
	return !compareEqual(o.Args[0], o.Args[1])
}

func helperNot(o *Options) interface{} {
	if len(o.Args) < 1 {
		return true
	}
	return !isTruthy(o.Args[0])
}

func helperAnd(o *Options) interface{} {
	for _, a := range o.Args {
		if !isTruthy(a) {
			return false
		}
	}
	return true
}

func helperOr(o *Options) interface{} {
	for _, a := range o.Args {
		if isTruthy(a) {
			return true
		}
	}
	return false
}

func helperGt(o *Options) interface{}  { return cmpNumbers(o) > 0 }
func helperLt(o *Options) interface{}  { return cmpNumbers(o) < 0 }
func helperGte(o *Options) interface{} { return cmpNumbers(o) >= 0 }
func helperLte(o *Options) interface{} { return cmpNumbers(o) <= 0 }

// cmpNumbers returns -1, 0 or 1 comparing the first two arguments numerically.
func cmpNumbers(o *Options) int {
	if len(o.Args) < 2 {
		return 0
	}
	a, aok := toFloat(o.Args[0])
	b, bok := toFloat(o.Args[1])
	if !aok || !bok {
		return 0
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// compareEqual compares two values, treating all numeric kinds uniformly.
func compareEqual(a, b interface{}) bool {
	if af, aok := toFloat(a); aok {
		if bf, bok := toFloat(b); bok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

// toFloat converts any numeric value to a float64.
func toFloat(v interface{}) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}
