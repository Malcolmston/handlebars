package handlebars

import "log"

// Option configures a Template's compile behaviour. Options are passed to
// Compile and mirror the flags accepted by Handlebars.compile in Handlebars.js.
type Option func(*config)

// NoEscape disables HTML escaping of {{expr}} mustaches so their output is
// emitted verbatim (equivalent to Handlebars' noEscape: true).
func NoEscape() Option {
	return func(c *config) { c.noEscape = true }
}

// Strict enables strict mode: referencing a missing value or helper is an error
// instead of rendering the empty string (Handlebars' strict: true).
func Strict() Option {
	return func(c *config) { c.strict = true }
}

// NoData disables @-data variables such as @index and @root (Handlebars'
// data: false). Data variables are enabled by default.
func NoData() Option {
	return func(c *config) { c.data = false }
}

// KnownHelpers marks the given names as helpers known at compile time. Combined
// with KnownHelpersOnly it restricts which names are treated as helper calls.
func KnownHelpers(names ...string) Option {
	return func(c *config) {
		if c.knownHelpers == nil {
			c.knownHelpers = map[string]bool{}
		}
		for _, n := range names {
			c.knownHelpers[n] = true
		}
	}
}

// KnownHelpersOnly restricts helper resolution to the built-ins plus any names
// registered with KnownHelpers; every other identifier is treated as a path
// (Handlebars' knownHelpersOnly: true).
func KnownHelpersOnly() Option {
	return func(c *config) { c.knownHelpersOnly = true }
}

// SetLogger overrides the logger used by the built-in {{log}} helper. It returns
// the template for chaining. Passing nil restores the default stderr logger.
func (t *Template) SetLogger(l *log.Logger) *Template {
	t.logger = l
	return t
}

// Decorator is the signature for a decorator, registered with RegisterDecorator
// and invoked via {{* name}} or {{#*name}}...{{/name}}. The canonical example is
// the built-in inline decorator, which registers its block body as a partial.
type Decorator func(*DecoratorOptions)

// DecoratorOptions is passed to every decorator invocation. It exposes the
// positional Args, the Hash arguments, the decorator's block body (Program, nil
// for the inline {{* name}} form) and a means to register in-scope partials.
type DecoratorOptions struct {
	// Name is the decorator's registered name.
	Name string
	// Args holds the evaluated positional arguments.
	Args []interface{}
	// Hash holds the evaluated key=value arguments.
	Hash map[string]interface{}
	// Program is the decorator block's body, or nil for the inline form.
	Program *Program

	frame *frame
}

// Arg returns the positional argument at index i, or nil if out of range.
func (o *DecoratorOptions) Arg(i int) interface{} {
	if i < 0 || i >= len(o.Args) {
		return nil
	}
	return o.Args[i]
}

// RegisterPartial registers prog as an in-scope partial visible to the current
// block and its descendants. The built-in inline decorator uses this.
func (o *DecoratorOptions) RegisterPartial(name string, prog *Program) {
	o.frame.setPartial(name, &partialDef{prog: prog})
}

// RegisterDecorator registers (or replaces) a decorator by name. It returns the
// template for chaining.
func (t *Template) RegisterDecorator(name string, fn Decorator) *Template {
	t.decorators[name] = fn
	return t
}

// registerBuiltinDecorators installs the standard inline decorator, which
// implements {{#*inline "name"}}...{{/inline}} by registering the block body as
// an in-scope partial.
func registerBuiltinDecorators(t *Template) {
	t.decorators["inline"] = func(o *DecoratorOptions) {
		if o.Program == nil || len(o.Args) == 0 {
			return
		}
		o.RegisterPartial(formatValue(o.Arg(0)), o.Program)
	}
}
