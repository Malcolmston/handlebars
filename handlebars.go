package handlebars

import (
	"log"
	"os"
	"strings"
)

// Helper is the signature for all helpers, both inline and block. The return
// value is stringified for output; return a SafeString to bypass HTML escaping.
// Block helpers render their body via the Options.Fn and Options.Inverse
// callbacks.
type Helper func(*Options) interface{}

// Template is a compiled Handlebars template together with its own registry of
// helpers, partials and decorators. Templates are safe for concurrent use once
// compiled and fully configured (register helpers, partials and options before
// rendering).
type Template struct {
	prog       *Program
	helpers    map[string]Helper
	partials   map[string]*Program
	decorators map[string]Decorator
	cfg        config
	logger     *log.Logger
}

// config holds the compile-time options that mirror Handlebars.js compile flags.
type config struct {
	noEscape         bool            // do not HTML-escape {{expr}} output
	strict           bool            // missing paths/helpers are errors
	data             bool            // enable @-data variables (default true)
	knownHelpers     map[string]bool // helper names known at compile time
	knownHelpersOnly bool            // treat only knownHelpers as helpers
}

// New creates an empty Template pre-populated with the built-in helpers and
// decorators. Use Parse to attach a source, or compile in one step with the
// package-level Parse.
func New() *Template {
	t := &Template{
		helpers:    map[string]Helper{},
		partials:   map[string]*Program{},
		decorators: map[string]Decorator{},
		cfg:        config{data: true, knownHelpers: map[string]bool{}},
		logger:     log.New(os.Stderr, "", log.LstdFlags),
	}
	registerBuiltins(t)
	registerBuiltinDecorators(t)
	return t
}

// Parse compiles source into a new Template that includes the built-in helpers.
func Parse(source string) (*Template, error) {
	t := New()
	if err := t.ParseString(source); err != nil {
		return nil, err
	}
	return t, nil
}

// Compile is like Parse but applies the given compile options (NoEscape, Strict,
// KnownHelpers and so on), mirroring Handlebars.compile's options argument.
func Compile(source string, opts ...Option) (*Template, error) {
	t := New()
	for _, opt := range opts {
		opt(&t.cfg)
	}
	if err := t.ParseString(source); err != nil {
		return nil, err
	}
	return t, nil
}

// MustParse is like Parse but panics on error. It is convenient for templates
// compiled from constant strings at start-up.
func MustParse(source string) *Template {
	t, err := Parse(source)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseString compiles source and stores it as this template's program,
// replacing any previously parsed source.
func (t *Template) ParseString(source string) error {
	prog, err := parse(source)
	if err != nil {
		return err
	}
	t.prog = prog
	return nil
}

// RegisterHelper registers (or replaces) a helper by name. It returns the
// template for chaining.
func (t *Template) RegisterHelper(name string, fn Helper) *Template {
	t.helpers[name] = fn
	return t
}

// RegisterPartial compiles and registers a partial template by name. It returns
// an error if the partial source fails to parse.
func (t *Template) RegisterPartial(name, source string) error {
	prog, err := parse(source)
	if err != nil {
		return err
	}
	t.partials[name] = prog
	return nil
}

// Render executes the template against data and returns the produced string.
func (t *Template) Render(data interface{}) (string, error) {
	if t.prog == nil {
		return "", nil
	}
	r := &renderer{tmpl: t, buf: &strings.Builder{}}
	return r.render(t.prog, data)
}

// MustRender is like Render but panics on error.
func (t *Template) MustRender(data interface{}) string {
	out, err := t.Render(data)
	if err != nil {
		panic(err)
	}
	return out
}

// Render is a convenience that parses source and renders it against data in one
// call. For repeated rendering, compile once with Parse and reuse the Template.
func Render(source string, data interface{}) (string, error) {
	t, err := Parse(source)
	if err != nil {
		return "", err
	}
	return t.Render(data)
}
