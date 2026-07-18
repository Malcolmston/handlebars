package handlebars

import "reflect"

// EscapeExpression renders a value the way Handlebars.js Utils.escapeExpression
// does: a SafeString is returned verbatim, nil becomes the empty string, and any
// other value is stringified (using the same rules as {{expr}} output) and then
// HTML-escaped with the Handlebars character set. Unlike EscapeHTML, which takes
// an already-stringified argument, EscapeExpression accepts any value and
// honours the SafeString opt-out, making it the correct choice for helpers that
// build HTML from untrusted values.
func EscapeExpression(v interface{}) string {
	switch s := v.(type) {
	case nil:
		return ""
	case SafeString:
		return string(s)
	}
	return EscapeHTML(Stringify(v))
}

// Stringify converts a value to the exact string representation the renderer
// would emit for {{{expr}}} (before any HTML escaping). Strings pass through
// unchanged, booleans become "true"/"false", integers and floats are formatted
// via strconv (floats without a trailing ".0"), []byte is treated as text, and
// values implementing fmt.Stringer or error use those methods. It exposes the
// engine's canonical stringification so helper authors can match template output
// exactly.
func Stringify(v interface{}) string {
	return formatValue(v)
}

// IsEmpty reports whether a value is "empty" using the same definition as
// Handlebars.js Utils.isEmpty, which drives the falsy branch of built-in block
// helpers. nil, false and the empty string are empty; an empty slice or array is
// empty; every number (including 0), every non-empty collection and every map or
// struct is not empty. Note that, matching Handlebars, an empty map is NOT
// considered empty. Pointers and interfaces are dereferenced first.
func IsEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return !rv.Bool()
	case reflect.String:
		return rv.Len() == 0
	case reflect.Slice, reflect.Array:
		return rv.Len() == 0
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return true
		}
		return IsEmpty(rv.Elem().Interface())
	default:
		return false
	}
}

// IsArray reports whether a value is a Go slice or array, mirroring
// Handlebars.js Utils.isArray. It returns false for nil, maps, strings and every
// scalar type.
func IsArray(v interface{}) bool {
	if v == nil {
		return false
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Slice, reflect.Array:
		return true
	default:
		return false
	}
}

// Extend copies the entries of each source map into dst and returns dst,
// mirroring Handlebars.js Utils.extend. Later sources overwrite earlier ones on
// key collisions. If dst is nil a new map is allocated and returned, so the
// result is always safe to use.
func Extend(dst map[string]interface{}, sources ...map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = map[string]interface{}{}
	}
	for _, src := range sources {
		for k, v := range src {
			dst[k] = v
		}
	}
	return dst
}

// CreateFrame builds a new data frame from an existing one, mirroring
// Handlebars.js Utils.createFrame. The returned map is a shallow copy of data
// with an added "_parent" entry pointing at the original, which is the mechanism
// Handlebars uses to layer @-variables (such as @index) onto a child scope while
// keeping the parent frame reachable. Passing nil yields a frame whose "_parent"
// is nil.
func CreateFrame(data map[string]interface{}) map[string]interface{} {
	frame := Extend(nil, data)
	frame["_parent"] = data
	return frame
}
