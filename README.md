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

The three most universal analyzers need no configuration beyond enabling them in `.golangci.yml`:

```yaml
version: "2"

linters:
  enable:
    - nospecialunicode
    - nounicodeescape
    - nodeepimports
  settings:
    custom:
      nodeepimports:
        type: "module"
        settings:
          module-root: "github.com/your-org/your-repo"
```

This turns on:

| Analyzer           | What it does                                                        |
| ------------------ | ------------------------------------------------------------------- |
| `nospecialunicode` | Catches smart quotes, invisible spaces, and other unicode impostors |
| `nounicodeescape`  | Prefers `"©"` over `"\u00A9"`                                       |
| `nodeepimports`    | Prevents importing too deep within the same top-level folder        |

The remaining analyzers need explicit configuration:

```yaml
linters:
  enable:
    - nofalsesharing
    - readfriendlyorder
  settings:
    custom:
      nofalsesharing:
        type: "module"
        settings:
          shared-dirs: "pkg/shared,internal/common"
          mode: "dir"
      readfriendlyorder:
        type: "module"
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

### `nodeepimports`

Forbids importing more than one level deeper than the current package within the same top-level folder. If `pkg/auth/login` imports from `pkg/auth/validators/internal/format`, that's reaching too deep into implementation details. This analyzer nudges you toward flatter structures and proper module boundaries.

Only triggers for imports within the same top-level folder. External packages and imports into other top-level folders are ignored.

#### Settings

| Setting       | Type     | Required | Description                                          |
| ------------- | -------- | -------- | ---------------------------------------------------- |
| `module-root` | `string` | yes      | Go module path prefix (e.g. `github.com/org/repo`)  |

#### Examples

Given a module with `module-root: "github.com/org/repo"`:

```go
// Package: github.com/org/repo/pkg/auth

// OK — one level deep, same folder
import "github.com/org/repo/pkg/auth/validators"

// Bad — two levels deep into same top-level folder
import "github.com/org/repo/pkg/auth/validators/internal"
```

### `nofalsesharing`

The "shared" folder anti-pattern detector. LLMs love creating shared utilities that are only used by one consumer — or worse, by nobody at all. This analyzer requires that packages inside your designated shared directories are actually imported by at least two separate entities. If it's only used in one place, it's not shared — it's misplaced.

#### Settings

| Setting       | Type     | Required | Description                                                              |
| ------------- | -------- | -------- | ------------------------------------------------------------------------ |
| `shared-dirs` | `string` | yes      | Comma-separated shared directory paths (e.g. `pkg/shared,internal/common`) |
| `mode`        | `string` | no       | Consumer counting mode: `file` (default) or `dir`                        |
| `module-root` | `string` | no       | Go module path (auto-detected from `go.mod` if not specified)            |

In `file` mode, each importing file counts as a separate consumer. In `dir` mode, each importing package's directory (up to 3 levels deep) counts as one consumer.

#### What it catches

```
pkg/shared/formatdate — only used by: pkg/features/calendar/view.go
  → error: must be used by 2+ entities

pkg/utils/oldhelper — not imported by any entity
  → error: must be used by 2+ entities
```

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

All analyzer tests use the shared plugin E2E harness (`internal/ruletest`), which runs each scenario through the real `custom-gcl` binary against temporary Go workspaces. Scenarios are defined inline in `plugin_test.go` files — no `testdata/` directories or `analysistest` fixtures.

Run repository checks with:

```bash
make test
make lint
```

`make test` builds `custom-gcl` first (if stale), then runs all tests.

## License

[MIT](./LICENSE)
