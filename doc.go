// Package handlebars is a dependency-free Handlebars/Mustache templating engine
// implemented in pure Go. It compiles a template source into an abstract syntax
// tree via its own lexer and parser, then renders that tree against a data value
// using reflection. It does not wrap text/template; the Handlebars semantics are
// implemented directly.
//
// # Quick start
//
//	out, err := handlebars.Render("Hello {{name}}!", map[string]any{"name": "World"})
//	// out == "Hello World!"
//
// For repeated rendering, compile once and reuse:
//
//	t := handlebars.MustParse("{{#each items}}{{@index}}:{{this}} {{/each}}")
//	out := t.MustRender(map[string]any{"items": []string{"a", "b"}})
//	// out == "0:a 1:b "
//
// # Mustaches
//
// A double mustache interpolates and HTML-escapes a value:
//
//	{{expr}}
//
// A triple mustache, or the ampersand form, emits the value without escaping:
//
//	{{{expr}}}
//	{{& expr}}
//
// The escaping set matches Handlebars.js: & < > " ' ` and =.
//
// # Path expressions
//
// Paths address values within the current context:
//
//	foo            field or map key "foo"
//	foo.bar.baz    nested lookup
//	foo.0          slice/array index
//	this or .      the current context itself
//	../foo         a value in the parent (enclosing block) context
//	@index @key    data variables (see #each)
//	@root          the top-level context
//	[a b]          bracketed segment containing spaces or dots
//
// # Data model
//
// Templates render against Go maps (any string- or integer-keyed map), structs
// and slices/arrays. Struct fields are matched first by exact exported name,
// then by json struct tag, then case-insensitively. Pointers and interfaces are
// dereferenced automatically. Values are stringified as follows: strings as-is,
// booleans as "true"/"false", integers and floats via strconv (floats without a
// trailing ".0"), []byte as text, and anything implementing fmt.Stringer or
// error via those methods.
//
// Truthiness for if/unless/with and plain sections follows JavaScript: nil,
// false, 0, "" and empty collections are falsy; everything else is truthy.
//
// # Built-in block helpers
//
//	{{#if cond}}...{{else}}...{{/if}}
//	{{#unless cond}}...{{/unless}}
//	{{#each collection}}...{{else}}...{{/each}}
//	{{#with object}}...{{/with}}
//
// {{#each}} iterates slices, arrays and maps. Within the body it exposes the
// data variables @index, @key, @first and @last, and rebinds this/. to the
// current element. Map keys are visited in sorted order so output is
// deterministic. An {{else}} section renders when the collection is empty.
//
// Inverse sections are written {{^cond}}...{{/cond}} and render when cond is
// falsy; a bare {{else}} (or {{^}}) separates the main and inverse bodies of any
// block.
//
// Chained conditionals are written with {{else if}} (and {{else unless}} /
// {{else with}}):
//
//	{{#if a}}A{{else if b}}B{{else}}C{{/if}}
//
// # Block parameters
//
// Blocks may name the values they yield with an "as |a b|" clause. {{#each}}
// binds the element and the index (or key); {{#with}} binds the value:
//
//	{{#each items as |item index|}}{{index}}:{{item}}{{/each}}
//	{{#with obj as |o|}}{{o.field}}{{/with}}
//
// Block parameters are lexically scoped and shadow the surrounding context.
// Custom block helpers pass them via Options.FnWithBlockParams.
//
// # Data variables
//
// Within blocks the following @-data variables are available: @root (the
// top-level context), @index and @key (position/key in an #each), @first and
// @last (loop-edge flags), and @level (block nesting depth). A parent loop's
// data is reached with the ../ prefix, e.g. @../index. Data variables can be
// disabled at compile time with the NoData option.
//
// # Inline partials, partial blocks and decorators
//
// Inline partials are declared with the built-in inline decorator and are
// visible to the surrounding scope:
//
//	{{#*inline "row"}}[{{name}}]{{/inline}}
//	{{#each people}}{{> row}}{{/each}}
//
// A partial block passes its body to the invoked partial as @partial-block, the
// mechanism behind layout templates:
//
//	{{#> layout}}page body{{/layout}}   with layout = "<main>{{> @partial-block}}</main>"
//
// If the partial is undefined the block body renders as a fallback. Decorators
// are registered with RegisterDecorator and invoked as {{* name}} (inline) or
// {{#*name}}...{{/name}} (block); the built-in inline decorator implements
// inline partials.
//
// # Helper-missing hooks
//
// Registering a helper named "helperMissing" or "blockHelperMissing" intercepts
// references to unknown helpers and unknown block helpers respectively, matching
// the Handlebars.js hooks. Without a hook, a bare {{foo}} for an unknown name is
// empty, {{foo arg}} is an error, and {{#foo}} falls back to Mustache section
// behaviour.
//
// # Compile options
//
// Compile accepts options mirroring Handlebars.compile: NoEscape (emit {{expr}}
// unescaped), Strict (missing paths/helpers are errors), NoData (disable
// @-data), and KnownHelpers / KnownHelpersOnly (restrict which names are treated
// as helper calls).
//
// # Built-in inline helpers
//
// These are handy on their own and inside subexpressions: lookup, eq, ne, not,
// and, or, gt, lt, gte and lte. For example:
//
//	{{#if (eq status "active")}}on{{else}}off{{/if}}
//	{{lookup map key}}
//
// # Custom helpers
//
// Register helpers with RegisterHelper. A helper receives an *Options carrying
// the positional Args, the Hash (key=value) arguments, and, for block helpers,
// the Fn and Inverse callbacks that render the block body and its else section:
//
//	t.RegisterHelper("shout", func(o *handlebars.Options) any {
//	    return strings.ToUpper(handlebars.EscapeHTML(fmt.Sprint(o.Arg(0)))) + "!"
//	})
//
//	t.RegisterHelper("bold", func(o *handlebars.Options) any {
//	    return "<b>" + o.Fn(o.Arg(0)) + "</b>"
//	})
//
// Return a SafeString to bypass HTML escaping of a helper's result. Hash
// arguments are read via o.Hash or o.HashStr. Subexpressions such as
// {{outer (inner x) y}} evaluate the inner helper first and pass its result as
// an argument.
//
// # Partials
//
// Register reusable fragments with RegisterPartial and invoke them with the
// partial syntax:
//
//	{{> name}}            render partial "name" with the current context
//	{{> name context}}    render with an explicit context value
//	{{> name key=value}}  render with extra named values overlaid on the context
//	{{> (expr)}}          dynamic partial whose name is computed at render time
//
// Standalone partials preserve indentation: the whitespace preceding the tag is
// applied to every line of the partial's output.
//
// # Comments and whitespace control
//
// Comments produce no output and come in two forms:
//
//	{{! inline comment }}
//	{{!-- may contain }} and other mustaches --}}
//
// Block, comment and partial tags that occupy a line on their own have that
// line's surrounding whitespace stripped (the standalone rule). Explicit control
// is available with the tilde: {{~expr}} trims whitespace to the left and
// {{expr~}} trims to the right.
package handlebars
