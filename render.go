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
	return o.fnProg(o.mainProg(), ctx, data, nil)
}

// FnWithBlockParams renders the block body binding the block's declared "as |a
// b|" parameters to params, in declaration order.
func (o *Options) FnWithBlockParams(ctx interface{}, params ...interface{}) string {
	return o.fnProg(o.mainProg(), ctx, nil, params)
}

// BlockParams returns the names declared by the block's "as |a b|" clause.
func (o *Options) BlockParams() []string {
	if o.block == nil {
		return nil
	}
	return o.block.Params
}

func (o *Options) mainProg() *Program {
	if o.block == nil {
		return nil
	}
	return o.block.mainProg()
}

func (o *Options) fnProg(prog *Program, ctx interface{}, data map[string]interface{}, params []interface{}) string {
	if prog == nil {
		return ""
	}
	child := o.frame.child(ctx)
	for k, v := range data {
		child.data[k] = v
	}
	if len(params) > 0 && o.block != nil && len(o.block.Params) > 0 {
		child.blockParams = map[string]interface{}{}
		for i, name := range o.block.Params {
			if i < len(params) {
				child.blockParams[name] = params[i]
			}
		}
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
	case *DecoratorNode:
		r.renderDecorator(node, f)
	}
}

// renderDecorator evaluates a decorator invocation ({{* name}} or the block form
// {{#*name}}...{{/name}}). Decorators produce no direct output; they mutate the
// surrounding scope, e.g. the built-in inline decorator registers a partial.
func (r *renderer) renderDecorator(node *DecoratorNode, f *frame) {
	name := blockName(node.Expr)
	dec := r.tmpl.decorators[name]
	if dec == nil {
		r.setErr(fmt.Errorf("handlebars: unknown decorator %q", name))
		return
	}
	args, hash := r.evalArgs(f, node.Expr)
	dec(&DecoratorOptions{Name: name, Args: args, Hash: hash, Program: node.Program, frame: f})
}

func (r *renderer) renderMustache(node *MustacheNode, f *frame) {
	v := r.evalCallee(f, node.Expr)
	if node.Unescaped || r.tmpl.cfg.noEscape {
		r.buf.WriteString(formatValue(v))
		return
	}
	if ss, ok := v.(SafeString); ok {
		r.buf.WriteString(string(ss))
		return
	}
	r.buf.WriteString(EscapeHTML(formatValue(v)))
}

// lookupHelper returns the helper registered under name, honouring the
// knownHelpersOnly compile option.
func (r *renderer) lookupHelper(name string) Helper {
	if name == "" {
		return nil
	}
	h := r.tmpl.helpers[name]
	if h == nil {
		return nil
	}
	if r.tmpl.cfg.knownHelpersOnly && !r.knownHelper(name) {
		return nil
	}
	return h
}

// knownHelper reports whether name is treated as a helper under knownHelpersOnly.
func (r *renderer) knownHelper(name string) bool {
	if r.tmpl.cfg.knownHelpers[name] {
		return true
	}
	return defaultKnownHelpers[name]
}

// defaultKnownHelpers are the names Handlebars always treats as helpers, used
// when the knownHelpersOnly option is set.
var defaultKnownHelpers = map[string]bool{
	"helperMissing": true, "blockHelperMissing": true, "each": true, "if": true,
	"unless": true, "with": true, "log": true, "lookup": true,
}

// evalCallee evaluates a mustache or section head expression. A bare string or
// number literal in this leading position is resolved as a path lookup by that
// literal's key (Handlebars.js "literal references", e.g. {{"foo bar"}} looks up
// the "foo bar" property), whereas the same literal used as an argument keeps
// its literal value. All other expressions defer to evalExpr.
func (r *renderer) evalCallee(f *frame, e *Expr) interface{} {
	if len(e.Params) == 0 && len(e.Hash) == 0 {
		var key string
		switch e.Kind {
		case exprString:
			key = e.Str
		case exprNumber:
			key = strconv.FormatFloat(e.Num, 'f', -1, 64)
		default:
			return r.evalExpr(f, e)
		}
		v, _ := resolvePath(f, &Path{Segments: []string{key}, Original: key})
		return v
	}
	return r.evalExpr(f, e)
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
		name := blockName(e)
		if len(e.Params) > 0 || len(e.Hash) > 0 {
			if r.lookupHelper(name) == nil {
				return r.helperMissing(f, e, name)
			}
			return r.callInlineHelper(f, e)
		}
		if r.lookupHelper(name) != nil {
			return r.callInlineHelper(f, e)
		}
		if e.Path.Data && !r.tmpl.cfg.data {
			return nil
		}
		v, ok := resolvePath(f, e.Path)
		if !ok {
			// An ambiguous bare mustache ({{name}}) whose name is neither a
			// registered helper nor found in the context invokes a custom
			// helperMissing hook if one is registered, matching Handlebars.js.
			// The default (no hook) still renders the empty string.
			if hook := r.tmpl.helpers["helperMissing"]; hook != nil && isAmbiguousName(e.Path) {
				return hook(&Options{Name: name, r: r, frame: f})
			}
			if r.tmpl.cfg.strict && !e.Path.Data {
				r.setErr(fmt.Errorf("handlebars: %q not defined in strict mode", e.Path.Original))
			}
		}
		return v
	}
}

