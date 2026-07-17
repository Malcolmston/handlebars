// Library content for the handlebars documentation site. Mirrors the shape used
// by the malcolmston/go landing site's data.ts so the sibling sites stay in sync.
export interface Lib {
  id: string; name: string; icon: string; accent: string; pkg: string; node: string;
  repo: string; docs: string; tagline: string; blurb: string; tags: string[];
  features: string[]; node_code: string; go_code: string; integrate: string;
}

export const NODE_ACCENT = '#8cc84b';

export const HANDLEBARS: Lib = {
  id:"handlebars", name:"Handlebars", icon:'<i class="fa-solid fa-code"></i>', accent:"#f0772b",
  pkg:"github.com/malcolmston/handlebars", node:"handlebars-lang/handlebars.js",
  repo:"https://github.com/malcolmston/handlebars", docs:"https://malcolmston.github.io/handlebars/",
  tagline:"Logic-less Handlebars/Mustache templating in pure Go.",
  blurb:"A dependency-free Handlebars/Mustache templating engine implemented entirely in Go's standard library. "+
    "It ships its own lexer, parser and AST, then renders that tree against any value using reflection — it does "+
    "not wrap text/template, so the Handlebars semantics are implemented directly and match Handlebars.js. "+
    "You get escaped and raw interpolation, rich path expressions with ../parent and @root, the full set of "+
    "built-in block helpers, custom helpers with hash arguments and subexpressions, partials, comments and "+
    "whitespace control — over maps, structs, slices and pointers.",
  tags:["{{mustache}}","HTML escaping","path expressions","block helpers","{{#each}}","custom helpers","partials","zero deps"],
  features:[
    "Interpolation — <code>{{expr}}</code> HTML-escapes; <code>{{{expr}}}</code> and <code>{{&amp; expr}}</code> emit raw, matching the Handlebars.js escape set",
    "Path expressions — <code>foo.bar.baz</code>, index <code>foo.0</code>, <code>this</code>/<code>.</code>, parent <code>../foo</code>, <code>@root</code> and bracketed <code>[a b]</code> segments",
    "Block helpers — <code>{{#if}}</code>/<code>{{else}}</code>, <code>{{#unless}}</code>, <code>{{#each}}</code> and <code>{{#with}}</code>, plus inverse sections <code>{{^cond}}</code>",
    "<code>{{#each}}</code> over slices <b>and</b> maps, exposing <code>@index</code>, <code>@key</code>, <code>@first</code>, <code>@last</code> with map keys visited in sorted order",
    "Built-in inline helpers — <code>lookup</code>, <code>eq</code>, <code>ne</code>, <code>not</code>, <code>and</code>, <code>or</code>, <code>gt</code>, <code>lt</code>, <code>gte</code>, <code>lte</code> — usable in subexpressions like <code>{{#if (eq a b)}}</code>",
    "Custom helpers via <code>RegisterHelper</code> — positional <code>Options.Arg</code>, hash args <code>Options.HashStr</code>, block callbacks <code>Options.Fn</code>/<code>Options.Inverse</code>, and <code>SafeString</code> to bypass escaping",
    "Partials via <code>RegisterPartial</code> — <code>{{&gt; name}}</code>, explicit context <code>{{&gt; name ctx}}</code>, hash overlays and dynamic <code>{{&gt; (expr)}}</code>, with indentation preserved",
    "Comments <code>{{! ... }}</code>/<code>{{!-- ... --}}</code>, standalone-line stripping and explicit whitespace control with <code>{{~ ~}}</code> — all with zero third-party dependencies"
  ],
  node_code:
`const Handlebars = require("handlebars");

const tmpl = Handlebars.compile(
  "{{#each items}}{{@index}}:{{this}} {{/each}}"
);
console.log(tmpl({ items: ["a", "b"] }));
// 0:a 1:b`,
  go_code:
`import "github.com/malcolmston/handlebars"

// One-shot render, or compile once with Parse/MustParse and reuse.
t := handlebars.MustParse("{{#each items}}{{@index}}:{{this}} {{/each}}")
out := t.MustRender(map[string]any{"items": []string{"a", "b"}})
// out == "0:a 1:b "`,
  integrate:
`t := handlebars.New()

<span class="tok-c">// A custom inline helper returning a SafeString, so its markup</span>
<span class="tok-c">// is not re-escaped by the {{ }} interpolation.</span>
t.RegisterHelper("shout", func(o *handlebars.Options) any {
	return handlebars.SafeString("<em>" + fmt.Sprint(o.Arg(0)) + "!</em>")
})

<span class="tok-c">// A block helper wraps its rendered body via o.Fn.</span>
t.RegisterHelper("bold", func(o *handlebars.Options) any {
	return "<b>" + o.Fn(o.Arg(0)) + "</b>"
})

<span class="tok-c">// A partial reused inside an {{#each}} loop, with @index and</span>
<span class="tok-c">// a value pulled from the parent context via ../team.</span>
_ = t.RegisterPartial("row", "{{@index}}. {{name}} ({{team}})\\n")
_ = t.ParseString("{{shout title}}\\n{{#each people}}{{> row team=../team}}{{/each}}")

out, _ := t.Render(map[string]any{
	"title":  "team",
	"team":   "Blue",
	"people": []map[string]any{{"name": "Ada"}, {"name": "Lin"}},
})`
};
