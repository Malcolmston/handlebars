# handlebars

Logic-less templating engine for Go — a dependency-free Handlebars/Mustache
implementation using only the standard library. It ships its own lexer, parser
and reflection-based renderer; it does **not** wrap `text/template`.

## Install

```sh
go get github.com/malcolmston/handlebars
```

Requires Go 1.24+. No third-party dependencies.

## Quick start

```go
package main

import (
	"fmt"

	"github.com/malcolmston/handlebars"
)

func main() {
	out, err := handlebars.Render("Hello {{name}}!", map[string]any{"name": "World"})
	if err != nil {
		panic(err)
	}
	fmt.Println(out) // Hello World!
}
```

Compile once and reuse for repeated rendering:

```go
t := handlebars.MustParse("{{#each items}}{{@index}}:{{this}} {{/each}}")
fmt.Println(t.MustRender(map[string]any{"items": []string{"a", "b"}}))
// 0:a 1:b
```

## Features

- **Interpolation**: `{{expr}}` (HTML-escaped) and `{{{expr}}}` / `{{& expr}}` (raw).
- **Paths**: `foo`, `foo.bar.baz`, `foo.0` (index), `this` / `.`, `../parent`,
  `@root`, and bracketed segments `[a b]`.
- **Block helpers**: `{{#if}}` / `{{else}}` / `{{/if}}`, `{{#unless}}`,
  `{{#each}}` (slices **and** maps, with `@index`, `@key`, `@first`, `@last`,
  `this`), `{{#with}}`, and inverse sections `{{^cond}}...{{/cond}}`.
- **Inline helpers**: `lookup`, `eq`, `ne`, `not`, `and`, `or`, `gt`, `lt`,
  `gte`, `lte` — usable in subexpressions such as `{{#if (eq a b)}}`.
- **Custom helpers**: `RegisterHelper(name, func(*Options) any)`, including block
  helpers (via `o.Fn` / `o.Inverse`), hash arguments (`{{h key=value}}`), and
  `SafeString` to opt out of escaping.
- **Partials**: `RegisterPartial(name, src)` with `{{> name}}`, explicit context
  `{{> name ctx}}`, hash overlays `{{> name k=v}}`, and dynamic `{{> (expr)}}`.
- **Comments**: `{{! ... }}` and `{{!-- ... --}}`.
- **Whitespace control**: standalone-line stripping plus explicit `{{~ ~}}`.
- **Data model**: maps, structs (field name, `json` tag, then case-insensitive),
  slices, pointers. Map iteration is sorted by key for deterministic output.

## Custom helpers

```go
t := handlebars.New()

t.RegisterHelper("shout", func(o *handlebars.Options) any {
	return handlebars.SafeString("<em>" + fmt.Sprint(o.Arg(0)) + "!</em>")
})

t.RegisterHelper("bold", func(o *handlebars.Options) any { // block helper
	return "<b>" + o.Fn(o.Arg(0)) + "</b>"
})

_ = t.ParseString("{{shout name}} {{#bold user}}{{name}}{{/bold}}")
```

## Partials

```go
t := handlebars.New()
_ = t.RegisterPartial("row", "{{@index}}. {{name}}\n")
_ = t.ParseString("{{#each people}}{{> row}}{{/each}}")
```

See the package documentation (`go doc github.com/malcolmston/handlebars`) for the
full reference.

## License

See repository.
