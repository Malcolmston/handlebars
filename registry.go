package handlebars

import "sort"

// RegisterHelpers registers (or replaces) several helpers at once. It mirrors
// the object form of Handlebars.js registerHelper, where a map of name→function
// is registered in a single call. Registering a nil function removes any helper
// previously registered under that name. It returns the template for chaining.
func (t *Template) RegisterHelpers(helpers map[string]Helper) *Template {
	for name, fn := range helpers {
		if fn == nil {
			delete(t.helpers, name)
			continue
		}
		t.helpers[name] = fn
	}
	return t
}

// RegisterPartials compiles and registers several partials at once, mirroring
// the object form of Handlebars.js registerPartial. Each value is a partial
// source. If any source fails to parse the corresponding partial is skipped and
// the first such error is returned; all parseable partials are still registered.
func (t *Template) RegisterPartials(partials map[string]string) error {
	// Register in a deterministic order so the returned error is stable.
	names := make([]string, 0, len(partials))
	for name := range partials {
		names = append(names, name)
	}
	sort.Strings(names)
	var firstErr error
	for _, name := range names {
		if err := t.RegisterPartial(name, partials[name]); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// UnregisterHelper removes the named helper, mirroring Handlebars.js
// unregisterHelper. Removing an absent name is a no-op. It returns the template
// for chaining.
func (t *Template) UnregisterHelper(name string) *Template {
	delete(t.helpers, name)
	return t
}

// UnregisterPartial removes the named partial, mirroring Handlebars.js
// unregisterPartial. Removing an absent name is a no-op. It returns the template
// for chaining.
func (t *Template) UnregisterPartial(name string) *Template {
	delete(t.partials, name)
	return t
}

// UnregisterDecorator removes the named decorator. It is the symmetric
// counterpart to RegisterDecorator. Removing an absent name is a no-op. It
// returns the template for chaining.
func (t *Template) UnregisterDecorator(name string) *Template {
	delete(t.decorators, name)
	return t
}

// HasHelper reports whether a helper is registered under name, including the
// built-in inline helpers.
func (t *Template) HasHelper(name string) bool {
	_, ok := t.helpers[name]
	return ok
}

// HasPartial reports whether a partial is registered under name. It does not
// account for inline partials, which exist only during rendering.
func (t *Template) HasPartial(name string) bool {
	_, ok := t.partials[name]
	return ok
}

// HelperNames returns the names of all registered helpers (built-in and custom)
// in sorted order, so the result is deterministic.
func (t *Template) HelperNames() []string {
	names := make([]string, 0, len(t.helpers))
	for name := range t.helpers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// PartialNames returns the names of all registered partials in sorted order, so
// the result is deterministic. Inline partials are not included.
func (t *Template) PartialNames() []string {
	names := make([]string, 0, len(t.partials))
	for name := range t.partials {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Clone returns an independent copy of the template: its compiled program,
// compile options, logger and its helper, partial and decorator registries are
// duplicated so that subsequent registrations or unregistrations on either the
// original or the clone do not affect the other. It mirrors the isolated
// environment produced by Handlebars.create in Handlebars.js and is useful for
// deriving a specialised template from a shared base without mutating it.
func (t *Template) Clone() *Template {
	nc := t.cfg
	nc.knownHelpers = make(map[string]bool, len(t.cfg.knownHelpers))
	for k, v := range t.cfg.knownHelpers {
		nc.knownHelpers[k] = v
	}
	clone := &Template{
		prog:       t.prog,
		helpers:    make(map[string]Helper, len(t.helpers)),
		partials:   make(map[string]*Program, len(t.partials)),
		decorators: make(map[string]Decorator, len(t.decorators)),
		cfg:        nc,
		logger:     t.logger,
	}
	for k, v := range t.helpers {
		clone.helpers[k] = v
	}
	for k, v := range t.partials {
		clone.partials[k] = v
	}
	for k, v := range t.decorators {
		clone.decorators[k] = v
	}
	return clone
}
