package handlebars_test

import (
	"fmt"
	"strings"

	"github.com/malcolmston/handlebars"
)

// Example demonstrates interpolation, escaping, blocks, each with data
// variables, a custom helper and a partial.
func Example() {
	t := handlebars.New()

	// A custom inline helper returning a SafeString (not re-escaped).
	t.RegisterHelper("shout", func(o *handlebars.Options) interface{} {
		return handlebars.SafeString(strings.ToUpper(fmt.Sprint(o.Arg(0))) + "!")
	})

	// A partial reused inside an #each loop.
	if err := t.RegisterPartial("row", "{{@index}}. {{name}} ({{team}})\n"); err != nil {
		panic(err)
	}

	const src = `Report for {{shout title}}
Owner: {{owner}}
{{#each people}}
{{> row team=../team}}
{{else}}
No people.
{{/each}}
{{#if count}}Total: {{count}}{{else}}Empty{{/if}}`

	if err := t.ParseString(src); err != nil {
		panic(err)
	}

	out, err := t.Render(map[string]interface{}{
		"title": "q3",
		"owner": "A&B",
		"team":  "Blue",
		"count": 2,
		"people": []map[string]interface{}{
			{"name": "Ada"},
			{"name": "Lin"},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(out)

	// Output:
	// Report for Q3!
	// Owner: A&amp;B
	// 0. Ada (Blue)
	// 1. Lin (Blue)
	// Total: 2
}

// Example_advanced demonstrates block parameters, an {{else if}} chain, inline
// partials and a partial block with @partial-block.
func Example_advanced() {
	const src = `{{#*inline "badge"}}[{{label}}]{{/inline}}` +
		`{{#each users as |user index|}}` +
		`{{index}}:{{user.name}} ` +
		`{{#if (gt user.score 90)}}A{{else if (gt user.score 80)}}B{{else}}C{{/if}} ` +
		`{{> badge label=user.role}}` + "\n" +
		`{{/each}}`

	out, err := handlebars.Render(src, map[string]interface{}{
		"users": []map[string]interface{}{
			{"name": "Ada", "score": 95, "role": "lead"},
			{"name": "Lin", "score": 84, "role": "dev"},
			{"name": "Sam", "score": 70, "role": "intern"},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Print(out)

	// Output:
	// 0:Ada A [lead]
	// 1:Lin B [dev]
	// 2:Sam C [intern]
}

// Example_partialBlock shows a layout partial wrapping caller-supplied content
// via @partial-block.
func Example_partialBlock() {
	t := handlebars.New()
	if err := t.RegisterPartial("layout", "<main>{{> @partial-block}}</main>"); err != nil {
		panic(err)
	}
	if err := t.ParseString("{{#> layout}}Hello {{name}}{{/layout}}"); err != nil {
		panic(err)
	}
	fmt.Println(t.MustRender(map[string]interface{}{"name": "World"}))

	// Output:
	// <main>Hello World</main>
}
