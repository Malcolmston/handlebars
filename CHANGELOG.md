# Changelog

All notable changes to this project are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/) and this project adheres to
semantic versioning.

## [0.3.0] - 2026-07-18

Further Handlebars.js parity, focused on the runtime registry and the `Utils`
namespace. All additions are backward compatible and standard-library-only.

### Added

- **Registry management** mirroring the Handlebars.js environment API:
  `Template.RegisterHelpers` and `Template.RegisterPartials` (bulk registration
  from a map, matching the object form of `registerHelper`/`registerPartial`),
  `Template.UnregisterHelper`, `Template.UnregisterPartial` and
  `Template.UnregisterDecorator`, plus `Template.HasHelper`,
  `Template.HasPartial`, `Template.HelperNames` and `Template.PartialNames`
  (sorted, deterministic).
- **`Template.Clone`**: an isolated deep copy of a template's program, options,
  logger and registries, mirroring `Handlebars.create`, so a specialised
  template can be derived from a shared base without mutating it.
- **`Utils` parity functions**: `EscapeExpression` (SafeString-aware escaping of
  any value, matching `Utils.escapeExpression`), `Stringify` (the engine's
  canonical value stringification), `IsEmpty` (matching `Utils.isEmpty`,
  including the empty-map-is-not-empty rule), `IsArray`, `Extend` and
  `CreateFrame`.

## [0.2.0] - 2026-07-17

Major push toward Handlebars.js parity. All additions are backward compatible.

### Added

- **Block parameters**: `{{#each items as |item index|}}` and
  `{{#with obj as |o|}}`, lexically scoped and shadowing the context. Custom
  block helpers receive them via `Options.FnWithBlockParams` and
  `Options.BlockParams`.
- **`{{else if}}` / `{{else unless}}` / `{{else with}}` chaining** for
  conditionals.
- **Inline partials** `{{#*inline "name"}}...{{/inline}}` and **partial blocks**
  `{{#> layout}}...{{/layout}}` exposing `{{> @partial-block}}`, with fallback
  to the block body when the partial is undefined.
- **Decorators**: `RegisterDecorator`, the `{{* name}}` and `{{#*name}}...{{/name}}`
  syntax, and a built-in `inline` decorator that implements inline partials.
- **Data variables** `@level` and parent-frame access `@../index`, alongside the
  existing `@root`, `@index`, `@key`, `@first`, `@last`.
- **Helper-missing hooks**: register `helperMissing` and `blockHelperMissing` to
  intercept unknown helpers and block helpers.
- **Built-in `log` helper** (`{{log msg level="info"}}`) with a configurable
  logger via `Template.SetLogger`.
- **Compile options**: `Compile(src, ...Option)` with `NoEscape`, `Strict`,
  `NoData`, `KnownHelpers` and `KnownHelpersOnly`.

### Changed

- Standalone-line whitespace stripping now covers inline partials, partial
  blocks and decorators.
- Unknown bare `{{foo}}` remains empty; `{{foo arg}}` errors (or invokes a
  registered `helperMissing`), matching Handlebars.js.

## [0.1.0]

Initial release: lexer, parser, reflection renderer; interpolation and escaping;
paths (`this`, `../`, `@root`, indices, bracketed segments); `if`/`unless`/`each`/
`with` blocks and inverse sections; inline comparison helpers; custom helpers with
hash args and `SafeString`; partials with context, hash overlays and dynamic
names; comments; and `{{~ ~}}` whitespace control.
