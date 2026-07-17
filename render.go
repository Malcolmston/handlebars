package handlebars

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// SafeString marks a string returned by a helper as already-escaped so the
// renderer emits it verbatim instead of HTML-escaping it.
type SafeString string

// Options is passed to every helper invocation. It exposes the positional
// arguments, the hash (named) arguments, and, for block helpers, the means to
// render the block body (Fn) and its inverse/else section (Inverse).
type Options struct {
	// Name is the helper's registered name.
	Name string
	// Args holds the evaluated positional arguments.
	Args []interface{}
	// Hash holds the evaluated key=value arguments.
	Hash map[string]interface{}

	r     *renderer
	frame *frame
	block *BlockNode
}

// Arg returns the positional argument at index i, or nil if out of range.
func (o *Options) Arg(i int) interface{} {
	if i < 0 || i >= len(o.Args) {
		return nil
	}
	return o.Args[i]
}

// HashStr returns the hash argument key as a string, or def if absent.
func (o *Options) HashStr(key, def string) string {
	if v, ok := o.Hash[key]; ok {
		return formatValue(v)
	}
	return def
}

// Fn renders the block body with ctx as the new current context. For a plain
// helper (non-block) it returns the empty string.
func (o *Options) Fn(ctx interface{}) string {
	return o.FnWith(ctx, nil)
}

// FnWith renders the block body with ctx as context and the supplied
// @-variables layered onto the frame's data.
func (o *Options) FnWith(ctx interface{}, data map[string]interface{}) string {
	if o.block == nil {
		return ""
	}
	prog := o.block.mainProg()
	if prog == nil {
		return ""
	}
	child := o.frame.child(ctx)
	for k, v := range data {
		child.data[k] = v
	}
	return o.r.renderProgramToString(prog, child)
}

// Inverse renders the block's else/inverse section with ctx as context.
func (o *Options) Inverse(ctx interface{}) string {
	if o.block == nil {
		return ""
	}
	prog := o.block.elseProg()
	if prog == nil {
		return ""
	}
	child := o.frame.child(ctx)
	return o.r.renderProgramToString(prog, child)
}

// mainProg / elseProg apply the {{^inverse}} swap so helpers can be written
// naively (render main when truthy, else when falsy).
func (b *BlockNode) mainProg() *Program {
	if b.Inverted {
		return b.Inverse
	}
	return b.Program
}

func (b *BlockNode) elseProg() *Program {
	if b.Inverted {
		return b.Program
	}
	return b.Inverse
}

// overlay layers a set of named values on top of a base context. It is used for
// partial hash arguments (e.g. {{> row label="Name"}}).
type overlay struct {
	base interface{}
	data map[string]interface{}
}

// renderer walks a parsed program and accumulates output.
type renderer struct {
	tmpl *Template
	buf  *strings.Builder
	err  error
}

func (r *renderer) setErr(err error) {
	if r.err == nil {
		r.err = err
	}
}

// render evaluates the whole program against the root context.
func (r *renderer) render(prog *Program, ctx interface{}) (string, error) {
	root := newRootFrame(ctx)
	r.renderProgram(prog, root)
	return r.buf.String(), r.err
}

func (r *renderer) renderProgram(prog *Program, f *frame) {
	if prog == nil {
		return
	}
	for _, n := range prog.Body {
		r.renderNode(n, f)
	}
}

// renderProgramToString renders a program into an isolated buffer, used by
// helper Fn/Inverse callbacks and partials.
func (r *renderer) renderProgramToString(prog *Program, f *frame) string {
	saved := r.buf
	r.buf = &strings.Builder{}
	r.renderProgram(prog, f)
	out := r.buf.String()
	r.buf = saved
	return out
}

func (r *renderer) renderNode(n Node, f *frame) {
	switch node := n.(type) {
	case *ContentNode:
		r.buf.WriteString(node.Value)
	case *CommentNode:
		// no output
	case *MustacheNode:
		r.renderMustache(node, f)
	case *BlockNode:
		r.renderBlock(node, f)
	case *PartialNode:
		r.renderPartial(node, f)
	}
}