// isAmbiguousName reports whether a path is a single, bare identifier (no ".",
// no "../", no leading "@" and not "this"). Only such names occupy the
// helper-or-context "ambiguous" position that triggers helperMissing in
// Handlebars.js; dotted or data paths that resolve to nothing stay empty.
func isAmbiguousName(p *Path) bool {
	return p != nil && !p.Data && !p.This && p.Depth == 0 && len(p.Segments) == 1
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
	h := r.lookupHelper(name)
	if h == nil {
		return r.helperMissing(f, e, name)
	}
	args, hash := r.evalArgs(f, e)
	o := &Options{Name: name, Args: args, Hash: hash, r: r, frame: f}
	return h(o)
}

// helperMissing handles a call to a name that is not a registered helper. If a
// "helperMissing" hook is registered it is invoked; otherwise the default
// behaviour applies: a bare {{name}} resolves as a path (empty if absent) while
// a call with arguments is an error, matching Handlebars.js.
func (r *renderer) helperMissing(f *frame, e *Expr, name string) interface{} {
	args, hash := r.evalArgs(f, e)
	if hook := r.tmpl.helpers["helperMissing"]; hook != nil {
		o := &Options{Name: name, Args: args, Hash: hash, r: r, frame: f}
		return hook(o)
	}
	if len(args) == 0 && len(hash) == 0 {
		v, ok := resolvePath(f, e.Path)
		if !ok && r.tmpl.cfg.strict {
			r.setErr(fmt.Errorf("handlebars: %q not defined in strict mode", e.Path.Original))
		}
		return v
	}
	r.setErr(fmt.Errorf("handlebars: missing helper %q", name))
	return nil
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
	if h := r.lookupHelper(name); h != nil {
		args, hash := r.evalArgs(f, node.Expr)
		o := &Options{Name: name, Args: args, Hash: hash, r: r, frame: f, block: node}
		r.buf.WriteString(formatValue(h(o)))
		return
	}
	r.blockHelperMissing(node, f)
}

// blockHelperMissing handles {{#name}} where name is not a registered block
// helper. A "blockHelperMissing" hook takes precedence; otherwise the default
// Mustache section behaviour applies.
func (r *renderer) blockHelperMissing(node *BlockNode, f *frame) {
	if hook := r.tmpl.helpers["blockHelperMissing"]; hook != nil {
		v := r.evalExpr(f, node.Expr)
		o := &Options{Name: blockName(node.Expr), Args: []interface{}{v}, r: r, frame: f, block: node}
		r.buf.WriteString(formatValue(hook(o)))
		return
	}
	r.builtinSection(node, f)
}

