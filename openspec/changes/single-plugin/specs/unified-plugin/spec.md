## ADDED Requirements

### Requirement: Single plugin registration replaces per-analyzer registrations
The plugin module SHALL register exactly one plugin named `gounslop` via `register.Plugin`. The four previous registrations (`boundarycontrol`, `nospecialunicode`, `nounicodeescape`, `readfriendlyorder`) SHALL be removed.

#### Scenario: Plugin registers under the gounslop name
- **WHEN** the plugin `init()` function executes
- **THEN** exactly one `register.Plugin("gounslop", ...)` call is made and no other `register.Plugin` calls are made

### Requirement: All analyzers are enabled by default
The plugin constructor SHALL include all four analyzers (`boundarycontrol`, `nospecialunicode`, `nounicodeescape`, `readfriendlyorder`) in the `BuildAnalyzers` return slice when no `disable` list is provided in settings.

#### Scenario: No settings produces all analyzers
- **WHEN** the plugin is constructed with nil or empty settings
- **THEN** `BuildAnalyzers` returns all four analyzers

#### Scenario: Empty disable list produces all analyzers
- **WHEN** the plugin is constructed with `{"disable": []}`
- **THEN** `BuildAnalyzers` returns all four analyzers

### Requirement: Analyzers can be disabled by name
The plugin settings SHALL accept a `disable` field containing a list of analyzer names. Analyzers whose names appear in the `disable` list SHALL be omitted from the `BuildAnalyzers` return slice.

#### Scenario: Single analyzer disabled
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicode"]}`
- **THEN** `BuildAnalyzers` returns three analyzers and `nospecialunicode` is not among them

#### Scenario: Multiple analyzers disabled
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicode", "nounicodeescape"]}`
- **THEN** `BuildAnalyzers` returns two analyzers and neither `nospecialunicode` nor `nounicodeescape` is among them

### Requirement: Unknown disable entries produce a startup error
The plugin constructor SHALL validate that every entry in the `disable` list matches a known analyzer name. If an unknown name is present, the constructor SHALL return an error.

#### Scenario: Typo in disable list fails at startup
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicod"]}`
- **THEN** the constructor returns an error mentioning the unknown analyzer name

### Requirement: Boundarycontrol architecture settings use flat top-level key
The plugin settings SHALL accept an `architecture` field at the top level of the settings map. When present, the plugin constructor SHALL decode and validate the architecture configuration and configure the `boundarycontrol` analyzer's flags before including it in the `BuildAnalyzers` return slice.

#### Scenario: Architecture settings are applied to boundarycontrol
- **WHEN** the plugin is constructed with `{"architecture": {"pkg/*": {"imports": ["internal/*"]}}}`
- **THEN** the `boundarycontrol` analyzer's `architecture` flag is set to the JSON-encoded architecture map

#### Scenario: Invalid architecture settings produce a startup error
- **WHEN** the plugin is constructed with `{"architecture": {"pkg/*": {"imports": "not-a-list"}}}`
- **THEN** the constructor returns an error describing the invalid architecture settings

#### Scenario: Architecture settings are ignored when boundarycontrol is disabled
- **WHEN** the plugin is constructed with `{"disable": ["boundarycontrol"], "architecture": {"pkg/*": {"imports": ["internal/*"]}}}`
- **THEN** the constructor succeeds and `BuildAnalyzers` omits `boundarycontrol`

### Requirement: Plugin load mode is always LoadModeTypesInfo
The plugin's `GetLoadMode` method SHALL return `LoadModeTypesInfo` unconditionally, regardless of which analyzers are enabled.

#### Scenario: Load mode with all analyzers enabled
- **WHEN** `GetLoadMode` is called on a plugin with all analyzers enabled
- **THEN** it returns `LoadModeTypesInfo`

#### Scenario: Load mode with only syntax analyzers enabled
- **WHEN** `GetLoadMode` is called on a plugin where `boundarycontrol` and `readfriendlyorder` are disabled
- **THEN** it still returns `LoadModeTypesInfo`

### Requirement: Each analyzer retains its own name for nolint granularity
Each `*analysis.Analyzer` returned by `BuildAnalyzers` SHALL have its `Name` field set to the analyzer's individual name (e.g. `boundarycontrol`, `nospecialunicode`), not to `gounslop`.

#### Scenario: Analyzer names are preserved
- **WHEN** `BuildAnalyzers` is called with no disabled analyzers
- **THEN** the returned analyzers have names `boundarycontrol`, `nospecialunicode`, `nounicodeescape`, and `readfriendlyorder`