func (r *renderer) renderMustache(node *MustacheNode, f *frame) {
	v := r.evalExpr(f, node.Expr)
	if node.Unescaped {
		r.buf.WriteString(formatValue(v))
		return
	}
	if ss, ok := v.(SafeString); ok {
		r.buf.WriteString(string(ss))
		return
	}
	r.buf.WriteString(EscapeHTML(formatValue(v)))
}

// evalExpr evaluates an expression to a Go value in a value context.
func (r *renderer) evalExpr(f *frame, e *Expr) interface{} {
	switch e.Kind {
	case exprString:
		return e.Str
	case exprNumber:
		return e.Num
	case exprBool:
		return e.Bool
	case exprSubexpr:
		return r.callInlineHelper(f, e)
	default: // exprPath
		if len(e.Params) > 0 || len(e.Hash) > 0 {
			return r.callInlineHelper(f, e)
		}
		name := blockName(e)
		if name != "" && r.tmpl.helpers[name] != nil {
			return r.callInlineHelper(f, e)
		}
		v, _ := resolvePath(f, e.Path)
		return v
	}
}

func (r *renderer) evalArgs(f *frame, e *Expr) ([]interface{}, map[string]interface{}) {
	args := make([]interface{}, len(e.Params))
	for i, p := range e.Params {
		args[i] = r.evalExpr(f, p)
	}
	var hash map[string]interface{}
	if len(e.Hash) > 0 {
		hash = make(map[string]interface{}, len(e.Hash))
		for _, hp := range e.Hash {
			hash[hp.Key] = r.evalExpr(f, hp.Value)
		}
	}
	return args, hash
}

// callInlineHelper invokes a helper (or subexpression) in a value context.
func (r *renderer) callInlineHelper(f *frame, e *Expr) interface{} {
	name := blockName(e)
	h := r.tmpl.helpers[name]
	if h == nil {
		r.setErr(fmt.Errorf("handlebars: unknown helper %q", name))
		return nil
	}
	args, hash := r.evalArgs(f, e)
	o := &Options{Name: name, Args: args, Hash: hash, r: r, frame: f}
	return h(o)
}

// renderBlock dispatches a block node to a built-in or custom helper, or to the
// default Mustache section behaviour.
func (r *renderer) renderBlock(node *BlockNode, f *frame) {
	name := blockName(node.Expr)
	switch name {
	case "if":
		r.builtinIf(node, f, false)
		return
	case "unless":
		r.builtinIf(node, f, true)
		return
	case "each":
		r.builtinEach(node, f)
		return
	case "with":
		r.builtinWith(node, f)
		return
	}
	if h := r.tmpl.helpers[name]; h != nil && name != "" {
		args, hash := r.evalArgs(f, node.Expr)
		o := &Options{Name: name, Args: args, Hash: hash, r: r, frame: f, block: node}
		r.buf.WriteString(formatValue(h(o)))
		return
	}
	r.builtinSection(node, f)
}

func (r *renderer) builtinIf(node *BlockNode, f *frame, negate bool) {
	var cond bool
	if len(node.Expr.Params) > 0 {
		cond = isTruthy(r.evalExpr(f, node.Expr.Params[0]))
	}
	if negate {
		cond = !cond
	}
	if cond {
		r.renderProgram(node.mainProg(), f)
	} else {
		r.renderProgram(node.elseProg(), f)
	}
}

func (r *renderer) builtinWith(node *BlockNode, f *frame) {
	var v interface{}
	if len(node.Expr.Params) > 0 {
		v = r.evalExpr(f, node.Expr.Params[0])
	}
	if isTruthy(v) {
		r.renderProgram(node.mainProg(), f.child(v))
	} else {
		r.renderProgram(node.elseProg(), f)
	}
}

func (r *renderer) builtinEach(node *BlockNode, f *frame) {
	var coll interface{}
	if len(node.Expr.Params) > 0 {
		coll = r.evalExpr(f, node.Expr.Params[0])
	}
	if !r.iterate(coll, node.mainProg(), f) {
		r.renderProgram(node.elseProg(), f)
	}
}

