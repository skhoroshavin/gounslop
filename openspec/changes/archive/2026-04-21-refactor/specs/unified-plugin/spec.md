## MODIFIED Requirements

### Requirement: All analyzers are enabled by default
The plugin constructor SHALL include all six analyzers (`importcontrol`, `exportcontrol`, `nofalsesharing`, `readfriendlyorder`, `nospecialunicode`, `nounicodeescape`) in the `BuildAnalyzers` return slice when no `disable` list is provided in settings.

#### Scenario: No settings produces all analyzers
- **WHEN** the plugin is constructed with nil or empty settings
- **THEN** `BuildAnalyzers` returns all six analyzers

#### Scenario: Empty disable list produces all analyzers
- **WHEN** the plugin is constructed with `{"disable": []}`
- **THEN** `BuildAnalyzers` returns all six analyzers

### Requirement: Analyzers can be disabled by name
The plugin settings SHALL accept a `disable` field containing a list of analyzer names. Analyzers whose names appear in the `disable` list SHALL be omitted from the `BuildAnalyzers` return slice.

#### Scenario: Single analyzer disabled
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicode"]}`
- **THEN** `BuildAnalyzers` returns five analyzers and `nospecialunicode` is not among them

#### Scenario: Multiple analyzers disabled
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicode", "nounicodeescape"]}`
- **THEN** `BuildAnalyzers` returns four analyzers and neither `nospecialunicode` nor `nounicodeescape` is among them

#### Scenario: Boundarycontrol-derived analyzers disabled individually
- **WHEN** the plugin is constructed with `{"disable": ["importcontrol"]}`
- **THEN** `BuildAnalyzers` returns five analyzers and `importcontrol` is not among them

### Requirement: Unknown disable entries produce a startup error
The plugin constructor SHALL validate that every entry in the `disable` list matches a known analyzer name. If an unknown name is present, the constructor SHALL return an error.

#### Scenario: Typo in disable list fails at startup
- **WHEN** the plugin is constructed with `{"disable": ["nospecialunicod"]}`
- **THEN** the constructor returns an error mentioning the unknown analyzer name

### Requirement: Architecture settings use flat top-level key
The plugin settings SHALL accept an `architecture` field at the top level of the settings map. When present, the plugin constructor SHALL decode and validate the architecture configuration and configure the `importcontrol`, `exportcontrol`, and `nofalsesharing` analyzers' flags before including them in the `BuildAnalyzers` return slice.

#### Scenario: Architecture settings are applied to boundarycontrol-derived analyzers
- **WHEN** the plugin is constructed with `{"architecture": {"pkg/*": {"imports": ["internal/*"]}}}`
- **THEN** the `importcontrol`, `exportcontrol`, and `nofalsesharing` analyzers' `architecture` flags are set to the JSON-encoded architecture map

#### Scenario: Invalid architecture settings produce a startup error
- **WHEN** the plugin is constructed with `{"architecture": {"pkg/*": {"imports": "not-a-list"}}}`
- **THEN** the constructor returns an error describing the invalid architecture settings

#### Scenario: Architecture settings are ignored when all boundarycontrol-derived analyzers are disabled
- **WHEN** the plugin is constructed with `{"disable": ["importcontrol", "exportcontrol", "nofalsesharing"], "architecture": {"pkg/*": {"imports": ["internal/*"]}}}`
- **THEN** the constructor succeeds and `BuildAnalyzers` omits `importcontrol`, `exportcontrol`, and `nofalsesharing`

### Requirement: Each analyzer retains its own name for nolint granularity
Each `*analysis.Analyzer` returned by `BuildAnalyzers` SHALL have its `Name` field set to the analyzer's individual name (e.g. `importcontrol`, `nospecialunicode`), not to `gounslop`.

#### Scenario: Analyzer names are preserved
- **WHEN** `BuildAnalyzers` is called with no disabled analyzers
- **THEN** the returned analyzers have names `importcontrol`, `exportcontrol`, `nofalsesharing`, `readfriendlyorder`, `nospecialunicode`, and `nounicodeescape`
