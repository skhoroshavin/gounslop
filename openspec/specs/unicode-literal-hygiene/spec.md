## Purpose

Define the baseline unicode-literal-hygiene rules for banned special Unicode characters, Unicode escape usage, and safe literal rewrite suggestions.

## Requirements

### Requirement: Special Unicode punctuation and spacing are disallowed in Go literals
The system SHALL report diagnostics for Go string and rune literals that contain any of the currently banned special Unicode punctuation or spacing characters: left and right single or double quotation marks, non-breaking and narrow/figure/punctuation/thin/hair/en/em/medium mathematical/ideographic spaces, zero-width space, zero-width no-break space, en dash, em dash, and horizontal ellipsis.

#### Scenario: Interpreted or raw string contains a banned character
- **WHEN** a Go string literal contains a banned character such as an em dash or non-breaking space
- **THEN** the analyzer reports a diagnostic that identifies the banned character and its code point

#### Scenario: Rune literal contains a banned character
- **WHEN** a Go rune literal contains a banned character such as an em dash
- **THEN** the analyzer reports the same literal-hygiene diagnostic for that rune literal

### Requirement: Safe ASCII replacements are suggested for banned special Unicode characters
The system SHALL suggest replacing an entire literal with ASCII equivalents when every banned character in that literal can be safely rewritten without breaking the literal's Go quoting form.

#### Scenario: Safe replacement is available for a quoted string
- **WHEN** a double-quoted or raw string literal contains a banned character with a safe ASCII replacement
- **THEN** the analyzer suggests rewriting the full literal with the ASCII-equivalent text

### Requirement: Unicode escape sequences are disallowed in interpreted Go literals
The system SHALL report diagnostics when interpreted Go string or rune literals use `\uXXXX` or `\UXXXXXXXX` escape sequences instead of the literal Unicode characters.

#### Scenario: Short or long Unicode escape is used in an interpreted literal
- **WHEN** an interpreted string or rune literal contains a `\uXXXX` or `\UXXXXXXXX` escape sequence
- **THEN** the analyzer reports a diagnostic instructing the author to use the actual character instead

#### Scenario: Raw strings are not checked for Unicode escapes
- **WHEN** a raw string contains the text `\u2014` or another escape-like sequence literally
- **THEN** the analyzer does not report a Unicode-escape diagnostic for that raw string

### Requirement: Unicode escape fixes require fully safe inlining
The system SHALL suggest replacing Unicode escape sequences with literal characters only when every matching escape in the literal can be inlined safely. Control characters, literal delimiters, backslashes, and Unicode format characters SHALL prevent the fix while still allowing the diagnostic.

#### Scenario: Safe Unicode escape can be inlined
- **WHEN** an interpreted literal contains only safely inlineable Unicode escapes such as `\u2014`
- **THEN** the analyzer suggests rewriting the literal with the actual Unicode character

#### Scenario: Mixed safe and unsafe escapes remain unfixed
- **WHEN** an interpreted literal contains at least one Unicode escape that cannot be safely inlined
- **THEN** the analyzer still reports the diagnostic and omits the suggested fix for that literal