// builtinSection implements plain {{#foo}} / {{^foo}} sections: iterate over
// collections, descend into truthy objects, or render the inverse otherwise.
func (r *renderer) builtinSection(node *BlockNode, f *frame) {
	v := r.evalExpr(f, node.Expr)
	rv := reflect.ValueOf(v)
	if v != nil && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
		if !r.iterate(v, node.mainProg(), f) {
			r.renderProgram(node.elseProg(), f)
		}
		return
	}
	if isTruthy(v) {
		r.renderProgram(node.mainProg(), f.child(v))
	} else {
		r.renderProgram(node.elseProg(), f)
	}
}

// iterate ranges over a slice/array/map, exposing @index/@key/@first/@last, and
// reports whether anything was rendered (false for empty/non-iterable values).
func (r *renderer) iterate(coll interface{}, prog *Program, f *frame) bool {
	if coll == nil {
		return false
	}
	rv := reflect.ValueOf(coll)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		if n == 0 {
			return false
		}
		for i := 0; i < n; i++ {
			child := f.child(rv.Index(i).Interface())
			child.data["index"] = i
			child.data["key"] = i
			child.data["first"] = i == 0
			child.data["last"] = i == n-1
			r.renderProgram(prog, child)
		}
		return true
	case reflect.Map:
		keys := sortedMapKeys(rv)
		if len(keys) == 0 {
			return false
		}
		for i, k := range keys {
			child := f.child(rv.MapIndex(k).Interface())
			child.data["index"] = i
			child.data["key"] = k.Interface()
			child.data["first"] = i == 0
			child.data["last"] = i == len(keys)-1
			r.renderProgram(prog, child)
		}
		return true
	default:
		return false
	}
}

func (r *renderer) renderPartial(node *PartialNode, f *frame) {
	name := r.partialName(f, node)
	prog, ok := r.tmpl.partials[name]
	if !ok {
		r.setErr(fmt.Errorf("handlebars: unknown partial %q", name))
		return
	}
	ctx := f.ctx
	if node.Context != nil {
		ctx = r.evalExpr(f, node.Context)
	}
	if len(node.Hash) > 0 {
		data := make(map[string]interface{}, len(node.Hash))
		for _, hp := range node.Hash {
			data[hp.Key] = r.evalExpr(f, hp.Value)
		}
		ctx = &overlay{base: ctx, data: data}
	}
	out := r.renderProgramToString(prog, f.child(ctx))
	if node.Indent != "" {
		out = indentLines(out, node.Indent)
	}
	r.buf.WriteString(out)
}

func (r *renderer) partialName(f *frame, node *PartialNode) string {
	switch node.Name.Kind {
	case exprString:
		return node.Name.Str
	case exprSubexpr:
		return formatValue(r.callInlineHelper(f, node.Name))
	default:
		if node.Name.Path != nil {
			return node.Name.Path.Original
		}
		return ""
	}
}

// blockName returns the callee name of a path expression, or "" if the callee
// is not a simple single-segment name.
func blockName(e *Expr) string {
	if e.Kind != exprPath && e.Kind != exprSubexpr {
		return ""
	}
	p := e.Path
	if p == nil || p.Data || p.Depth > 0 || p.This || len(p.Segments) != 1 {
		return ""
	}
	return p.Segments[0]
}

func sortedMapKeys(rv reflect.Value) []reflect.Value {
	keys := rv.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprint(keys[i].Interface()) < fmt.Sprint(keys[j].Interface())
	})
	return keys
}

func indentLines(s, indent string) string {
	if s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = indent + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// formatValue converts a value to its string representation for output.
func formatValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case SafeString:
		return string(t)
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.Itoa(t)
	case int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case uint, uint8, uint16, uint32, uint64:
		return strconv.FormatUint(reflect.ValueOf(v).Uint(), 10)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case []byte:
		return string(t)
	case fmt.Stringer:
		return t.String()
	case error:
		return t.Error()
	default:
		return fmt.Sprintf("%v", v)
	}
}
