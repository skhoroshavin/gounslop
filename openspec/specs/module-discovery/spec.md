# Purpose

TBD — Module context discovery, import classification, and caching for Go module-based analyzers.

## Requirements

### Requirement: Module discovery finds the nearest go.mod for any package file
The system SHALL discover the nearest enclosing `go.mod` file for any analyzed package by walking up the directory tree from the package's first source file. It SHALL parse the `module` directive from that `go.mod` to determine the module path. It SHALL also discover any nested modules (other `go.mod` files) within the same directory tree and record their module paths.

#### Scenario: Nearest go.mod defines module scope
- **WHEN** a file under `tools/internal/checker` is analyzed and the nearest enclosing `go.mod` declares `module example.com/root/tools`
- **THEN** the system uses `example.com/root/tools` as the module scope for that package

#### Scenario: Missing go.mod fails clearly
- **WHEN** an analyzed package has no enclosing `go.mod`
- **THEN** the system returns a clear error that module scope could not be discovered

### Requirement: Module discovery classifies import paths relative to the module
The system SHALL classify any import path relative to a discovered module into one of three categories: `OutsideModule`, `CurrentModule`, or `NestedModule`. An import path outside the module path prefix is `OutsideModule`. An import path matching the module path or a descendant of it is `CurrentModule`, unless it matches a nested module path (exact or prefix), in which case it is `NestedModule`.

#### Scenario: External import is outside module
- **WHEN** the module path is `example.com/mod` and the import path is `github.com/external/lib`
- **THEN** the classification is `OutsideModule`

#### Scenario: Import within current module
- **WHEN** the module path is `example.com/mod` and the import path is `example.com/mod/feature/api`
- **THEN** the classification is `CurrentModule` with relative path `feature/api`

#### Scenario: Import within nested module
- **WHEN** the module path is `example.com/root` and the import path is `example.com/root/tools/pkg` where `tools/go.mod` declares `module example.com/root/tools`
- **THEN** the classification is `NestedModule`

### Requirement: Module discovery caches results per module directory
The system SHALL cache discovered module information by module directory to avoid repeated filesystem traversal and `go.mod` parsing across multiple analyzed packages in the same module.

#### Scenario: Multiple packages in same module use cached discovery
- **WHEN** two packages in the same module are analyzed sequentially
- **THEN** the second analysis reuses the cached module information without re-reading `go.mod`
