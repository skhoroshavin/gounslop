# gounslop

Go static analysis linters that catch common LLM-generated code smells — the kind of subtle junk that sneaks in when your LLM is feeling creative. Smart quotes, invisible unicode, spaghetti imports, dead "shared" code that nobody shares, and declarations ordered for machines instead of humans.

Distributed as a [golangci-lint v2](https://golangci-lint.run/) module plugin. Requires golangci-lint v2.11+.

## Installation

Add the plugin to your `.custom-gcl.yml`:

```yaml
version: v2.11.4
name: custom-gcl
destination: .
plugins:
  - module: 'github.com/skhoroshavin/gounslop'
    import: 'github.com/skhoroshavin/gounslop/plugin'
```

Then build and run:

```bash
golangci-lint custom
./custom-gcl run ./...
```

## Quick Start

The two most universal analyzers need no configuration beyond enabling them in `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - nospecialunicode
    - nounicodeescape
```

This turns on:

| Analyzer           | What it does                                                        |
| ------------------ | ------------------------------------------------------------------- |
| `nospecialunicode` | Catches smart quotes, invisible spaces, and other unicode impostors |
| `nounicodeescape`  | Prefers `"©"` over `"\u00A9"`                                       |

The remaining analyzers need explicit configuration:

```yaml
linters:
  enable:
    - gounslop
  settings:
    custom:
      gounslop:
        type: "module"
        settings:
          architecture:
            "pkg/shared":
              shared: true
            ".":
              imports: ["pkg/+", "internal/*", "pkg/shared"]
            "pkg/repository/*":
              imports: ["pkg/models/+", "pkg/utils"]
```

## Analyzers

### `nospecialunicode`

Disallows special unicode punctuation and whitespace characters in string and character literals. LLMs love to sprinkle in smart quotes, non-breaking spaces, and other invisible gremlins that look fine in a PR review but cause fun bugs at runtime.

Caught characters include: left/right smart quotes, non-breaking space, en/em dash, horizontal ellipsis, zero-width space, and various other exotic whitespace.

```go
// Bad — these contain invisible special characters that look normal
msg := "Hello World"     // a non-breaking space (U+00A0) is hiding between the words
quote := "He said "hi""  // smart double quotes (U+201C, U+201D)

// Good
msg := "Hello World"     // regular ASCII space
quote := "He said \"hi\""  // plain ASCII quotes
```

Note: the bad examples above contain actual unicode characters that may be indistinguishable from their ASCII counterparts in your font — that's exactly the problem this analyzer catches.

### `nounicodeescape`

Prefers actual characters over `\uXXXX` escape sequences. If your string says `\u00A9`, just write `©` — your coworkers will thank you. LLM-generated code sometimes encodes characters as escape sequences for no good reason.

```go
// Bad
copyright := "\u00A9 2025"
arrow := "\u2192"

// Good
copyright := "© 2025"
arrow := "→"
```

### `importcontrol`

Enforces selector-based package import boundaries inside the nearest enclosing Go module. `importcontrol` auto-discovers module scope from `go.mod`, ignores standard-library and third-party imports outside that module, excludes nested-module imports owned by a deeper `go.mod`, and enforces same-scope deep-import restrictions.

### `exportcontrol`

Enforces export contract patterns for top-level declarations. When a package selector has `exports` patterns, only exported symbols matching those regex patterns are allowed.

### `nofalsesharing`

Detects exported symbols in shared packages that are not actually used by 2+ entities. A package marked with `shared: true` should share its symbols broadly; if a symbol is only used by one consumer (or not at all), it should either be unexported or moved.

#### Settings

| Setting | Type | Required | Description |
| ------- | ---- | -------- | ----------- |
| `architecture` | `map[string]object` | yes | Boundary, export-contract, and shared-package policy keyed by package selector |

Each architecture entry has this shape:

```yaml
"pkg/repository/*":
  imports: ["pkg/models/+", "pkg/utils"]

"pkg/api":
  exports: ["^New[A-Z].*$", "^[A-Z][A-Za-z0-9]*Error$"]

"pkg/shared":
  shared: true
```

Supported key selector forms:

| Selector | Matches |
| -------- | ------- |
| `.` | The module root package only |
| `pkg/models` | That package and all descendants |
| `pkg/repository/*` | Each direct child subtree under `pkg/repository`, but not `pkg/repository` itself |

Supported `imports` selector forms:

| Selector | Matches |
| -------- | ------- |
| `pkg/models` | That exact package only |
| `pkg/models/*` | Direct child packages only |
| `pkg/models/+` | The package itself and its direct children |

#### Example

```yaml
linters:
  enable:
    - gounslop
  settings:
    custom:
      gounslop:
        type: "module"
        settings:
          architecture:
            "pkg/shared":
              shared: true
            ".":
              imports: ["pkg/+", "internal/*", "pkg/shared"]
            "pkg/api":
              imports: ["pkg/contracts", "pkg/shared/+", "internal/http/*"]
              exports: ["^New[A-Z].*$", "^[A-Z][A-Za-z0-9]*Error$"]
            "pkg/repository/*":
              imports: ["pkg/models/+", "pkg/utils"]
```

Notes:

- Unmatched in-module packages behave as `imports: []`.
- Imports from a package to its immediate child package stay allowed even without an explicit rule.
- Same-scope deep imports are rejected directly by `importcontrol`.
- Module scope comes from the nearest enclosing `go.mod`; nested modules are treated as out of scope for the parent module.
- A selector with `exports` enables export-contract checks for exported top-level package declarations owned by that selector.
- Export-contract patterns use full-name regex matching; exported methods are excluded from this check.
- A selector with `shared: true` marks its owned package subtree for exported-symbol-level false-sharing checks.
- Shared-package consumers are counted per exported symbol by direct non-test importing package path, with same-package references also counting as a consumer for that symbol.

### `readfriendlyorder`

Enforces a top-down reading order for your code. The idea: when someone opens a file, they should see the important stuff first and the helpers below. LLM-generated code often scatters declarations in random order, making files harder to follow.

This analyzer covers three areas:

**Top-level ordering** — Exported symbols should come before the private helpers they use. Read the API first, implementation details second.

```go
// Bad — helper defined before its consumer
func formatName(name string) string {
    return strings.TrimSpace(strings.ToLower(name))
}

func CreateUser(name string) User {
    return User{Name: formatName(name)}
}

// Good — consumer first, helper below
func CreateUser(name string) User {
    return User{Name: formatName(name)}
}

func formatName(name string) string {
    return strings.TrimSpace(strings.ToLower(name))
}
```

**Init function placement** — `init()` functions should appear before other functions for visibility.

**Method dependency ordering** — Methods on a type should follow dependency order.

**Test file ordering** — `TestMain` should appear first in test files.

## A Note on Provenance

Yes, a fair amount of this was vibe-coded with LLM assistance — which is fitting, since that's exactly the context these linters are designed for. That said, the ideas behind these analyzers, the decisions about what to catch and how to catch it, and the overall design are mine. Every piece of code went through human review, and the test cases in particular were written and verified with deliberate care.

The project also dogfoods itself: `gounslop` is linted using `gounslop`.

## Contributing

See [AGENTS.md](./AGENTS.md) for development setup and guidelines.

### Test Strategy

All analyzer tests use the shared plugin E2E harness (`tests/rule`), which runs each scenario through the real `custom-gcl` binary against temporary Go workspaces. Scenarios are defined inline in `tests/<analyzer>_test.go` files — no `testdata/` directories or `analysistest` fixtures.

Run repository checks with:

```bash
make test
make lint
```

`make test` builds `custom-gcl` first (if stale), then runs all tests.

## License

[MIT](./LICENSE)
