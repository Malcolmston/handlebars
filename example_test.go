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
