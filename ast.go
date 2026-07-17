package handlebars

// Node is any element of a parsed template's abstract syntax tree.
type Node interface{ node() }

// Program is a sequence of nodes forming a template or a block body.
type Program struct {
	Body []Node
}

func (*Program) node() {}

// ContentNode is literal template text emitted verbatim.
type ContentNode struct {
	Value string
}

func (*ContentNode) node() {}

// CommentNode is a Handlebars comment; it produces no output.
type CommentNode struct {
	Value string
}

func (*CommentNode) node() {}

// MustacheNode is an interpolation such as {{expr}} or {{{expr}}}.
type MustacheNode struct {
	Expr      *Expr
	Unescaped bool
}

func (*MustacheNode) node() {}

// BlockNode is a block helper invocation such as {{#each items}}...{{/each}}.
//
// Program holds the main body. Inverse holds the {{else}} body (or the body of
// an {{^inverse}} form). Inverted is true when the block was written using the
// {{^name}} shorthand, in which case Program and Inverse are conceptually
// swapped by the parser (body lives in Inverse).
type BlockNode struct {
	Expr     *Expr
	Program  *Program
	Inverse  *Program
	Inverted bool
}

func (*BlockNode) node() {}

// PartialNode is a partial invocation such as {{> name}} or {{> name ctx}}.
type PartialNode struct {
	Name    *Expr // literal name, path, or subexpression (dynamic partial)
	Context *Expr // optional explicit context argument
	Hash    []HashPair
	Indent  string // leading indentation for standalone partials
}

func (*PartialNode) node() {}

// exprKind classifies an Expr.
type exprKind int

const (
	exprPath exprKind = iota // variable path or helper name
	exprString
	exprNumber
	exprBool
	exprSubexpr // (helper args...)
)

// HashPair is a single key=value hash argument.
type HashPair struct {
	Key   string
	Value *Expr
}

// Expr is an expression: a path, a literal, or a helper/subexpression call.
//
// For a mustache like {{helper a b key=c}} the Expr has Kind exprPath, Path set
// to the callee, Params holding the positional arguments and Hash holding the
// named arguments. A plain variable {{foo.bar}} is simply an exprPath with no
// Params or Hash.
type Expr struct {
	Kind   exprKind
	Path   *Path
	Str    string
	Num    float64
	Bool   bool
	Params []*Expr
	Hash   []HashPair
}

// Path is a resolved path expression such as foo.bar.0, ../name, this or @index.
type Path struct {
	Data     bool     // leading @ (data variable such as @index)
	Depth    int      // number of leading ../ segments
	This     bool     // explicit this or . reference
	Segments []string // dotted segments, e.g. ["foo","bar","0"]
	Original string   // source text, for diagnostics
}