func (r *renderer) builtinIf(node *BlockNode, f *frame, negate bool) {
	var cond bool
	if len(node.Expr.Params) > 0 {
		v := r.evalExpr(f, node.Expr.Params[0])
		cond = blockTruthy(v)
		// With includeZero=true a numeric zero counts as truthy, matching the
		// Handlebars.js #if helper's includeZero hash option.
		if !cond && !negate && isZeroNumber(v) && r.hashFlag(f, node.Expr, "includeZero") {
			cond = true
		}
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

// hashFlag reports whether the named hash argument of e is present and truthy.
func (r *renderer) hashFlag(f *frame, e *Expr, key string) bool {
	for _, hp := range e.Hash {
		if hp.Key == key {
			return isTruthy(r.evalExpr(f, hp.Value))
		}
	}
	return false
}

// isZeroNumber reports whether v is a numeric zero (any Go integer or float
// kind). It is used by #if includeZero handling.
func isZeroNumber(v interface{}) bool {
	switch t := v.(type) {
	case float64:
		return t == 0
	case float32:
		return t == 0
	case int:
		return t == 0
	case int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return rv.Int() == 0
		default:
			return rv.Uint() == 0
		}
	}
	return false
}

func (r *renderer) builtinWith(node *BlockNode, f *frame) {
	var v interface{}
	if len(node.Expr.Params) > 0 {
		v = r.evalExpr(f, node.Expr.Params[0])
	}
	if blockTruthy(v) {
		child := f.child(v)
		bindBlockParams(child, node.Params, v)
		r.renderProgram(node.mainProg(), child)
	} else {
		r.renderProgram(node.elseProg(), f)
	}
}

func (r *renderer) builtinEach(node *BlockNode, f *frame) {
	var coll interface{}
	if len(node.Expr.Params) > 0 {
		coll = r.evalExpr(f, node.Expr.Params[0])
	}
	if !r.iterateBlock(coll, node, f) {
		r.renderProgram(node.elseProg(), f)
	}
}

// bindBlockParams binds a block's "as |a b|" names on a child frame. The first
// name receives the value and the second (if any) the index or key.
func bindBlockParams(child *frame, names []string, value interface{}, extra ...interface{}) {
	if len(names) == 0 {
		return
	}
	child.blockParams = map[string]interface{}{}
	child.blockParams[names[0]] = value
	for i, ex := range extra {
		if i+1 < len(names) {
			child.blockParams[names[i+1]] = ex
		}
	}
}

// builtinSection implements plain {{#foo}} / {{^foo}} sections: iterate over
// collections, descend into truthy objects, or render the inverse otherwise.
func (r *renderer) builtinSection(node *BlockNode, f *frame) {
	v := r.evalCallee(f, node.Expr)
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

// iterate ranges over a slice/array/map without block parameters. It is used by
// the default section behaviour for plain {{#collection}} blocks.
func (r *renderer) iterate(coll interface{}, prog *Program, f *frame) bool {
	return r.iterateBlock(coll, &BlockNode{Program: prog}, f)
}

// iterateBlock ranges over a slice/array/map, exposing @index/@key/@first/@last
// and any declared block parameters, and reports whether anything was rendered
// (false for empty/non-iterable values).
func (r *renderer) iterateBlock(coll interface{}, node *BlockNode, f *frame) bool {
	if coll == nil {
		return false
	}
	prog := node.mainProg()
	rv := reflect.ValueOf(coll)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		if n == 0 {
			return false
		}
		for i := 0; i < n; i++ {
			elem := rv.Index(i).Interface()
			child := f.child(elem)
			child.data["index"] = i
			child.data["key"] = i
			child.data["first"] = i == 0
			child.data["last"] = i == n-1
			bindBlockParams(child, node.Params, elem, i)
			r.renderProgram(prog, child)
		}
		return true
	case reflect.Map:
		keys := sortedMapKeys(rv)
		if len(keys) == 0 {
			return false
		}
		for i, k := range keys {
			elem := rv.MapIndex(k).Interface()
			child := f.child(elem)
			child.data["index"] = i
			child.data["key"] = k.Interface()
			child.data["first"] = i == 0
			child.data["last"] = i == len(keys)-1
			bindBlockParams(child, node.Params, elem, k.Interface())
			r.renderProgram(prog, child)
		}
		return true
	default:
		return false
	}
}

func (r *renderer) renderPartial(node *PartialNode, f *frame) {
	name := r.partialName(f, node)

	// Resolve the partial program: @partial-block, in-scope inline / block
	// partials, then the template's global registry.
	prog, defFrame := r.resolvePartial(f, name)
	if prog == nil {
		// A partial block ({{#> layout}}body{{/layout}}) whose partial is missing
		// falls back to rendering its own body.
		if node.Program != nil {
			r.buf.WriteString(r.renderProgramToString(node.Program, f))
			return
		}
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

	child := f.child(ctx)
	// A partial block makes its body available as @partial-block inside the
	// invoked partial, rendered against the calling context.
	if node.Program != nil {
		child.setPartial("@partial-block", &partialDef{prog: node.Program, frame: f})
	}
	// Captured @partial-block content renders against its original frame.
	if defFrame != nil {
		child = defFrame.child(defFrame.ctx)
	}

	out := r.renderProgramToString(prog, child)
	if node.Indent != "" {
		out = indentLines(out, node.Indent)
	}
	r.buf.WriteString(out)
}

// resolvePartial finds a partial by name, checking @partial-block and in-scope
// inline/partial-block partials before the template's global registry. It
// returns the program and, for captured @partial-block content, the frame it
// should render against (nil otherwise).
func (r *renderer) resolvePartial(f *frame, name string) (*Program, *frame) {
	if def, ok := f.localPartial(name); ok {
		return def.prog, def.frame
	}
	if prog, ok := r.tmpl.partials[name]; ok {
		return prog, nil
	}
	return nil, nil
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
