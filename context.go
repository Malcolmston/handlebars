package handlebars

import (
	"reflect"
	"strconv"
)

// frame is one level of the render context stack. Each block that changes the
// context (each, with, default sections) pushes a new frame whose parent points
// at the enclosing scope, enabling ../ path traversal.
type frame struct {
	ctx    interface{}
	parent *frame
	data   map[string]interface{} // @-variables such as @index, @key, @root
}

// newRootFrame creates the base frame for a render, seeding @root.
func newRootFrame(ctx interface{}) *frame {
	f := &frame{ctx: ctx, data: map[string]interface{}{}}
	f.data["root"] = ctx
	return f
}

// child pushes a new frame with the given context, inheriting @root.
func (f *frame) child(ctx interface{}) *frame {
	data := map[string]interface{}{}
	if root, ok := f.dataVar("root"); ok {
		data["root"] = root
	}
	return &frame{ctx: ctx, parent: f, data: data}
}

// dataVar looks up an @-variable, walking up through parent frames.
func (f *frame) dataVar(name string) (interface{}, bool) {
	for cur := f; cur != nil; cur = cur.parent {
		if cur.data != nil {
			if v, ok := cur.data[name]; ok {
				return v, true
			}
		}
	}
	return nil, false
}

// resolvePath evaluates a path against the frame stack, returning the value and
// whether it was found.
func resolvePath(f *frame, p *Path) (interface{}, bool) {
	base := f
	for i := 0; i < p.Depth; i++ {
		if base.parent == nil {
			break
		}
		base = base.parent
	}

	if p.Data {
		// @root is special: it restarts path resolution from the root context.
		if len(p.Segments) > 0 && p.Segments[0] == "root" {
			root, _ := base.dataVar("root")
			return walk(root, p.Segments[1:])
		}
		if len(p.Segments) == 0 {
			return nil, false
		}
		v, ok := base.dataVar(p.Segments[0])
		if !ok {
			return nil, false
		}
		return walk(v, p.Segments[1:])
	}

	if p.This || len(p.Segments) == 0 {
		return base.ctx, true
	}
	return walk(base.ctx, p.Segments)
}

// walk descends through a sequence of segments starting at cur.
func walk(cur interface{}, segs []string) (interface{}, bool) {
	for _, seg := range segs {
		var ok bool
		cur, ok = lookup(cur, seg)
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// lookup resolves a single segment on a value, supporting maps, structs
// (by field name and json tag), slices/arrays (by index) and pointers.
func lookup(cur interface{}, seg string) (interface{}, bool) {
	if cur == nil {
		return nil, false
	}
	if ov, ok := cur.(*overlay); ok {
		if v, ok := ov.data[seg]; ok {
			return v, true
		}
		return lookup(ov.base, seg)
	}
	v := reflect.ValueOf(cur)
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		return lookupMap(v, seg)
	case reflect.Struct:
		return lookupStruct(v, seg)
	case reflect.Slice, reflect.Array:
		return lookupIndex(v, seg)
	default:
		return nil, false
	}
}

func lookupMap(v reflect.Value, seg string) (interface{}, bool) {
	kt := v.Type().Key()
	if kt.Kind() == reflect.String {
		mv := v.MapIndex(reflect.ValueOf(seg).Convert(kt))
		if mv.IsValid() {
			return mv.Interface(), true
		}
		return nil, false
	}
	// Non-string keys: allow integer keys addressed numerically.
	if n, err := strconv.Atoi(seg); err == nil && isIntKind(kt.Kind()) {
		mv := v.MapIndex(reflect.ValueOf(n).Convert(kt))
		if mv.IsValid() {
			return mv.Interface(), true
		}
	}
	return nil, false
}

func lookupStruct(v reflect.Value, seg string) (interface{}, bool) {
	t := v.Type()
	// Exact field name.
	if f, ok := t.FieldByName(seg); ok && f.PkgPath == "" {
		return v.FieldByIndex(f.Index).Interface(), true
	}
	// json tag, then case-insensitive field name.
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue // unexported
		}
		if tag := jsonName(sf.Tag.Get("json")); tag != "" && tag == seg {
			return v.Field(i).Interface(), true
		}
	}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if equalFold(sf.Name, seg) {
			return v.Field(i).Interface(), true
		}
	}
	return nil, false
}

func lookupIndex(v reflect.Value, seg string) (interface{}, bool) {
	n, err := strconv.Atoi(seg)
	if err != nil {
		if seg == "length" {
			return v.Len(), true
		}
		return nil, false
	}
	if n < 0 || n >= v.Len() {
		return nil, false
	}
	return v.Index(n).Interface(), true
}

func jsonName(tag string) string {
	if tag == "" {
		return ""
	}
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			return tag[:i]
		}
	}
	return tag
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func isIntKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	}
	return false
}

// isTruthy applies JavaScript-like truthiness used by if/unless/with and the
// default section behaviour: nil, false, 0, "", and empty collections are false.
func isTruthy(v interface{}) bool {
	if v == nil {
		return false
	}
	if _, ok := v.(*overlay); ok {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Bool:
		return rv.Bool()
	case reflect.String:
		return rv.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return rv.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return rv.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return rv.Float() != 0
	case reflect.Slice, reflect.Array, reflect.Map:
		return rv.Len() > 0
	case reflect.Ptr, reflect.Interface:
		return !rv.IsNil()
	default:
		return true
	}
}
